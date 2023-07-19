package main

import (
	"fmt"
	"image"
)

type result struct {
	domain string
	icon   image.Image
}

type processReturn struct {
	domain   string
	picture  string
	dataIcon image.Image
}

var maxConcurrentProcesses = 2
var targetImageSize = 120

func main() {
	domains := readDomains()
	urls := make(chan string, len(domains))
	returns := make(chan processReturn)

	workers := len(domains)
	if workers > maxConcurrentProcesses {
		workers = maxConcurrentProcesses
	}

	for worker := 0; worker < workers; worker++ {
		go processImageGetting(urls, targetImageSize, returns)
	}

	for _, domain := range domains {
		urls <- domain.domain
	}

	results := make(map[string]result, len(domains))
	for domainResult := range returns {
		newresults := result{domain: domainResult.picture, icon: domainResult.dataIcon}
		//saveImageAsPNG(result.dataIcon, domainResult.domain)
		results[domainResult.domain] = newresults
		fmt.Println("Finished working on:", domainResult.domain)
		if len(results) == len(domains) {
			break
		}
	}
	close(urls)

	fmt.Println("------ Finished -------")
	fmt.Println(results)
}
