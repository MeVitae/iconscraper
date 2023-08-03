package iconscraper

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"

	_ "github.com/mat/besticon/ico"
)

// imageWorkers is a collection of goroutines working on getting a parsing images
//
// It is not safe for concurrent use (though it does spawn concurrent workers).
type imageWorkers struct {
	// domain is the domain they're scraping images from.
	domain string
	// resultChan is the channel workers send succesfully parsed icons on.
	//
	// Each worker must send at most one result.
	resultChan chan Icon
	// failureChan is used to signal that a worker has failed.
	//
	// A worker must send a message on this channel if and only if it does not send a result.
	failureChan chan struct{}
	// numImages is the total number of workers spawned.
	numImages int
	// http is the underlying HTTP worker pool.
	http *httpWorkerPool
	// errors is the channel to send errors to, as many errors as needed may be sent.
	errors chan error

	// warnings channel to send warnings to.
	warnings chan error
}

func newImageWorkers(domain string, http *httpWorkerPool, errors chan error, warnings chan error) imageWorkers {
	return imageWorkers{
		domain:      domain,
		resultChan:  make(chan Icon),
		failureChan: make(chan struct{}),
		http:        http,
		errors:      errors,
		warnings:    warnings,
	}
}

// spawn a worker to collect and parse the image from url
//
// It is not safe for concurrent use (though it does spawn concurrent workers).
func (workers *imageWorkers) spawn(url string) {
	workers.numImages += 1
	go workers.getImage(url)
}

// results waits for a collects the results from all previously spawned workers.
//
// New jobs musn't be spawned after results has been called.
//
// It is not safe for concurrent use.
func (workers *imageWorkers) results() []Icon {
	results := make([]Icon, 0, workers.numImages)
	// For each image, we must have exactly one result or exactly one failure
	for idx := 0; idx < workers.numImages; idx++ {
		select {
		case result := <-workers.resultChan:
			results = append(results, result)
		case _ = <-workers.failureChan:
		}
	}
	close(workers.resultChan)
	return results
}

// getImage fetches the image data from the specified URL, decodes its config, and returns information about the image.
//
// If the URL is valid however does not return an image (or returns a non-200 status), it is ignored.
func (workers *imageWorkers) getImage(url string) {
	if !isURL(url) {
		url = "https://" + url
	}

	httpResult := workers.http.get(url)
	// Report an error
	if httpResult.err != nil {
		workers.errors <- fmt.Errorf("Failed to get icon %s: %w", url, httpResult.err)
		workers.failureChan <- struct{}{}
		return
	}
	// Ignore things that aren't 200 (they won't be the icons!)
	if httpResult.status != 200 {
		workers.warnings <- fmt.Errorf("Failed to get icon %s: http %d", url, httpResult.status)
		workers.failureChan <- struct{}{}
		return
	}
	// Check the content type, ingore if it's not an image.
	body := httpResult.body
	typ := http.DetectContentType(body)
	var img image.Config
	var err error
	// If it is not SVG decode the image config.
	if !isSVGImage(url) {
		if !strings.HasPrefix(typ, "image/") {
			workers.failureChan <- struct{}{}
			return
		}
		// Decode the image properties, and raise an error if this doesn't work.
		img, _, err = image.DecodeConfig(bytes.NewReader(body))
		if err != nil {
			workers.warnings <- fmt.Errorf("failed to decode image %s: %w", url, err)
			workers.failureChan <- struct{}{}
			return
		}
	}
	workers.resultChan <- Icon{
		URL:         url,
		Type:        typ,
		ImageConfig: img,
		Source:      body,
	}
}

// isSVGImage checks if a link is a svg file by looking at their format.
//
// isSVG := isSVGImage("/static/images/favicon.svg")
//
//	if isSVG {
//		fmt.Println("Image is SVG")
//	}else{
//
// fmt.Println("Image is not SVG")
// }
func isSVGImage(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".svg")
}

// pickBestImage picks the image from the given list that best matches the target size.
//
// It chooses the smallest image taller than `targetHeight` or, if none exists, the largest image.
// If there are no input images, or `squareOnly` is true and none are square, returns `nil`.
//
//		images := []imageData{
//		    {name: "image1.jpg", size: size{1200, 800}},
//		    {name: "image2.jpg", size: size{1920, 1080}},
//		    {name: "image3.jpg", size: size{800, 600}},
//		}
//		targetHeight := 700
//		bestImage := pickBestImage(squareOnly, targetHeight, images)
//	    // bestImage.img.Height == 800
func pickBestImage(config Config, images []Icon) *Icon {
	// Track the largest image
	var largestImage *Icon
	// Track the smallest image larger than `targetHeight`
	var smallestOkImage *Icon

	for idx := range images {
		image := &images[idx]
		// Always prefer SVG icons
		if config.AllowSvg && isSVGImage(image.URL) {
			return image
		}
		// Maybe skip non-square images
		if config.SquareOnly && image.ImageConfig.Width != image.ImageConfig.Height {
			continue
		}

		// Update `smallestOkImage`
		diff := image.ImageConfig.Height - config.TargetHeight
		if diff >= 0 {
			if smallestOkImage == nil || image.ImageConfig.Height < smallestOkImage.ImageConfig.Height {
				smallestOkImage = image
			}
		}

		// Update `largestImage`
		if largestImage == nil || image.ImageConfig.Height > largestImage.ImageConfig.Height {
			largestImage = image
		}
	}

	if smallestOkImage != nil {
		return smallestOkImage
	}
	return largestImage
}
