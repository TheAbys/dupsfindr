package main

import (
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

func main() {
	start := time.Now()

	directory := flag.String("directory", "", "the absolute directorypath to search for duplicates")
	flag.Parse()

	if len(*directory) == 0 {
		panic("no directory provided")
	}

	fmt.Println("DupsFindr started with: ", *directory)

	files := make(chan string, 10)
	filesWithoutDuplicates := make([]string, 0, 5)

	var wg sync.WaitGroup
	// read the directory and send all files to the channel
	go readDirectory(*directory, files, &wg)

	counter := 0
	// recieve from files and add to slice
	go readFiles(files, &filesWithoutDuplicates, &counter, &wg)

	wg.Wait()
	// is this okay, or should it be closed after the last directory was read?
	// --> closing a channel should always be done by the sending goroutine
	// close(files)

	fmt.Printf("\n\n%v", filesWithoutDuplicates)
	fmt.Printf("\n\nExecuted in: %v", time.Since(start))
}

func readDirectory(directory string, files chan<- string, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	f, err := os.Open(directory)
	if err != nil {
		fmt.Println(err)
	}

	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		log.Println(err)
	}

	if info.IsDir() {
		directoryInfos, err := f.Readdir(-1)

		if err != nil {
			log.Println(err)
		}

		for _, directoryInfo := range directoryInfos {
			path := fmt.Sprintf("%s/%s", directory, directoryInfo.Name())

			if directoryInfo.IsDir() {
				go readDirectory(path, files, wg)
			} else {
				// sends to channel files, this locks as long as nothing is taken
				files <- path
			}
		}
	}
}

func readFiles(files <-chan string, filesWithoutDuplicates *[]string, counter *int, wg *sync.WaitGroup) {
	for {
		select {
		case filepath := <-files:
			// it seems like this gets executed one time to often and that time filepath is empty
			// why?
			// --> because closing a channel always sends a message :)
			if len(filepath) > 0 {
				*counter++

				file, err := os.Open(filepath)
				if err != nil {
					log.Println(err)
				}

				hash := sha256.New()
				if _, err := io.Copy(hash, file); err != nil {
					log.Println(err)
				}
				sha := base64.URLEncoding.EncodeToString(hash.Sum(nil))

				fmt.Printf("[%d] Reading: %s -> %s\n", *counter, filepath, sha)

				if !contains(*filesWithoutDuplicates, sha) {
					*filesWithoutDuplicates = append(*filesWithoutDuplicates, sha)
				}
			}
		case <-time.NewTimer(time.Second * 10).C:
			// is this really a proper way to do this?
			// at least it seems to work...
			fmt.Println("ending")
			wg.Done()
		}
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
