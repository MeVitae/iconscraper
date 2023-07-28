package scraper

import (
	"golang.org/x/net/html"
)

// attrValues is a list of HTML attribute values we can look for to find images
var iconRelValues = []string{"icon", "image_src", "apple-touch-icon", "shortcut icon", "img", "image"}

// getNodeAttr attempts to find the value of the attribute with the provided key.
//
// If no attribute is found, "" is returned.
func getNodeAttr(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func getURL(domain, path string) string {
	if len(path) == 0 {
		return "https://" + domain
	}
	if isURL(path) {
		return path
	}
	if path[0] == '/' {
		return "https://" + domain + path
	}
	return "https://" + domain + "/" + path
}

// getImagesFromHTML spawns image workers for all the icons referenced within a HTML page.
//
// - n: The HTML node to search for image-related attributes.
// - url: The base URL to resolve relative image URLs.
func getImagesFromHTML(node *html.Node, domain string, workers *imageWorkers) {
	// TODO: don't even traverse into the body!
	if node.Type == html.ElementNode && node.Data == "head" {
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == "link" {
				rel := getNodeAttr(c, "rel")
				if rel == "manifest" {
					// Parse link rel="manifest"
					if href := getNodeAttr(c, "href"); href != "" {
						processManifest(domain, getURL(domain, href), workers)
					}
				} else if contains(iconRelValues, rel) {
					// Process any icons links
					if href := getNodeAttr(c, "href"); href != "" {
						workers.spawn(getURL(domain, href))
					}
				}
			}
			if c.Type == html.ElementNode && c.Data == "meta" {
				itemprop := getNodeAttr(c, "itemprop")
				if itemprop == "image" {
					// Process any icons links
					if href := getNodeAttr(c, "content"); href != "" {
						workers.spawn(getURL(domain, href))
					}
				}
			}
		}
	} else {
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			getImagesFromHTML(c, domain, workers)
		}
	}
}

// contains checks if the target string is present in the provided list of strings.
func contains(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}
