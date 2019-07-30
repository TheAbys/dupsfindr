package main

import (
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

func main() {
	start := time.Now()

	directory := flag.String("directory", "", "the absolute directorypath to search for duplicates")
	flag.Parse()

	fmt.Println("DupsFindr started with: ", *directory)

	if len(*directory) > 0 {
		files := make(chan string, 10)

		allFilesWithoutDuplicates := make([]string, 0, 5)

		var wg sync.WaitGroup
		wg.Add(1)
		// read the directory and send all files to the channel
		go readDirectory(*directory, files, &wg)

		// go listFiles(files)
		fmt.Println("")
		counter := 0
		wg.Add(1)
		go readFiles(files, &allFilesWithoutDuplicates, &counter, &wg)
		// go readFiles(files, finals, &counter)

		wg.Wait()
		close(files)

		fmt.Println()
		fmt.Println()
		fmt.Println(allFilesWithoutDuplicates)
	}
	fmt.Println()
	fmt.Println()
	fmt.Println("Executed in: ", time.Since(start))
}

func readDirectory(directory string, files chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	// fmt.Println("Directory:", directory)
	f, err := os.Open(directory)
	defer f.Close()

	if err != nil {
		fmt.Println(err)
	}

	info, err := f.Stat()

	if err != nil {
		fmt.Println(err)
	}

	if info.IsDir() {
		directoryInfos, err := f.Readdir(-1)

		if err != nil {
			fmt.Println(err)
		}

		for _, directoryInfo := range directoryInfos {
			if directoryInfo.IsDir() {
				wg.Add(1)
				go readDirectory(directory+"/"+directoryInfo.Name(), files, wg)
			} else {
				// sends to channel files, this locks as long as nothing is taken
				files <- directory + "/" + directoryInfo.Name()
			}
		}
	}
}

func readFiles(files <-chan string, allFilesWithoutDuplicates *[]string, counter *int, wg *sync.WaitGroup) {
	for {
		select {
		case filepath := <-files:
			// it seems like this gets executed one time to often and that time filepath is empty
			// why?
			if len(filepath) > 0 {
				*counter++
				fmt.Printf("%d Reading: [%s]", *counter, filepath)
				file, err := os.Open(filepath)
				if err != nil {
					fmt.Println(err)
				}

				hash := sha256.New()
				if _, err := io.Copy(hash, file); err != nil {
					fmt.Println(err)
				}
				sha := base64.URLEncoding.EncodeToString(hash.Sum(nil))
				fmt.Printf(" - %s\n", sha)
				if !contains(*allFilesWithoutDuplicates, sha) {
					*allFilesWithoutDuplicates = append(*allFilesWithoutDuplicates, sha)
				}
			}
		case <-time.NewTimer(time.Nanosecond * 1).C:
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
