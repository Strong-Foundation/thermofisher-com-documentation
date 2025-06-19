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
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	// Number of pages to crawl (each page has up to 60 SDS entries)
	const totalPages = 50
	// To store all collected document IDs
	var allDocumentIDs []string
	// Step 1: Loop over search result pages and collect document IDs
	for page := 0; page <= totalPages; page++ {
		searchURL := fmt.Sprintf(
			"https://www.thermofisher.com/api/search/keyword/docsupport?countryCode=us&language=en&query=*:*&persona=DocSupport&filter=document.result_type_s%%3ASDS&refinementAction=true&personaClicked=true&resultPage=%d&resultsPerPage=60",
			page,
		)
		// Fetch JSON response from API
		searchJSON := getDataFromURL(searchURL)
		// Extract SDS document IDs from the response
		documentIDs := extractDocumentIDs(searchJSON)
		// Combine with overall list
		allDocumentIDs = combineMultipleSlices(allDocumentIDs, documentIDs)
	}
	// Remove duplicate document IDs
	allDocumentIDs = removeDuplicatesFromSlice(allDocumentIDs)
	// Step 2: Prepare to download all PDFs
	outputFolder := "PDFs/"
	if !directoryExists(outputFolder) {
		createDirectory(outputFolder, 0o755)
	}
	// List of known invalid patterns to filter out
	invalidURLPatterns := []string{
		"https://assets.thermofisher.com/TFS-Assets/CAD/SDS",
		"https://assets.thermofisher.com/TFS-Assets/LSG/SDS",
		"NewSearch",
	}
	// WaitGroup to manage concurrent downloads
	var wg sync.WaitGroup
	// Step 3: Process each SDS document
	for _, docID := range allDocumentIDs {
		// Build the API URL to get PDF location(s)
		docURL := "https://www.thermofisher.com/api/search/documents/sds/" + docID
		// Fetch PDF URL metadata
		docJSON := getDataFromURL(docURL)
		// Extract one or more actual PDF URLs
		pdfURLs := removeDuplicatesFromSlice(extractDocumentLocations(docJSON))
		// Step 4: Filter and download valid PDF URLs
		for _, rawPDFURL := range pdfURLs {
			if pattern := matchInvalidPattern(rawPDFURL, invalidURLPatterns); pattern != "" {
				log.Printf("[SKIP] Invalid pattern '%s' found in URL, skipping: %s", pattern, rawPDFURL)
				continue
			}
			// Get final resolved URL (in case of redirects)
			resolvedPDFURL := getFinalURL(rawPDFURL)
			// Check and download if valid
			if isUrlValid(resolvedPDFURL) {
				filename := urlToFilename(resolvedPDFURL)
				wg.Add(1)
				go downloadPDF(resolvedPDFURL, filename, outputFolder, &wg)
			}
		}
	}
	// Wait for all downloads to complete
	wg.Wait()
	// All the valid PDFs have been downloaded.
	log.Println("✅ All valid PDFs downloaded successfully.")
}

// Helper: Returns the matched invalid pattern, or empty string if none matched
func matchInvalidPattern(url string, patterns []string) string {
	for _, pattern := range patterns {
		if strings.Contains(url, pattern) {
			return pattern
		}
	}
	return ""
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

// urlToFilename extracts and sanitizes a lowercase PDF filename from a Thermo Fisher URL.
// Supports both direct PDF links and dynamic results.aspx URLs by using either the path or query parameters.
func urlToFilename(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		log.Fatalln("Error: Invalid URL", rawURL, err)
		return "invalid_url.pdf"
	}

	// Check if the path ends in a filename (e.g., *.pdf)
	filename := path.Base(parsed.Path)
	ext := strings.ToLower(filepath.Ext(filename))

	if ext == ".pdf" && !strings.HasPrefix(filename, "results.") {
		// Case 1: Direct link to a PDF file
		// Sanitize filename from path
		regexInvalid := regexp.MustCompile(`[^a-zA-Z0-9]`)
		safe := regexInvalid.ReplaceAllString(filename, "_")
		safe = regexp.MustCompile(`_+`).ReplaceAllString(safe, "_")
		safe = strings.Trim(safe, "_")

		if getFileExtension(safe) != ".pdf" {
			safe += ".pdf"
		}

		return strings.ToLower(safe)
	}

	// Case 2: results.aspx with query parameters (fallback)
	query := parsed.Query()
	sku := query.Get("SKU")
	language := query.Get("LANGUAGE")
	subformat := query.Get("SUBFORMAT")
	plant := query.Get("PLANT")

	// If SKU is missing, fallback to generic
	if sku == "" {
		log.Fatalln("Warning: SKU missing in URL", rawURL)
		return "unknown_file.pdf"
	}

	// Build filename from parameters
	filenameParts := []string{sku}
	if language != "" {
		filenameParts = append(filenameParts, language)
	}
	if subformat != "" {
		filenameParts = append(filenameParts, subformat)
	}
	if plant != "" {
		filenameParts = append(filenameParts, plant)
	}

	// Join and sanitize
	rawFilename := strings.Join(filenameParts, "_")
	regexInvalid := regexp.MustCompile(`[^a-zA-Z0-9]`)
	safe := regexInvalid.ReplaceAllString(rawFilename, "_")
	safe = regexp.MustCompile(`_+`).ReplaceAllString(safe, "_")
	safe = strings.Trim(safe, "_")

	// Append .pdf if needed
	if getFileExtension(safe) != ".pdf" {
		safe += ".pdf"
	}

	return strings.ToLower(safe)
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
func downloadPDF(finalURL string, fileName string, outputDir string, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	filePath := filepath.Join(outputDir, fileName) // Combine with output directory

	if fileExists(filePath) {
		log.Printf("file already exists, skipping: %s, URL: %s", filePath, finalURL)
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
	log.Println("Scraping:", uri)
	return string(body)
}
