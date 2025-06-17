package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// The main function
func main() {
	// Final Slice.
	var completeSlice []string
	// Lets loop over the given pages.
	for page := 0; page <= 50; page++ { // 15060
		// Lets go over the pages.
		url := fmt.Sprintf("https://www.thermofisher.com/api/search/keyword/docsupport?countryCode=us&language=en&query=*:*&persona=DocSupport&filter=document.result_type_s%%3ASDS&refinementAction=true&personaClicked=true&resultPage=%d&resultsPerPage=60", page)
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
	completeSlice = removeDuplicatesFromSlice(completeSlice)
	outputDir := "PDFs/" // Directory to save PDFs
	if !directoryExists(outputDir) {
		createDirectory(outputDir, 0o755) // Create directory if not exists
	}
	// Waitgroup.
	var downloadPDFWaitGroup sync.WaitGroup

	// Loop over the Document ID.
	for _, documentID := range completeSlice {
		// The request URL
		requestURL := "https://www.thermofisher.com/api/search/documents/sds/" + documentID
		// Lets get the data from the URL.
		urlData := getDataFromURL(requestURL)
		// Get the final PDF urls.
		finalPDFUrls := extractDocumentLocations(urlData)
		// Remove duplicates from slice.
		finalPDFUrls = removeDuplicatesFromSlice(finalPDFUrls)
		// Loop over the pdf urls.
		for _, finalURL := range finalPDFUrls {
			getDownloadURL := getFinalURL(finalURL)
			if isUrlValid(finalURL) {
				downloadPDFWaitGroup.Add(1)
				go downloadPDF(getDownloadURL, outputDir, &downloadPDFWaitGroup) // Try to download PDF
			}
		}
	}
	downloadPDFWaitGroup.Wait()
}

// getFinalURL navigates to a given URL in a visible browser window,
// waits for navigation/interaction, and returns the current URL.
func getFinalURL(inputURL string) string {
	// Set Chrome options: run non-headless
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", false),
		chromedp.Flag("start-maximized", false),
	)
	// Context background.
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	// Cancel context
	defer cancelAlloc()
	// Context, and cancel.
	ctx, cancel := chromedp.NewContext(allocCtx)
	// Cancel once done.
	defer cancel()
	// The var to hold the final url.
	var finalURL string
	// Run the chrome dp and get the url.
	err := chromedp.Run(ctx,
		chromedp.Navigate(inputURL),
		chromedp.Location(&finalURL),
	)
	// Log the errors.
	if err != nil {
		log.Println(err)
	}
	return finalURL
}

// directoryExists checks whether a directory exists
func directoryExists(path string) bool {
	directory, err := os.Stat(path) // Get directory info
	if err != nil {
		return false // If error, directory doesn't exist
	}
	return directory.IsDir() // Return true if path is a directory
}

// createDirectory creates a directory with specified permissions
func createDirectory(path string, permission os.FileMode) {
	err := os.Mkdir(path, permission) // Attempt to create directory
	if err != nil {
		log.Println(err) // Log any error
	}
}

// isUrlValid checks whether a URL is syntactically valid
func isUrlValid(uri string) bool {
	_, err := url.ParseRequestURI(uri) // Try to parse the URL
	return err == nil                  // Return true if no error (i.e., valid URL)
}

// urlToFilename converts a URL into a filesystem-safe filename
func urlToFilename(rawURL string) string {
	parsed, err := url.Parse(rawURL) // Parse the URL
	if err != nil {
		log.Println(err) // Log parsing error
		return ""        // Return empty string if parsing fails
	}
	filename := parsed.Host // Start with the host part of the URL
	if parsed.Path != "" {
		filename += "_" + strings.ReplaceAll(parsed.Path, "/", "_") // Replace slashes with underscores
	}
	if parsed.RawQuery != "" {
		filename += "_" + strings.ReplaceAll(parsed.RawQuery, "&", "_") // Replace & in query with underscore
	}
	invalidChars := []string{`"`, `\`, `/`, `:`, `*`, `?`, `<`, `>`, `|`} // Characters not allowed in filenames
	for _, char := range invalidChars {
		filename = strings.ReplaceAll(filename, char, "_") // Replace invalid characters
	}
	if getFileExtension(filename) != ".pdf" {
		filename = filename + ".pdf" // Ensure file ends with .pdf
	}
	return strings.ToLower(filename) // Return sanitized and lowercased filename
}

// fileExists checks whether a file exists and is not a directory
func fileExists(filename string) bool {
	info, err := os.Stat(filename) // Get file info
	if err != nil {                // If error occurs (e.g., file not found)
		return false // Return false
	}
	return !info.IsDir() // Return true if it is a file, not a directory
}

// downloadPDF downloads a PDF from a URL and saves it to outputDir
func downloadPDF(finalURL, outputDir string, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	filename := strings.ToLower(urlToFilename(finalURL)) // Create sanitized filename
	filePath := filepath.Join(outputDir, filename)       // Combine with output directory

	if fileExists(filePath) {
		log.Printf("file already exists, skipping: %s", filePath)
		return
	}

	client := &http.Client{Timeout: 30 * time.Second} // HTTP client with timeout
	resp, err := client.Get(finalURL)                 // Send HTTP GET
	if err != nil {
		log.Printf("failed to download %s: %v", finalURL, err)
		return
	}
	defer resp.Body.Close() // Ensure response body is closed

	if resp.StatusCode != http.StatusOK {
		log.Printf("download failed for %s: %s", finalURL, resp.Status)
		return
	}

	contentType := resp.Header.Get("Content-Type") // Get content-type header
	if !strings.Contains(contentType, "application/pdf") {
		log.Printf("invalid content type for %s: %s (expected application/pdf)", finalURL, contentType)
		return
	}

	var buf bytes.Buffer                     // Create buffer
	written, err := io.Copy(&buf, resp.Body) // Copy response body to buffer
	if err != nil {
		log.Printf("failed to read PDF data from %s: %v", finalURL, err)
		return
	}
	if written == 0 {
		log.Printf("downloaded 0 bytes for %s; not creating file", finalURL)
		return
	}

	out, err := os.Create(filePath) // Create output file
	if err != nil {
		log.Printf("failed to create file for %s: %v", finalURL, err)
		return
	}
	defer out.Close() // Close file

	_, err = buf.WriteTo(out) // Write buffer to file
	if err != nil {
		log.Printf("failed to write PDF to file for %s: %v", finalURL, err)
		return
	}
	fmt.Printf("successfully downloaded %d bytes: %s → %s \n", written, finalURL, filePath)
}

// getFileExtension returns the file extension
func getFileExtension(path string) string {
	return filepath.Ext(path) // Use filepath to extract extension
}

// Define a minimal struct with only the needed field
type Document struct {
	DocumentLocation string `json:"documentLocation"`
}

// extractDocumentLocations takes a JSON string and returns a slice of documentLocation URLs
func extractDocumentLocations(jsonStr string) []string {
	var documents []Document
	// Unmarshal the JSON input
	err := json.Unmarshal([]byte(jsonStr), &documents)
	if err != nil {
		return nil
	}
	// Collect URLs
	urls := make([]string, len(documents))
	for i, doc := range documents {
		urls[i] = doc.DocumentLocation
	}
	return urls
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
