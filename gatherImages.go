package scraper

import (
	"encoding/json"
	"fmt"
	"image"
	"log"
	"strconv"
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
//	src (string): The source URL or file path of the image.
//	size ([2]int): The size of the image specified as a fixed-size array with two elements representing width and height.
//	data (image.Image): The decoded image data represented as an image.Image type.
//	Source ([]byte): The raw image data as a byte list.
type imagesStruct struct {
	src    string
	size   [2]int
	data   image.Image
	Source []byte
}

// keys is a list of HTML attributes keys used for searching icons.
var keys = []string{"rel", "meta", "href", "content"}

// values is a list of values we can look for to find images
var values = []string{"icon", "image_src", "apple-touch-icon", "shortcut icon", "img", "image"}

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
func processImageGetting(urls chan string, bestSize int, rez chan processReturn) {
	for url := range urls {
		fmt.Println("Started working on:", url)

		htmlContent, err := getUrlData(url)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		doc, err := html.Parse(strings.NewReader(htmlContent))
		if err != nil {
			log.Fatal(err)
		}

		images := make([]imagesStruct, 0)
		manifest := ""

		getImages(doc, &images, &manifest, url)

		// If a manifest is found get the manifest data and pick the best image from there based on the target given.
		if manifest != "" {
			jsonStr, err := getUrlData(url + manifest)
			var app app
			err = json.Unmarshal([]byte(jsonStr), &app)
			if err != nil {
				fmt.Println("Error:", err)
			}

			// Look through all icons given within the manifest.
			for _, icon := range app.Icons {
				sizes := strings.Split(icon.Sizes, "x")
				width, _ := strconv.Atoi(sizes[0])
				height, _ := strconv.Atoi(sizes[1])
				size := [2]int{width, height}
				imgData, err := getImageData(url + icon.Src)
				if err == nil {
					images = append(images, imagesStruct{src: icon.Src, size: size, Source: imgData})
				}
			}

			// Get the best image from all images found and push it back the channel
			bestImage := pickBestImage(bestSize, images)
			result := Result{URL: bestImage.src, Image: bestImage.data, Source: bestImage.Source}
			rez <- processReturn{domain: url, result: result}
		}

		// Get the best image from all images found and push it back the channel
		bestImage := pickBestImage(bestSize, images)
		result := Result{URL: bestImage.src, Image: bestImage.data, Source: bestImage.Source}
		rez <- processReturn{domain: url, result: result}
	}
}

// getImages finds all images based on keys and values variables in the provided HTML node (n).
// It extracts image sizes and sets the sizes back to the main images variable.
//
// Parameters:
//
//	n (*html.Node): The HTML node to search for image-related attributes.
//	images (*[]imagesStruct): A pointer to a list of imagesStruct where the image information will be appended.
//	manifestSTR (*string): A pointer to a string to store the manifest attribute value, if found.
//	url (string): The base URL to resolve relative image URLs.
//
// Example:
//
//	var images []imagesStruct
//	var manifest string
//	getImages(htmlNode, &images, &manifest, "https://example.com")
//	// The images list now contains information about the images found in the HTML node.
func getImages(n *html.Node, images *[]imagesStruct, manifestSTR *string, url string) {
	localWG := sync.WaitGroup{}
	if n.Type == html.ElementNode && (n.Data == "link" || n.Data == "meta") {
		for _, a := range n.Attr {
			if a.Key == "rel" && a.Val == "manifest" {
				*manifestSTR = a.Val
			} else if contains(keys, a.Key) || contains(values, a.Val) {
				localWG.Add(1)
				go func(aVal string) {

					if isURL(aVal) {
						getImageSize(aVal, images)
					} else {
						getImageSize(url+aVal, images)
					}
					defer localWG.Done()
				}(a.Val)
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		getImages(c, images, manifestSTR, url)
	}

	localWG.Wait()
}

// contains checks if the target string is present in the provided list of strings.
// It returns true if the target is found, otherwise, it returns false.
//
// Parameters:
//   list ([]string): The list of strings to search in.
//   target (string): The string to find in the list.
//
// Returns:
//   (bool): True if the target is found in the list, false otherwise.
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
