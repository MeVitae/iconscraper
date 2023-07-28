# Icon-Scraper Package Documentation

`icon-scraper` is a Go package that provides a easy way to get logos from domains and find best target sizes.

## Description

`icon-scraper` is a Go package that provides a robust, concurent solution for scraping and processing images from defined domains. It fetches the images concurrently, identifying and returning the one that best matches your target size from each domain.

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
import "github.com/MeVitae/icon-scraper"

// Define the list of domains you want to get the logo from.
domains := []string{"https://example.com", "https://example.net", "https://example.org"}

// Call GetIcons function with:
// 1. Domains list 
// 2. Square Only Requirement 
// 3. Target Height 
// 4. Max Concurrent processes (Set this based on your network!)
icons := scraper.GetIcons(domains, false, 100, 4)

// Iterate over the return map to use the scraped icons
for domain, icon := range icons {
	fmt.Println("Domain:",domain,", Icon URL:", icon.URL)
}
```

### Get Icon from a single domain:

```go
import "github.com/MeVitae/icon-scraper"

// Define the domain you want to get the logo from.
domain := "https://example.com"

// Call GetIcons function with:
// 1. Domains list 
// 2. Square Only Requirement 
// 3. Target Height 
// 4. Max Concurrent processes (Set this based on your network!)
icon := scraper.GetIcon(domain, false, 64, 4)

// Use the returned icon
fmt.Println("Domain:",domain,", Icon URL:", icon.URL)
```
## Notes
### Ico files
In the case of ico images the returned Icon struct will not include a image.Image only source!
### Target size
It chooses the smallest image taller than `targetHeight` or, if none exists, the largest image.
