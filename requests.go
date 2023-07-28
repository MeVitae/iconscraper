package scraper

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.6.1 Safari/605.1.15"

// httpJob represents a GET request, where the results should be sent down the result channel.
type httpJob struct {
	url    string
	result chan httpResult
}

// httpResult represents the result of attempting to make a HTTP request. There will only be an error if multiple attempts where made.
type httpResult struct {
	// url sent to receive the final response, this be different if a redirect occured
	url    *url.URL
	status int
	body   []byte
	err    error
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

// get requests a worker perform a HTTP GET request, and then waits for and returns the result.
//
// Requests are made on a first-come-first-serve basis.
func (pool *httpWorkerPool) get(url string) httpResult {
	httpResultChan := make(chan httpResult)
	pool.jobs <- httpJob{
		url:    url,
		result: httpResultChan,
	}
	return <-httpResultChan
}

// Wait for all jobs to be completed and end all worker threads.
func (pool *httpWorkerPool) close() {
	close(pool.jobs)
	pool.wg.Wait()
}

// httpGet sends an HTTP GET request to the specified URL and returns the result as a httpResult.
//
// It sets a custom User-Agent header in the request to avoid being blocked by some servers.
func httpGet(url string) httpResult {
	if !isURL(url) {
		url = "https://" + url
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return httpResult{
			url:    req.URL,
			status: 0,
			body:   nil,
			err:    fmt.Errorf("Failed to create request: %w", err),
		}
	}
	req.Header.Set("User-Agent", UserAgent)

	var resp *http.Response
	var body []byte
	for attempt := 0; attempt < 6; attempt++ {
		time.Sleep(500 * time.Duration(attempt) * time.Millisecond)
		resp, err = client.Do(req)
		if err != nil {
			err = fmt.Errorf("Failed to send GET request: %w", err)
			continue
		}
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			err = fmt.Errorf("Server returned a server error status: %d %s", resp.StatusCode, resp.Status)
			continue
		}

		body, err = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			err = fmt.Errorf("Failed to read response body: %w", err)
			continue
		}

		break
	}
	if err != nil {
		return httpResult{
			url:    req.URL,
			status: 0,
			body:   nil,
			err:    err,
		}
	}
	return httpResult{
		url:    resp.Request.URL,
		status: resp.StatusCode,
		body:   body,
		err:    nil,
	}
}

// isURL checks whether the provided string `str` is a valid URL.
//
// It uses Go's url.Parse and checks if it returns any error to determine if the URL is valid.
//
// If the URL is valid and contains both a scheme and a host, the function returns true; otherwise, it returns false.
func isURL(str string) bool {
	_, err := url.ParseRequestURI(str)
	if err != nil {
		return false
	}
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}
