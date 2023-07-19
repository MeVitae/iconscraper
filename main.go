package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/disintegration/imaging"
	"golang.org/x/net/html"
)

type result struct {
	domain string
	icon   string
}

type processReturn struct {
	domain  string
	picture string
}

var maxConcurrentProcesses = 2

func main() {
	domains := readDomains()
	urls := make(chan string, len(domains))
	returns := make(chan processReturn)

	workers := len(domains)
	if workers > maxConcurrentProcesses {
		workers = maxConcurrentProcesses
	}

	for worker := 0; worker < workers; worker++ {
		go processImageGetting(urls, 120, returns)
	}

	for _, domain := range domains {
		urls <- domain.domain
	}

	results := make(map[string]string, len(domains))
	for result := range returns {
		results[result.domain] = result.picture
		if len(results) == len(domains) {
			break
		}
	}
	close(urls)

	fmt.Println("------ Finished -------")
	fmt.Println(results)
}

var wg sync.WaitGroup

type Icon struct {
	Src     string `json:"src"`
	Sizes   string `json:"sizes"`
	Type    string `json:"type"`
	Density string `json:"density"`
}

type App struct {
	Name  string `json:"name"`
	Icons []Icon `json:"icons"`
}

type Images struct {
	src  string
	size [2]int
}

func processImageGetting(urls chan string, bestSize int, rez chan processReturn) {
	for url := range urls {
		fmt.Println("Started working on:", url)

		htmlContent, err := getHTML(getFinalUrl(url))
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		doc, err := html.Parse(strings.NewReader(htmlContent))
		if err != nil {
			log.Fatal(err)
		}

		images := make([]Images, 0)
		manifest := ""

		getImages(doc, &images, &manifest, getFinalUrl(url))
		if manifest != "" {
			jsonStr, err := getManifestData(getFinalUrl(url) + manifest)
			var app App
			err = json.Unmarshal([]byte(jsonStr), &app)
			if err != nil {
				fmt.Println("Error:", err)
			}

			for _, icon := range app.Icons {
				sizes := strings.Split(icon.Sizes, "x")
				width, _ := strconv.Atoi(sizes[0])
				height, _ := strconv.Atoi(sizes[1])
				size := [2]int{width, height}
				images = append(images, Images{src: icon.Src, size: size})
			}
			rez <- processReturn{domain: url, picture: pickBestImage(bestSize, images).src}
		}
		rez <- processReturn{domain: url, picture: pickBestImage(bestSize, images).src}
	}
}

func pickBestImage(target int, images []Images) Images {
	bestImage := Images{}
	minDiff := 999

	for _, image := range images {
		diff := int(math.Abs(float64(image.size[0]) - float64(target)))
		if diff < minDiff {
			minDiff = diff
			bestImage = image
		}
	}

	return bestImage
}
func getImages(n *html.Node, images *[]Images, manifestSTR *string, url string) {
	localWG := sync.WaitGroup{}
	if n.Type == html.ElementNode && (n.Data == "link" || n.Data == "meta") {
		for _, a := range n.Attr {
			if a.Key == "rel" && a.Val == "manifest" {
				*manifestSTR = a.Val
			} else if (a.Key == "rel" || a.Key == "meta" || a.Key == "href" || a.Key == "content") || (a.Val == "icon" || a.Val == "image_src" || a.Val == "apple-touch-icon" || a.Val == "shortcut icon" || strings.Contains(a.Val, "img") || strings.Contains(a.Val, "image")) {
				localWG.Add(1) // Increment the local WaitGroup counter
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

func getManifestData(url string) (string, error) {
	data, err := sendGetRequestWithUserAgent(url)
	if err != nil {
		return "", fmt.Errorf("Failed to retrieve manifest data: %s", err)
	}

	return data, nil
}

func getHTML(url string) (string, error) {
	data, err := sendGetRequestWithUserAgent(url)
	if err != nil {
		return "", fmt.Errorf("Failed to retrieve manifest data: %s", err)
	}

	return data, nil
}

func getFinalUrl(domain string) (finalURL string) {
	resp, err := http.Get("https://" + domain)
	if err != nil {
		log.Fatalf("http.Get => %v", err.Error())
	}
	finalURL = resp.Request.URL.String()
	return
}

func getImageSize(url string, images *[]Images) {
	body, err := sendGetRequestWithUserAgentIMGS(url)
	if !isImage(body) {
		return
	}
	if err != nil {
		return
	}

	// Check if the image is an ICO file
	if isICOFile(body) {
		width, height, err := getICOSize(body)

		if err == nil {
			size := [2]int{width, height}
			*images = append(*images, Images{src: url, size: size})
		}
	}

	img, err := imaging.Decode(bytes.NewReader(body))
	if err != nil {
		return
	}

	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	if err == nil {
		size := [2]int{width, height}
		*images = append(*images, Images{src: url, size: size})
	}

	return
}

func isICOFile(data []byte) bool {
	return len(data) > 2 && data[0] == 0 && data[1] == 0x01
}

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

func sendGetRequestWithUserAgent(url string) (string, error) {
	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.6.1 Safari/605.1.15"

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}

	req, err := http.NewRequest("GET", url, nil)
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

func sendGetRequestWithUserAgentIMGS(url string) ([]byte, error) {
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

func isImage(data []byte) bool {
	contentType := http.DetectContentType(data)
	if strings.HasPrefix(contentType, "image/") {
		return true
	}

	return false
}

func getICOSize(data []byte) (int, int, error) {
	// ICO file header
	const (
		iconDirEntrySize         = 16
		iconDirEntryWidthOffset  = 6
		iconDirEntryHeightOffset = 8
	)

	// Check ICO file signature
	if len(data) < 6 || data[0] != 0 && data[1] != 0 || data[2] != 1 && data[3] != 0 {
		return 0, 0, fmt.Errorf("Invalid ICO file format")
	}

	// Number of icon directory entries
	iconCount := int(data[4])

	// Iterate through each icon directory entry
	for i := 0; i < iconCount; i++ {
		offset := 6 + (i * iconDirEntrySize)

		// Retrieve width and height from the icon directory entry
		width := int(data[offset+iconDirEntryWidthOffset])
		height := int(data[offset+iconDirEntryHeightOffset])

		// Check if the icon has dimensions specified (non-zero)
		if width > 0 && height > 0 {
			return width, height, nil
		}
	}

	return 0, 0, fmt.Errorf("No valid icon size found")
}
