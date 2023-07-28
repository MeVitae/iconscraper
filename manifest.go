package scraper

import (
	"encoding/json"
	"fmt"
)

// icon is a struct used to decode JSON data that holds information about an icon.
// It represents the properties of an icon, such as its source URL (Src), sizes, type, and density.
//
// Fields:
//
// - Src (string): The URL or file path of the icon.
// - Sizes (string): The size(s) of the icon, typically specified as width x height (e.g., "16x16").
// - Type (string): The MIME type or file format of the icon (e.g., "image/png").
// - Density (string): The pixel density descriptor of the icon (e.g., "1x").
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

// processManifest loads and parses a Web App Manifest
// (https://developer.mozilla.org/en-US/docs/Web/Manifest), and then spawns workers to process the
// icons defined.
func processManifest(domain, manifestUrl string, workers *imageWorkers) {
	httpResult := workers.http.get(manifestUrl)
	// Report an error
	if httpResult.err != nil {
		workers.errors <- httpResult.err
		return
	}
	// Ignore things that aren't 200 (they won't be the manifest!)
	if httpResult.status != 200 {
		workers.errors <- fmt.Errorf("Failed to get manifest %s: http %d", manifestUrl, httpResult.status)
		return
	}

	// Parse the manifest
	var manifest app
	err := json.Unmarshal(httpResult.body, &manifest)
	if err != nil {
		workers.errors <- fmt.Errorf("Failed to parse manifest %s: %w", manifestUrl, err)
		return
	}

	// Spawn an image worker for each icon
	for _, icon := range manifest.Icons {
		workers.spawn(getURL(domain, icon.Src))
	}
}
