# Icon-Scraper Package Documentation

`iconscraper` is a Go package that provides a easy way to get logos from domains and find best target sizes.

## Description

`iconscraper` is a Go package that provides a robust, concurent solution for scraping and processing images from defined domains. It fetches the images concurrently, identifying and returning the one that best matches your target size from each domain.

The package is highly performant, utilizing worker goroutines and channels for efficient processing. It offers options to filter square images and define a target size for the images. 

## Icon Sources

- `/favicon.ico`
- [Icon (`<link rel="icon" href="favicon.ico">`)](https://developer.mozilla.org/en-US/docs/Web/HTML/Attributes/rel#icon)
- [Web app manifest (`<link rel="manifest" href="manifest.json">`)](https://developer.mozilla.org/en-US/docs/Web/Manifest)
- [`link rel="shortcut icon"`](https://stackoverflow.com/questions/13211206/html5-link-rel-shortcut-icon)
- [`link rel="apple-touch-icon"`](https://developer.mozilla.org/en-US/docs/Web/HTML/Attributes/rel#non-standard_values)
- [`link rel="msapplication-TileImage"`](https://stackoverflow.com/questions/61686919/what-is-the-use-of-the-msapplication-tileimage-meta-tag)
- [`link rel="mask-icon"`](http://microformats.org/wiki/existing-rel-values)
- [`link rel="image_src"`](http://microformats.org/wiki/existing-rel-values) (also [this post](https://www.niallkennedy.com/blog/2009/03/enhanced-social-share.html))
- [`meta itemprop="image"`](https://schema.org/image)

### Other sources

These aren't currently scraped, but might be of interest:

- [`link rel="apple-touch-startup-image"`](http://microformats.org/wiki/existing-rel-values)
- [`meta property="og:image"`](https://ogp.me/)

## Usage

### Get Icons from multiple domains:

```go
import "github.com/MeVitae/iconscraper"

// Receive errors when happening (you can use this to save them and analyse what does not work).
func handleErrors(errors chan error) {
	for err := range errors {
		fmt.Fprintln(os.Stderr, err.Error())
		// do some processing on the errors
	}
}

// Receive warnings when happening (you can use this to save them and analyse what does not work).
func handleWarnings(warnings chan error) {
	for err := range warnings {
		fmt.Fprintln(os.Stderr, err.Error())
		// do some processing on the errors
	}
}


// Create the config on which you will be looking for icons.
// If Errors or Warnings fields left empty, a channel for 
// logging will automatically be created and errors will be printed!
config := iconscraper.Config{
	SquareOnly:             true,
	TargetHeight:           128,
	MaxConcurrentProcesses: 20,
	AllowSvg:               false,
	Errors:	            make(chan error, 32000),
	Warnings:               make(chan error, 32000),
}

// create go routines for handling errors and warnings.
go handleErrors(config.Errors)
go handleWarnings(config.Warnings)

// Define the list of domains you want to get the logo from.
domains := []string{"https://example.com", "https://example.net", "https://example.org"}

//Create a go routine for

// Call GetIcons function with:
// 1. Domains list 
// 2. Square Only Requirement 
// 3. Target Height 
// 4. Max Concurrent processes (Set this based on your network!)
icons := iconscraper.GetIcons(domains, config)

// Iterate over the return map to use the scraped icons
for domain, icon := range icons {
	fmt.Println("Domain:",domain,", Icon URL:", icon.URL)
}
```

### Get Icon from a single domain:

```go
import "github.com/MeVitae/iconscraper"

// Create the config on which you will be looking for icons.
config := iconscraper.Config{
	SquareOnly:             true,
	TargetHeight:           128,
	MaxConcurrentProcesses: 20,
	AllowSvg:               false,
}

// Define the domain you want to get the logo from.
domain := "https://example.com"

// Call GetIcons function with:
// 1. Domains list 
// 2. Square Only Requirement 
// 3. Target Height 
// 4. Max Concurrent processes (Set this based on your network!)
icon := iconscraper.GetIcon(domain, config)

// Use the returned icon
fmt.Println("Domain:",domain,", Icon URL:", icon.URL)
```
## Notes
### Ico files
In the case of ico images the returned Icon struct will not include a image.Image only source!
### Target size
It chooses the smallest image taller than `targetHeight` or, if none exists, the largest image.
