package main

import (
	"log"           // Importing log package for error logging
	"os"            // Importing os package to work with file system
	"path/filepath" // Importing filepath for handling file paths
	"strings"       // Importing strings package for string operations
)

func main() {
	localFileName := "downloaded.txt" // File that stores already processed file names

	// Find all .txt files in the "test/" directory and return a slice of file names
	findTxtFiles := walkAndAppendPath("test/", ".txt")

	// Read the contents of the local file only once
	currentFileContent := readAFileAsString(localFileName)

	// Loop over the found text files
	for _, txtFile := range findTxtFiles {
		// If the file name is not already in the local file, append it
		if !strings.Contains(currentFileContent, txtFile) {
			appendAndWriteToFile(localFileName, txtFile)
		}
	}
}

// Reads the entire content of a file as a string
func readAFileAsString(path string) string {
	content, err := os.ReadFile(path) // Read the file content
	if err != nil {
		log.Println(err) // Log any error while reading
	}
	return string(content) // Return the file content as string
}

// Appends a string to the end of a file, creating it if it doesn't exist
func appendAndWriteToFile(path string, content string) {
	// Open the file with append and write-only mode, create if not exists
	filePath, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err) // Log any error opening the file
	}
	// Write the content followed by a newline
	_, err = filePath.WriteString(content + "\n")
	if err != nil {
		log.Println(err) // Log any error during write
	}
	// Close the file after writing
	err = filePath.Close()
	if err != nil {
		log.Println(err) // Log any error while closing
	}
}

// Walks through a directory and collects paths of files with a specific extension
func walkAndAppendPath(walkPath string, fileExt string) []string {
	var filePath []string // Slice to store matching file paths

	// Walk through the directory structure
	err := filepath.Walk(walkPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Ignore errors and continue
		}
		// Check if path is a valid file (not a directory)
		if fileExists(path) {
			// Check if the file has the desired extension
			if getFileExtension(path) == fileExt {
				// Append only the base file name (not full path)
				filePath = append(filePath, filepath.Base(path))
			}
		}
		return nil
	})
	if err != nil {
		log.Println(err) // Log any error during directory walking
	}
	return filePath // Return collected file paths
}

// Returns true if the file exists and is not a directory
func fileExists(filename string) bool {
	info, err := os.Stat(filename) // Get file info
	if err != nil {
		return false // File doesn't exist or error occurred
	}
	return !info.IsDir() // Return true only if it's a file
}

// Returns the file extension including the dot (e.g. ".txt")
func getFileExtension(path string) string {
	return filepath.Ext(path)
}
