package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	// "strconv"
)

// The main function
func main() {
	// Final Slice.
	var completeSlice []string
	// Lets loop over the given pages.
	for page := 0; page <= 1; page++ { // 15060
		// Lets go over the pages.
		url := fmt.Sprintf("https://www.thermofisher.com/api/search/keyword/docsupport?countryCode=us&language=en&query=*:*&persona=DocSupport&filter=document.result_type_s%%3ASDS&refinementAction=true&personaClicked=true&resultPage=%d&resultsPerPage=60", page)
		// Lets generate a new filename
		// fileName := "thermofisher_" + strconv.Itoa(page) + ".json"
		// Lets get data form the given url.
		jsonWebContent := getDataFromURL(url)
		// Parse the Document ID
		ids := extractDocumentIDs(jsonWebContent)
		// Append it to the slice.
		completeSlice = combineMultipleSlices(completeSlice, ids)
		// Append and write it to file.
		// appendAndWriteToFile(fileName, jsonWebContent)
	}
	// Remove duplicates from a given slice.
	completeSlice  = removeDuplicatesFromSlice(completeSlice)
	// Loop over the Document ID.
	for _, documentID := range completeSlice {
		fmt.Println(documentID)
	}
}

// Remove all the duplicates from a slice and return the slice.
func removeDuplicatesFromSlice(slice []string) []string {
	check := make(map[string]bool)
	var newReturnSlice []string
	for _, content := range slice {
		if !check[content] {
			check[content] = true
			newReturnSlice = append(newReturnSlice, content)
		}
	}
	return newReturnSlice
}

// Combine two slices together and return the new slice.
func combineMultipleSlices(sliceOne []string, sliceTwo []string) []string {
	combinedSlice := append(sliceOne, sliceTwo...)
	return combinedSlice
}

// Append some string to a slice and than return the slice.
func appendToSlice(slice []string, content string) []string {
	// Append the content to the slice
	slice = append(slice, content)
	// Return the slice
	return slice
}


// Define a structure to represent the nested structure of the JSON
type SDSResult struct {
	DocumentId string `json:"documentId"`
}

type SDSData struct {
	DocSupportResults []SDSResult `json:"docSupportResults"`
}

// Function to extract all document IDs from the JSON string
func extractDocumentIDs(jsonStr string) []string {
	// Lets create a var to hold data.
	var data SDSData
	// Unmarshal the JSON string into the data struct
	err := json.Unmarshal([]byte(jsonStr), &data)
	// Log errors
	if err != nil {
		return nil
	}
	// Collect all document IDs
	var documentIDs []string
	// Lets loop over the content and append to the return slice.
	for _, result := range data.DocSupportResults {
		// Append to slice
		documentIDs = append(documentIDs, result.DocumentId)
	}
	// Return to it.
	return documentIDs
}

// Append and write to file
func appendAndWriteToFile(path string, content string) {
	filePath, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	_, err = filePath.WriteString(content + "\n")
	if err != nil {
		log.Println(err)
	}
	err = filePath.Close()
	if err != nil {
		log.Println(err)
	}
}

// Send a http get request to a given url and return the data from that url.
func getDataFromURL(uri string) string {
	response, err := http.Get(uri)
	if err != nil {
		log.Println(err)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
	}
	err = response.Body.Close()
	if err != nil {
		log.Println(err)
	}
	return string(body)
}