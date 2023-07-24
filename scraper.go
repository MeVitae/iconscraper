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

type channelHandling struct {
	urlChan   chan string
	imageCh   chan imageData
	imgDoneCh chan imageData
}

type SafeCounter struct {
	mu    sync.Mutex
	count int
}
// asd
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
func GetIcons(domains []string, squareOnly bool, targetHeight, maxConcurrentProcesses int) {
	var ProcessingImagesWG sync.WaitGroup
	var WebProcessingWG sync.WaitGroup
	AllLinksSent := false
	WorkersWaiting := SafeCounter{}
	channels := map[string]channelHandling{}
	imageSendingCh := make(chan imageData, 32000)
	errors := make(chan error, 32000)
	imageReceivingCh := make(chan imageData, 32000)
	for _, domain := range domains {
		channels[domain] = channelHandling{urlChan: make(chan string, 32000), imageCh: make(chan imageData, 32000), imgDoneCh: make(chan imageData, 32000)}
		go processDomain(domain, channels[domain], errors, &WebProcessingWG, imageReceivingCh)
	}

	for i := 0; i < maxConcurrentProcesses; i++ {
		go worker(&imageSendingCh, &imageReceivingCh, errors, &ProcessingImagesWG, &AllLinksSent, &WorkersWaiting)
	}
	WebProcessingWG.Wait()
	AllLinksSent = true
	ProcessingImagesWG.Wait()
	close(imageSendingCh)
	for img := range imageSendingCh {
		fmt.Println(img.src)
	}
}

func GetIcon(domain string, squareOnly bool, targetHeight, maxConcurrentProcesses int) {
	var wg sync.WaitGroup
	var wg2 sync.WaitGroup
	AllLinksSent := false
	WorkersWaiting := SafeCounter{}
	imageSendingCh := make(chan imageData, 32000)
	errors := make(chan error, 32000)
	imageReceivingCh := make(chan imageData, 32000)

	channels := channelHandling{urlChan: make(chan string, 32000), imageCh: make(chan imageData, 32000), imgDoneCh: make(chan imageData, 32000)}
	go processDomain(domain, channels, errors, &wg2, imageReceivingCh)

	for i := 0; i < maxConcurrentProcesses; i++ {
		go worker(&imageSendingCh, &imageReceivingCh, errors, &wg, &AllLinksSent, &WorkersWaiting)
	}
	wg2.Wait()
	AllLinksSent = true
	wg.Wait()
	close(imageSendingCh)
	for img := range imageSendingCh {
		fmt.Println(img.src)
	}
}

func worker(sendCh, receiveCh *chan imageData, errors chan error, wg *sync.WaitGroup, AllLinksSent *bool, WorkersWaiting *SafeCounter) {
	wg.Add(1)
	for receive := range *receiveCh {
		WorkersWaiting.mu.Lock()
		WorkersWaiting.count--
		WorkersWaiting.mu.Unlock()
		getImage(receive.src, &receive.domain, *sendCh, errors)
		WorkersWaiting.mu.Lock()
		WorkersWaiting.count++
		if WorkersWaiting.count == 0 && *AllLinksSent {
			close(*receiveCh)
			WorkersWaiting.mu.Unlock()
			break
		}
		WorkersWaiting.mu.Unlock()
	}
	defer wg.Done()
}

// GetIcon scrapes icons from the provided domain and returns the smallest icon taller than the target height, or the largest icon if none are taller).
// Parameters
//   - domains: The domains from which icons are to be scraped.
//   - squareOnly: If true, only square icons are considered.
//   - targetHeight: An integer representing the target height of the images to be fetched (height).
