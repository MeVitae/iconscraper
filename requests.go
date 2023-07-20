package scraper

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

// getImageData sends an HTTP GET request to the specified URL and returns the response body as bytes.
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
func getImageData(url string) ([]byte, error) {
	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.6.1 Safari/605.1.15"

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create request: %s", err)
	}

	req.Header.Set("User-Agent", userAgent)

	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to send GET request: %s", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read response body: %s", err)
	}

	return body, nil
}

// getUrlData sends an HTTP GET request to the specified URL and returns the response body as a string.
// It sets a custom User-Agent header in the request to avoid being blocked by some servers.
//
// Parameters:
//
//	url (string): The URL to fetch data from.
//
// Returns:
//
//	(string, error): The response body as a string and any error encountered during the request.
//
// Example:
//
//	data, err := getUrlData("https://example.com/api/data")
//	if err != nil {
//	    fmt.Println("Error:", err)
//	} else {
//	    // Process the data (e.g., parse it as JSON or display it in the console).
//	}
func getUrlData(url string) (string, error) {
	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.6.1 Safari/605.1.15"

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}

	req, err := http.NewRequest("GET", "https://"+url, nil)
	if err != nil {
		return "", fmt.Errorf("Failed to create request: %s", err)
	}

	req.Header.Set("User-Agent", userAgent)

	response, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Failed to send GET request: %s", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("Failed to read response body: %s", err)
	}

	return string(body), nil
}
