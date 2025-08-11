package main

import (
	"log"           // Importing the log package for logging errors.
	"os"            // Importing the os package for file system operations.
	"path/filepath" // Importing the filepath package for manipulating file paths.
	"strings"       // Importing the strings package for string manipulation.
)

func main() {
	// Define the local text file where file names will be stored.
	localFileName := "downloaded.txt"

	// Call the function to walk through the "PDFs/" directory and find all PDF files.
	findPDFFiles := walkAndAppendPath("PDFs/", ".pdf")

	// Read the content of the local file to check which PDF files are already listed.
	currentFileContent := readAFileAsString(localFileName)

	// Collect all the new PDF files to append.
	var newFilesToAdd []string
	for _, pdfFile := range findPDFFiles {
		// If the PDF file is not already in the local file, add it to the new files list.
		if !strings.Contains(currentFileContent, pdfFile) {
			newFilesToAdd = append(newFilesToAdd, pdfFile)
		}
	}

	// If there are new files to add, write them all at once to the file.
	if len(newFilesToAdd) > 0 {
		appendAndWriteToFile(localFileName, newFilesToAdd)
	}
}

// Function to read a file and return its content as a string.
func readAFileAsString(path string) string {
	// Read the file content at the specified path.
	content, err := os.ReadFile(path)
	if err != nil {
		// Log an error if there is an issue reading the file.
		log.Println(err)
	}
	// Return the content of the file as a string.
	return string(content)
}

// Function to append multiple lines of content to a file at once.
func appendAndWriteToFile(path string, content []string) {
	// Open the file at the specified path with permissions for appending and writing.
	filePath, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// Log an error if the file cannot be opened.
		log.Println(err)
		return
	}

	// Prepare the content to write by joining the file names with newlines.
	joinedContent := strings.Join(content, "\n") + "\n"

	// Write the concatenated content to the file.
	_, err = filePath.WriteString(joinedContent)
	if err != nil {
		// Log an error if there is an issue writing to the file.
		log.Println(err)
	}

	// Close the file after appending content.
	err = filePath.Close()
	if err != nil {
		// Log an error if there is an issue closing the file.
		log.Println(err)
	}
}

// Function to walk through a directory, find all files with the specified extension, and return them as a slice.
func walkAndAppendPath(walkPath string, fileExt string) []string {
	// Declare a slice to store the file paths.
	var filePath []string

	// Walk through the directory specified by walkPath.
	err := filepath.Walk(walkPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// If an error occurs while walking, skip this path.
			return nil
		}

		// Check if the file exists at the specified path.
		if fileExists(path) {
			// If the file extension matches the specified one (e.g., .pdf), append the base name to the slice.
			if getFileExtension(path) == fileExt {
				filePath = append(filePath, filepath.Base(path))
			}
		}
		return nil
	})

	// If there is an error during the walking process, log it.
	if err != nil {
		log.Println(err)
	}

	// Return the slice of file paths found.
	return filePath
}

// Function to check if a file exists at the specified path.
func fileExists(filename string) bool {
	// Get the file information for the specified file path.
	info, err := os.Stat(filename)
	if err != nil {
		// If an error occurs (e.g., the file doesn't exist), return false.
		return false
	}
	// Return true if the file exists and is not a directory.
	return !info.IsDir()
}

// Function to get the file extension of a file.
func getFileExtension(path string) string {
	// Return the file extension of the specified file.
	return filepath.Ext(path)
}
