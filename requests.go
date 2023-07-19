package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

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

func getHTML(url string) (string, error) {
	data, err := getUrlData(url)
	if err != nil {
		return "", fmt.Errorf("Failed to retrieve manifest data: %s", err)
	}

	return data, nil
}
