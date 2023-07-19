package main

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
	data image.Image
}

var Keys = []string{"rel", "meta", "href", "content"}
var Values = []string{"icon", "image_src", "apple-touch-icon", "shortcut icon", "img", "image"}

func processImageGetting(urls chan string, bestSize int, rez chan processReturn) {
	for url := range urls {
		fmt.Println("Started working on:", url)

		htmlContent, err := getHTML(url)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		doc, err := html.Parse(strings.NewReader(htmlContent))
		if err != nil {
			log.Fatal(err)
		}

		images := make([]Images, 0)
		manifest := ""

		getImages(doc, &images, &manifest, url)
		if manifest != "" {
			jsonStr, err := getHTML(url + manifest)
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
			bestImage := pickBestImage(bestSize, images)
			rez <- processReturn{domain: url, picture: bestImage.src, dataIcon: bestImage.data}
		}
		bestImage := pickBestImage(bestSize, images)
		rez <- processReturn{domain: url, picture: bestImage.src, dataIcon: bestImage.data}
	}
}

func getImages(n *html.Node, images *[]Images, manifestSTR *string, url string) {
	localWG := sync.WaitGroup{}
	if n.Type == html.ElementNode && (n.Data == "link" || n.Data == "meta") {
		for _, a := range n.Attr {
			if a.Key == "rel" && a.Val == "manifest" {
				*manifestSTR = a.Val
			} else if contains(Keys, a.Key) || contains(Values, a.Val) {
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

func contains(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}
