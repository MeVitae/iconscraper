// Package scraper provides functionalities for scraping and processing images from domains and returns the best one based on its size and your target size.
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

	// result holds the acctual result of the worker, it holds the URL, Image and Source.
	result Icon
}

// Icon is an icon
type Icon struct {
	// URL is the source location from which the data was fetched or derived.
	URL string

	// Image holds the parsed image, or nil if the image wasn't parsed (currently the case for ico files).
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

type channelHandling struct {
	urlChan   chan string
	imageCh   chan imageData
	imgDoneCh chan imageData
}

type SafeCounter struct {
	mu    sync.Mutex
	count int
}

// GetIcons scrapes icons from the provided domains concurrently and returns the results as a map from domain to the best image based on the given target.
//
// It fetches images from the given domains using multiple worker goroutines.
//
// Parameters:
//   - domains: The domains from which icons are to be scraped.
//   - squareOnly: If true, only square icons are considered.
//   - targetHeight: An integer representing the target height of the images to be fetched.
//   - maxConcurrentProcesses: An integer defining the maximum number of concurrent worker goroutines to be used.
func GetIcons(domains []string, squareOnly bool, targetHeight, maxConcurrentProcesses int) map[string]Icon {
	// Channel to collect errors
	errors := make(chan error, 32000)
	defer close(errors)
	go logErrors(errors)

	// HTTP worker pool
	http := newHttpWorkerPool(maxConcurrentProcesses)
	defer http.close()

	// Channel to collect results
	results := make(chan processReturn)
	defer close(results)

	// Spawn a goroutine for every domain, these will be rate limited by the http pool.
	for _, domain := range domains {
		go processDomain(domain, squareOnly, targetHeight, http, errors, results)
	}

	// Collect results
	resultMap := make(map[string]Icon, len(domains))
	for idx := 0; idx < len(domains); idx++ {
		res := <-results
		resultMap[res.domain] = res.result
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
func GetIcon(domain string, squareOnly bool, targetHeight, maxConcurrentProcesses int) Icon {
	// Channel to collect errors
	errors := make(chan error, 32000)
	defer close(errors)
	go logErrors(errors)

	// HTTP worker pool
	http := newHttpWorkerPool(maxConcurrentProcesses)
	defer http.close()

	// Channel to collect results
	results := make(chan processReturn, 1)
	defer close(results)

	processDomain(domain, squareOnly, targetHeight, http, errors, results)
	return (<-results).result
}

// httpJob represents a GET request, where the results should be sent down the result channel.
type httpJob struct {
	url    string
	result chan httpResult
}

// httpWorkerPool manages a fixed size pool of workers to perform HTTP requests.
type httpWorkerPool struct {
	// jobs is the channel to send jobs to be completed
	//
	// Results are returned down the channel specified in the job
	jobs chan httpJob

	// wg for the spawned workers
	wg sync.WaitGroup
}

func newHttpWorkerPool(workers int) *httpWorkerPool {
	jobs := make(chan httpJob)
	var wg sync.WaitGroup
	wg.Add(workers)
	pool := &httpWorkerPool{
		jobs: jobs,
		wg:   wg,
	}
	for i := 0; i < workers; i++ {
		go pool.worker()
	}
	return pool
}

func (pool *httpWorkerPool) worker() {
	defer pool.wg.Done()
	for job := range pool.jobs {
		job.result <- httpGet(job.url)
	}
}

func (pool *httpWorkerPool) get(url string) httpResult {
	httpResultChan := make(chan httpResult)
	pool.jobs <- httpJob{
		url:    url,
		result: httpResultChan,
	}
	return <-httpResultChan
}

func (pool *httpWorkerPool) close() {
	close(pool.jobs)
	pool.wg.Wait()
}
