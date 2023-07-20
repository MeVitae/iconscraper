package scraper

import (
	"encoding/json"
	"fmt"
	"image"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

// icon is a struct used to decode JSON data that holds information about an icon.
// It represents the properties of an icon, such as its source URL (Src), sizes, type, and density.
//
// Fields:
//
//	Src (string): The URL or file path of the icon.
//	Sizes (string): The size(s) of the icon, typically specified as width x height (e.g., "16x16").
//	Type (string): The MIME type or file format of the icon (e.g., "image/png").
//	Density (string): The pixel density descriptor of the icon (e.g., "1x").
type icon struct {
	Src     string `json:"src"`
	Sizes   string `json:"sizes"`
	Type    string `json:"type"`
	Density string `json:"density"`
}

// app is a struct used to decode JSON data that holds information about a web app's manifest.
// It represents the properties of the manifest, such as the app's name and a list of icons.
//
// Fields:
//
//	Name (string): The name of the web app specified in the manifest.
//	Icons ([]icon): A list of icon structs, representing the various icons defined in the manifest.
type app struct {
	Name  string `json:"name"`
	Icons []icon `json:"icons"`
}

// imagesStruct represents the information about an image, including its source URL or file path, size, decoded image data, and the raw image data.
// It is used to store details of images fetched from different sources, such as URLs or local files.
//
// Fields:
//
//	data (image.Image): The decoded image data represented as an image.Image type.
//	Source ([]byte): The raw image data as a byte list.
type imageData struct {
	// src url of the image
	src string
	// size of the image
	size size
	// img is the parsed image, if it was parsed
	img image.Image
	// data is the image data
	data []byte
}

type size struct {
	width  int
	height int
}

// attrKeys is a list of HTML attributes keys used for searching icons.
var attrKeys = []string{"rel", "meta", "href", "content"}

// attrValues is a list of HTML attribute values we can look for to find images
var attrValues = []string{"icon", "image_src", "apple-touch-icon", "shortcut icon", "img", "image"}

// processImageGetting is a worker function that processes getting images for a domain.
// It receives URLs from the `urls` channel, fetches HTML content from each URL, parses the HTML content,
// and extracts image information based on keys and values variables. It then picks the best image from the
// extracted images based on the `bestSize` parameter and pushes the best image back to the `rez` channel
// for further processing.
//
// Parameters:
//
//	urls (chan string): A channel that provides URLs to fetch and process images.
//	bestSize (int): The target size that the best image should match.
//	rez (chan processReturn): A channel to which the best image information is pushed for further processing.
//
// Example:
//
//	urls := make(chan string)
//	rez := make(chan processReturn)
//	bestSize := 1280
//
//	// Start worker goroutines
//	for i := 0; i < numWorkers; i++ {
//	    go processImageGetting(urls, bestSize, rez)
//	}
//
//	// Send URLs to the worker goroutines for processing
//	for _, url := range urlList {
//	    urls <- url
//	}
//	close(urls)
//
//	// Process the results from the worker goroutines
//	for i := 0; i < len(urlList); i++ {
//	    result := <-rez
//	    // Process the result (e.g., save it to a file or display it).
//	}
func processImageGetting(
	urls chan string,
	squareOnly bool,
	targetHeight int,
	rez chan processReturn,
	errors chan error,
	wg *sync.WaitGroup,
) {
	for url := range urls {
		res, err := processDomain(url, squareOnly, targetHeight, errors)
		if err != nil {
			errors <- err
		}
		if res != nil {
			rez <- *res
		}
		wg.Done()
	}
}

func processDomain(url string, squareOnly bool, targetHeight int, errors chan error) (*processReturn, error) {
	htmlContent, _, err := httpGet(url)
	if err != nil {
		return nil, fmt.Errorf("Error fetching page for %s: %w", url, err)
	}
	doc, err := html.Parse(strings.NewReader(string(htmlContent)))
	if err != nil {
		return nil, fmt.Errorf("Error parsing HTML from %s: %w", url, err)
	}

	images, manifest, err := getImages(doc, url)
	if err != nil {
		return nil, fmt.Errorf("Error getting images from %s: %w", url, err)
	}

	// If a manifest is found get the manifest data and pick the best image from there based on the target given.
	if manifest != "" {
		jsonData, ok, err := httpGet(url + manifest)
		if ok {
			var app app
			err = json.Unmarshal(jsonData, &app)
			if err != nil {
				errors <- fmt.Errorf("Error parsing manifest: %s", manifest)
			} else {
				// Look through all icons given within the manifest.
				for _, icon := range app.Icons {
					imgData, err := getImage(url + icon.Src)
					if imgData != nil {
						images = append(images, *imgData)
					}
					if err != nil {
						return nil, fmt.Errorf("Error fetching image from manifest %s: %w", icon.Src, err)
					}
				}
			}
		}
	}

	// Get the best image from all images found and push it back the channel
	bestImage := pickBestImage(squareOnly, targetHeight, images)
    if bestImage == nil {
        return nil, nil
    }
	result := Icon{URL: bestImage.src, Image: bestImage.img, Source: bestImage.data}
	return &processReturn{domain: url, result: result}, nil
}

// getImages finds all images based on keys and values variables in the provided HTML node (n).
//
// Parameters:
//
//	- n: The HTML node to search for image-related attributes.
//	- url: The base URL to resolve relative image URLs.
func getImages(n *html.Node, url string) (images []imageData, manifestPath string, err error) {
	if n.Type == html.ElementNode && (n.Data == "link" || n.Data == "meta") {
		for _, attr := range n.Attr {
			if attr.Key == "rel" && attr.Val == "manifest" {
				manifestPath = attr.Val
			} else if contains(attrKeys, attr.Key) || contains(attrValues, attr.Val) {
				imgUrl := attr.Val
				if !isURL(attr.Val) {
					imgUrl = url + imgUrl
				}
				var img *imageData
				img, err = getImage(imgUrl)
				if img != nil {
					images = append(images, *img)
				}
				if err != nil {
					return
				}
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		var childImages []imageData
		childImages, manifestPath, err = getImages(c, url)
		images = append(images, childImages...)
	}

	return
}

// contains checks if the target string is present in the provided list of strings.
//
// It returns true if the target is found, otherwise, it returns false.
//
// Example:
//   list := []string{"apple", "banana", "orange"}
//   target := "banana"
//   found := contains(list, target)
//   if found {
//       fmt.Println("The target is present in the list.")
//   } else {
//       fmt.Println("The target is not present in the list.")
//   }
func contains(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}
