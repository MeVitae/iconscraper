// Package yourpackage provides functionalities for scraping and processing images from domains and returns the best one based on its size and your target size.
package scraper

import (
	"image"
)

// processReturn is the struct used to store values received when a channel worker sends back a result.
type processReturn struct {
	// Domain is simply the domain that was processed.
	domain string

	// Result holds the acctual result of the worker, it holds the URL,Image and Source.
	result Result
}

// Result represents the data being returned from a certain operation.
type Result struct {
	// URL is the source location from which the data was fetched or derived.
	URL string

	// Image holds the image data returned as a result.
	// It implements the image.Image interface, enabling various image operations.
	Image image.Image

	// Source contains additional data related to the result.
	// This can be used to store supplementary information or any data relevant to the returned result.
	Source []byte
}

// ScrapeAll scrapes images from the provided domains concurrently and returns the results as a map with the best image based on the given target.
// It fetches images from the given domains using multiple worker goroutines to achieve parallelism.
// The function returns a map with domain names as keys and their corresponding Result as values.
// The Result struct contains information about the URL, the image data, and additional source data.
//
// Parameters:
//   - domains: A list of strings representing the domains from which images need to be scraped.
//   - targetImgSize: An integer representing the target size of the images to be fetched (e.g., width or height).
//   - maxConcurrentProcesses: An integer defining the maximum number of concurrent worker goroutines to be used.
//     The function will limit the number of workers to this value if it exceeds the length of domains.
//
// Returns:
// A map[string]Result: The map containing domain names as keys and their corresponding Result as values.
//
// Example:
// domains := []string{"https://example.com", "https://test.com", "https://yourdomain.com"}
// targetSize := 500 // Set your target image size here.
// maxProcesses := 5 // Set your desired maximum concurrent processes here.
// results := ScrapeAll(domains, targetSize, maxProcesses)
func ScrapeAll(domains []string, targetImgSize, maxConcurrentProcesses int) map[string]Result {
	urls := make(chan string, len(domains))
	returns := make(chan processReturn)

	for _, domain := range domains {
		urls <- domain
	}

	workers := len(domains)
	if workers > maxConcurrentProcesses {
		workers = maxConcurrentProcesses
	}

	for worker := 0; worker < workers; worker++ {
		go processImageGetting(urls, targetImgSize, returns)
	}

	results := make(map[string]Result, len(domains))
	for domainResult := range returns {
		results[domainResult.domain] = domainResult.result
		if len(results) == len(domains) {
			break
		}
	}
	close(urls)

	return results
}

// ScrapeOne scrapes images from the provided domain and returns the result the best image based on the given target.
// It fetches images from the given domain using a worker goroutine to perform the operation.
// The function returns a Result struct containing information about the URL, the image data, and additional source data.
//
// Parameters:
// - domain: A string representing the domain from which the image needs to be scraped.
// - targetImgSize: An integer representing the target size of the image to be fetched (e.g., width or height).
//
// Returns:
// Result: The Result struct containing information about the URL, the image data, and additional source data.
//
// Example:
// domain := "https://example.com"
// targetSize := 500 // Set your target image size here.
// result := ScrapeOne(domain, targetSize)
func ScrapeOne(domain string, targetImgSize int) Result {
	urls := make(chan string, 1)
	returns := make(chan processReturn)

	urls <- domain

	go processImageGetting(urls, targetImgSize, returns)

	result := Result{}
	for domainResult := range returns {
		result = domainResult.result
		break
	}
	close(urls)

	return result
}
