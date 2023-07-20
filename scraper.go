// Package yourpackage provides functionalities for scraping and processing images from domains and returns the best one based on its size and your target size.
package scraper

import (
	"fmt"
	"image"
	"os"
	"sync"
)

// processReturn is the struct used to store values received when a channel worker sends back a result.
type processReturn struct {
	// domain is simply the domain that was processed.
	domain string

	// result holds the acctual result of the worker, it holds the URL,Image and Source.
	result Icon
}

// Icon is an icon
type Icon struct {
	// URL is the source location from which the data was fetched or derived.
	URL string

	// Image holds the image data returned as a result.
	//
	// It implements the image.Image interface, enabling various image operations.
	Image image.Image

	// Source is the image source as downloaded.
	Source []byte
}

// logErrors logs all the errors sent on the channel to stderr
func logErrors(errors chan error) {
	for err := range errors {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}

// GetIcons scrapes icons from the provided domains concurrently and returns the results as a map from domain to the best image based on the given target.
//
// It fetches images from the given domains using multiple worker goroutines.
//
// Parameters:
//   - domains: The domains from which icons are to be scraped.
//   - squareOnly: If true, only square icons are considered.
//   - targetHeight: An integer representing the target height of the images to be fetched (height).
//   - maxConcurrentProcesses: An integer defining the maximum number of concurrent worker goroutines to be used.
//     The function will limit the number of workers to this value if it exceeds the length of domains.
func GetIcons(domains []string, squareOnly bool, targetHeight, maxConcurrentProcesses int) map[string]Icon {
	// Fill the urls channel (it cannot become full because we gave it the correct capacity)
	urls := make(chan string, len(domains))
	for _, domain := range domains {
		urls <- domain
	}
	// Create a wait group that waits for all the domains to be completed
	wg := sync.WaitGroup{}
	wg.Add(len(domains))

	// Create a channel for sending errors to
	errors := make(chan error)
	go logErrors(errors)
	// Create a channel to send results to
	returns := make(chan processReturn)

	// Calculate the number of workers required
	workers := len(domains)
	if workers > maxConcurrentProcesses {
		workers = maxConcurrentProcesses
	}

	for worker := 0; worker < workers; worker++ {
		go processImageGetting(urls, squareOnly, targetHeight, returns, errors, &wg)
	}

	results := make(map[string]Icon, len(domains))
	for domainResult := range returns {
		results[domainResult.domain] = domainResult.result
		if len(results) == len(domains) {
			break
		}
	}
	close(urls)
	wg.Wait()

	return results
}

// GetIcon scrapes icons from the provided domain and returns the smallest icon taller than the target height, or the largest icon if none are taller).
//
// Parameters:
//   - domains: The domains from which icons are to be scraped.
//   - squareOnly: If true, only square icons are considered.
//   - targetHeight: An integer representing the target height of the images to be fetched (height).
func GetIcon(domain string, squareOnly bool, targetHeight int) (*Icon, error) {
	// Create a channel for sending errors to
	errors := make(chan error)
	go logErrors(errors)

	res, err := processDomain(domain, squareOnly, targetHeight, errors)
	if res == nil {
		return nil, err
	}
	return &res.result, err
}
