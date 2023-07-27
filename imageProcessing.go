package scraper

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

// getImage fetches the image data from the specified URL, decodes it, and returns information about the image.
//
// It supports both regular image formats and ICO files.
//
// If the URL is not a valid image or an error occurs during the process, the function returns without an image or an error.
//
// Example:
//
//	img, err := getImage("https://example.com/image.jpg")
//	if err != nil {
//	    // there was an error fetching the image
//	} else if img == nil {
//	    // couldn't decode the image
//	} else {
//	    // all good :)
//	}
func (workers *imageWorkers) getImage(url string) {
	if !isURL(url) {
		url = "https://" + url
	}

	httpResult := workers.http.get(url)
	if httpResult.err != nil {
		workers.errors <- httpResult.err
		workers.failureChan <- struct{}{}
		return
	}
	body := httpResult.body
	if httpResult.status != 200 || !isImage(body) {
		workers.failureChan <- struct{}{}
		return
	}

	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		// TODO: maybe these should be warnings not errors
		workers.errors <- fmt.Errorf("failed to decode image %s: %w", url, err)
		workers.failureChan <- struct{}{}
		return
	}
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	size := size{width, height}
	workers.resultChan <- imageData{domain: workers.domain, src: url, size: size, img: img, data: body}
}

// isImage checks whether the provided `data` represents an image.
//
// It uses http.DetectContentType to identify the content type of the data and checks if it starts with the prefix "image/".
//
// If the data represents an image, the function returns true; otherwise, it returns false.
//
// Example:
//
//	imageData := []byte{255, 216, ...}
//	isImg := isImage(imageData)
//	if isImg {
//	    fmt.Println("The data represents an image.")
//	} else {
//	    fmt.Println("The data is not an image.")
//	}
func isImage(data []byte) bool {
	return strings.HasPrefix(http.DetectContentType(data), "image/")
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
//	 // bestImage.size.height == 800
func pickBestImage(squareOnly bool, targetHeight int, images []imageData) *imageData {
	// Track the largest image
	var largestImage *imageData
	// Track the smallest image larger than `targetHeight`
	var smallestOkImage *imageData

	for idx := range images {
		image := &images[idx]
		// Maybe skip non-square images
		if squareOnly && image.size.width != image.size.height {
			continue
		}

		// Update `smallestOkImage`
		diff := image.size.height - targetHeight
		if diff >= 0 {
			if smallestOkImage == nil || image.size.height < smallestOkImage.size.height {
				smallestOkImage = image
			}
		}

		// Update `largestImage`
		if largestImage == nil || image.size.height > largestImage.size.height {
			largestImage = image
		}
	}

	if smallestOkImage != nil {
		return smallestOkImage
	}
	return largestImage
}
