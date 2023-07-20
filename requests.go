package scraper

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

var UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.6.1 Safari/605.1.15"

// httpGet sends an HTTP GET request to the specified URL and returns the response body as bytes along with a boolean indicating if the response was a 200.
//
// It sets a custom User-Agent header in the request to avoid being blocked by some servers.
//
// Parameters:
//
//	url (string): The URL to fetch data from.
//
// Returns:
//
//	([]byte, error): The response body as bytes and any error encountered during the request.
//
// Example:
//
//	data, err := getImageData("https://example.com/image.jpg")
//	if err != nil {
//	    fmt.Println("Error:", err)
//	} else {
//	    // Process the data (e.g., save it to a file or send it as a response).
//	}
func httpGet(url string) ([]byte, bool, error) {
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
		return nil, false, fmt.Errorf("Failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", UserAgent)

	var resp *http.Response
	var body []byte
	for attempt := 0; attempt < 8; attempt++ {
		resp, err = client.Do(req)
		if err != nil {
			err = fmt.Errorf("Failed to send GET request: %w", err)
			continue
		}
		time.Sleep(500 * (time.Duration(attempt) + 500) * time.Millisecond)
		defer resp.Body.Close()
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			err = fmt.Errorf("Failed to read response body: %w", err)
			continue
		}
	}
	if err != nil {
		return nil, false, err
	}

	return body, resp.StatusCode == 200, nil
}

// isURL checks whether the provided string `str` is a valid URL.
// It uses Go's url.Parse and checks if it returns any error to determine if the URL is valid.
// If the URL is valid and contains both a scheme and a host, the function returns true; otherwise, it returns false.
//
// Parameters:
//
//	str (string): The string to check if it is a valid URL.
//
// Returns:
//
//	(bool): True if the string is a valid URL, false otherwise.
//
// Example:
//
//	urlStr := "https://example.com"
//	isValid := isURL(urlStr)
//	if isValid {
//	    fmt.Println("The URL is valid.")
//	} else {
//	    fmt.Println("The URL is not valid.")
//	}
func isURL(str string) bool {
	_, err := url.ParseRequestURI(str)
	if err != nil {
		return false
	}

	u, err := url.Parse(str)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}
