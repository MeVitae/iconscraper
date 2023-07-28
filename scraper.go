// package iconscraper provides functionalities for scraping and processing images from domains and returns the best one based on its size and your target size.
package iconscraper

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"regexp"

	"golang.org/x/net/html"
)

// logErrors logs all the errors sent on the channel to stderr
func logErrors(errors chan error) {
	for err := range errors {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}

// logWarnings logs all the warnings sent on the channel to stderr
func logWarnings(errors chan error) {
	for err := range errors {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}

// Icon is an icon
type Icon struct {
	// URL is the source location from which the data was fetched or derived.
	URL string

	// Type is the sniffed MIME type of the image.
	Type string

	// Image holds the parsed image config.
	ImageConfig image.Config

	// Source is the image source as downloaded.
	Source []byte
}

// ScraperConfig is the config used for GetIcons and GetIcon.
type Config struct {
	//squareOnly If true, only square icons are considered.
	SquareOnly bool

	// targetHeight An integer representing the target height of the images to be fetched.
	TargetHeight int

	// allowSvg If true, if svg is found svg will be returned.
	AllowSvg bool

	//maxConcurrentProcesses An integer defining the maximum number of concurrent worker goroutines to be used.
	MaxConcurrentProcesses int

	// errors is the channel for receiving errors (if left empty a go routine will automatically be created to log the errrors).
	Errors chan error

	// errors is the channel for receiving warnings (if left empty a go routine will automatically be created to log the warnings).
	Warnings chan error
}

// GetIcons scrapes icons from the provided domains concurrently and returns the results as a map from domain to the best image based on the given target.
//
// It finds the smallest icon taller than targetHeight or, if there are none, the tallest icon.
//
// If no icon is not found for a domain (or no square icon if squareOnly is true), that domain is ommited from the output map.
//
// Parameters:
//   - domains: The domains from which icons are to be scraped.
//   - config: Of type ScraperConfig which holds all the config needed for the scraper to run and find best icons.
//
// TODO: add `allowSvg` to also collect SVG images and prefer them over any other image (since they can be infinitely resized)
func GetIcons(domains []string, config Config) map[string]Icon {
	// Creating go routines for handling Errors and Warnings where none are initialised.
	if config.Errors == nil {
		config.Errors = make(chan error, 32000)
		go logErrors(config.Errors)
		defer close(config.Errors)
	}
	if config.Warnings == nil {
		config.Warnings = make(chan error, 32000)
		go logWarnings(config.Warnings)
	}

	// HTTP worker pool
	http := newHttpWorkerPool(config.MaxConcurrentProcesses)
	defer http.close()

	// Channel to collect results
	results := make(chan processReturn)
	defer close(results)

	// Spawn a goroutine for every domain, these will be rate limited by the http pool.
	for _, domain := range domains {
		go processDomain(domain, config.SquareOnly, config.TargetHeight, http, config.Warnings, config.Errors, results, config.AllowSvg)
	}

	// Collect results
	resultMap := make(map[string]Icon, len(domains))
	for idx := 0; idx < len(domains); idx++ {
		res := <-results
		if res.result != nil {
			resultMap[res.domain] = *res.result
		}
	}
	return resultMap
}

// GetIcon scrapes icons from the provided domain concurrently and returns the results as a map from domain to the best image based on the given target.
//
// It fetches images from the given domains using multiple worker goroutines.
//
// Parameters:
//   - domain: The domain from which icons are to be scraped.
//   - squareOnly: If true, only square icons are considered.
//   - targetHeight: An integer representing the target height of the images to be fetched.
//   - maxConcurrentProcesses: An integer defining the maximum number of concurrent worker goroutines to be used
//   - maxConcurrentProcesses:(this should be set from based on the network speed of the machine you are running it on).
func GetIcon(domain string, config Config) *Icon {
	// Channel to collect errors
	errors := make(chan error, 32000)
	warnings := make(chan error, 32000)
	defer close(errors)
	go logErrors(errors)
	go logWarnings(warnings)

	// HTTP worker pool
	http := newHttpWorkerPool(config.MaxConcurrentProcesses)
	defer http.close()

	// Channel to collect results
	results := make(chan processReturn, 1)
	defer close(results)

	processDomain(domain, config.SquareOnly, config.TargetHeight, http, warnings, errors, results, config.AllowSvg)
	return (<-results).result
}

// processReturn is the output of processDomain
type processReturn struct {
	// domain is the domain that was processed.
	domain string

	// result holds the result, or nil if there isn't one.
	result *Icon
}

var domainNameRegexp = regexp.MustCompile(`^([a-zA-Z0-9_][a-zA-Z0-9_-]{0,64})(\.[a-zA-Z0-9_][a-zA-Z0-9_-]{0,64})*[\._]?$`)

// couldBeDomain returns false if domain definitely isn't a valid domain.
func couldBeDomain(domain string) bool {
	return len(domain) <= 512 && domainNameRegexp.MatchString(domain)
}

// processDomain is a worker function that processes getting images for a domain.
//
// It fetches HTML content from each URL, parses the HTML content, and extracts
// image information based on keys and values variables. It then picks the best
// image from the extracted images based on the `bestSize` parameter and sends
// the best image back on the result channel, or, if not image was found, it
// sends back a nil result.
func processDomain(
	domain string,
	squareOnly bool,
	targetHeight int,
	http *httpWorkerPool,
	warnings,
	errors chan error,
	result chan processReturn,
	allowSvg bool,
) {
	// Check for obvious cases where the domain passed is invalid
	if !couldBeDomain(domain) {
		warnings <- fmt.Errorf("Invalid domain name %s", domain)
		result <- processReturn{
			domain: domain,
			result: nil,
		}
	}

	url := "https://" + domain
	httpResult := http.get(url)
	// Only check for network errors fetching, if it's an error page, that'll do.
	if httpResult.err != nil {
		warnings <- fmt.Errorf("Failed to get %s: %w", url, httpResult.err)
		result <- processReturn{
			domain: domain,
			result: nil,
		}
		return
	}

	// Parse the output HTML
	doc, err := html.Parse(bytes.NewReader(httpResult.body))
	if err != nil {
		errors <- fmt.Errorf("Error parsing HTML from %s: %w", url, err)
		result <- processReturn{
			domain: domain,
			result: nil,
		}
		return
	}

	// Our requests will be now rooted at the domain we were redirected to.
	redirectDomain := httpResult.url.Host
	url = "https://" + redirectDomain

	workers := newImageWorkers(redirectDomain, http, errors, warnings)
	// Always check for `/favicon.ico`, it's not always linked from the HTML.
	workers.spawn(url + "/favicon.ico")
	// Spawn workers scraping all the linked icons
	getImagesFromHTML(doc, redirectDomain, &workers)

	// Pick the best size image from all the results
	result <- processReturn{
		domain: domain,
		result: pickBestImage(squareOnly, targetHeight, workers.results(), allowSvg),
	}
}
