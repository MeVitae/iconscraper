// package iconscraper provides a robust solution to get icons from domains.
//
// # Icon Sources
//
// - `/favicon.ico`
// - [Icon (`<link rel="icon" href="favicon.ico">`)](https://developer.mozilla.org/en-US/docs/Web/HTML/Attributes/rel#icon)
// - [Web app manifest (`<link rel="manifest" href="manifest.json">`)](https://developer.mozilla.org/en-US/docs/Web/Manifest)
// - [`link rel="shortcut icon"`](https://stackoverflow.com/questions/13211206/html5-link-rel-shortcut-icon)
// - [`link rel="apple-touch-icon"`](https://developer.mozilla.org/en-US/docs/Web/HTML/Attributes/rel#non-standard_values)
// - [`link rel="msapplication-TileImage"`](https://stackoverflow.com/questions/61686919/what-is-the-use-of-the-msapplication-tileimage-meta-tag)
// - [`link rel="mask-icon"`](http://microformats.org/wiki/existing-rel-values)
// - [`link rel="image_src"`](http://microformats.org/wiki/existing-rel-values) (also [this post](https://www.niallkennedy.com/blog/2009/03/enhanced-social-share.html))
// - [`meta itemprop="image"`](https://schema.org/image)
//
// # Other sources
//
// These aren't currently scraped, but might be of interest:
//
// - [`link rel="apple-touch-startup-image"`](http://microformats.org/wiki/existing-rel-values)
// - [`meta property="og:image"`](https://ogp.me/)
//
// # Get icons from multiple domains
//
//     import "github.com/MeVitae/iconscraper"
//
//     config := Config{
//         SquareOnly:            true,
//         TargetHeight:          128,
//         MaxConcurrentRequests: 32,
//         AllowSvg:              false,
//     }
//
//     domains := []string{"mevitae.com", "example.com", "gov.uk", "golang.org", "rust-lang.org"}
//
//     icons := iconscraper.GetIcons(config, domains)
//
//     for domain, icon := range icons {
//     	fmt.Println("Domain: " + domain + ", Icon URL: " + icon.URL)
//     }
//
// # Handle errors and warnings.
//
// Errors related to decoding images or resources not being found on a web server (but the connection
// being ok) will be reported as warnings instead of errors.
//
// By default, errors and warnings are only logged to the console. You can handle errors yourself by
// adding your own channel in the config, for example:
//
//     import "github.com/MeVitae/iconscraper"
//
//     config := Config{
//         SquareOnly:            true,
//         TargetHeight:          128,
//         MaxConcurrentRequests: 32,
//         AllowSvg:              false,
//         Errors:                make(chan error),
//     }
//
//     go func(){
//         for err := range config.Errors {
//             // Handle err
//         }
//     }()
//
//     domains := []string{"mevitae.com", "example.com", "gov.uk", "golang.org", "rust-lang.org"}
//
//     icons := iconscraper.GetIcons(config, domains)
//
//     for domain, icon := range icons {
//     	fmt.Println("Domain: " + domain + ", Icon URL: " + icon.URL)
//     }
//
// Warnings can be similarly handled using the `Warnings` field.
//
// # Get icon from a single domain
//
// Icons can be scraped for a single domain using `GetIcon`. Errors and warnings are handled in the
// same way.
package iconscraper

import (
	"bytes"
	"fmt"
	"image"
	"log"
	"regexp"

	"golang.org/x/net/html"
)

// logErrors logs all the errors sent on the channel to stderr
func logErrors(errors chan error) {
	for err := range errors {
		log.Println(err.Error())
	}
}

// logWarnings logs all the warnings sent on the channel to stderr
func logWarnings(errors chan error) {
	for err := range errors {
		log.Println("Warning:", err.Error())
	}
}

// Icon is an icon
type Icon struct {
	// URL is the source location from which the data was fetched or derived.
	URL string

	// Type is the sniffed MIME type of the image.
	Type string

	// Image holds the parsed image config. This is nil for SVGs (type image/svg+xml).
	ImageConfig image.Config

	// Source is the image source as downloaded.
	Source []byte
}

// Config is the config used for GetIcons and GetIcon.
type Config struct {
	// SquareOnly determines if only square icons are considered.
	SquareOnly bool

	// TargetHeight of the icon to be fetched. The shortest image larger than this size will be
	// returned and, if none are available, the tallest image smaller than this will be returned.
	TargetHeight int

	// AllowSvg allows SVGs to be returned. An SVG will always supersede a non-vector image.
	AllowSvg bool

	// MaxConcurrentRequests sets the maximum number of concurrent HTTP requests.
	MaxConcurrentRequests int

	// Errors is the channel for receiving errors.
	//
	// If nil, errors will instead by logged to the default logger.
	//
	// The channel must not block.
	Errors chan error

	// Warnings is the channel for receiving warning. Errors related to decoding images or resources
	// not being found on a web server (but the connection being ok) will be reported as warnings
	// instead of errors.
	//
	// If nil, warnings will instead by logged to the default logger.
	//
	// The channel must not block.
	Warnings chan error
}

// GetIcons scrapes icons from the provided domains concurrently and returns the results as a map from domain to the best image based on the given target.
//
// It finds the smallest icon taller than targetHeight or, if there are none, the tallest icon.
//
// If no icon is not found for a domain (or no square icon if squareOnly is true), that domain is omitted from the output map.
func GetIcons(config Config, domains []string) map[string]Icon {
	// Create error and warning handler channels if not provided. By default, these are consumed and logged.
	if config.Errors == nil {
		config.Errors = make(chan error)
		go logErrors(config.Errors)
		defer close(config.Errors)
	}
	if config.Warnings == nil {
		config.Warnings = make(chan error)
		go logWarnings(config.Warnings)
		defer close(config.Warnings)
	}

	// HTTP worker pool
	http := newHttpWorkerPool(config.MaxConcurrentRequests)
	defer http.close()

	// Channel to collect results
	results := make(chan processReturn)
	defer close(results)

	// Spawn a goroutine for every domain, these will be rate limited by the http pool.
	for _, domain := range domains {
		go processDomain(config, domain, http, results)
	}

	// Collect results
	resultMap := make(map[string]Icon, len(domains))
	for idx := 0; idx < len(domains); idx++ {
		res := <-results
		if res.result != nil {
			resultMap[res.domain] = *res.result
		}
	}
	return resultMap
}

// GetIcons scrapes icons from the provided domain and finds the smallest icon taller than targetHeight or, if there are none, the tallest icon.
//
// Errors that occur are sent to the config.Errors, unless it's nil, in which case, they are logged.
func GetIcon(config Config, domain string) *Icon {
	// Create error and warning handler channels if not provided. By default, these are consumed and logged.
	if config.Errors == nil {
		config.Errors = make(chan error)
		go logErrors(config.Errors)
		defer close(config.Errors)
	}
	if config.Warnings == nil {
		config.Warnings = make(chan error)
		go logWarnings(config.Warnings)
		defer close(config.Warnings)
	}

	// HTTP worker pool
	http := newHttpWorkerPool(config.MaxConcurrentRequests)
	defer http.close()

	// Channel to collect results
	results := make(chan processReturn, 1)
	defer close(results)

	go processDomain(config, domain, http, results)
	return (<-results).result
}

// processReturn is the output of processDomain
type processReturn struct {
	// domain is the domain that was processed.
	domain string

	// result holds the result, or nil if there isn't one.
	result *Icon
}

var domainNameRegexp = regexp.MustCompile(`^([a-zA-Z0-9_][a-zA-Z0-9_-]{0,64})(\.[a-zA-Z0-9_][a-zA-Z0-9_-]{0,64})*[\._]?$`)

// couldBeDomain returns false if domain definitely isn't a valid domain.
func couldBeDomain(domain string) bool {
	return len(domain) <= 512 && domainNameRegexp.MatchString(domain)
}

// processDomain is a worker function that processes getting images for a domain.
//
// It fetches HTML content from each URL, parses the HTML content, and extracts
// image information based on keys and values variables. It then picks the best
// image from the extracted images based on the `bestSize` parameter and sends
// the best image back on the result channel, or, if not image was found, it
// sends back a nil result.
func processDomain(
	config Config,
	domain string,
	http *httpWorkerPool,
	result chan processReturn,
) {
	// Check for obvious cases where the domain passed is invalid
	if !couldBeDomain(domain) {
		config.Errors <- fmt.Errorf("Invalid domain name %s", domain)
		result <- processReturn{
			domain: domain,
			result: nil,
		}
	}

	url := "https://" + domain
	httpResult := http.get(url)
	// Only check for network errors fetching, if it's an error page, that'll do.
	if httpResult.err != nil {
		config.Errors <- fmt.Errorf("Failed to get %s: %w", url, httpResult.err)
		result <- processReturn{
			domain: domain,
			result: nil,
		}
		return
	}

	// Parse the output HTML
	doc, err := html.Parse(bytes.NewReader(httpResult.body))
	if err != nil {
		config.Errors <- fmt.Errorf("Error parsing HTML from %s: %w", url, err)
		result <- processReturn{
			domain: domain,
			result: nil,
		}
		return
	}

	// Our requests will be now rooted at the domain we were redirected to.
	redirectDomain := httpResult.url.Host
	url = "https://" + redirectDomain

	workers := newImageWorkers(redirectDomain, http, config.Errors, config.Warnings)
	// Always check for `/favicon.ico`, it's not always linked from the HTML.
	workers.spawn(url + "/favicon.ico")
	// Spawn workers scraping all the linked icons
	getImagesFromHTML(doc, redirectDomain, &workers)

	// Pick the best size image from all the results
	result <- processReturn{
		domain: domain,
		result: pickBestImage(config, workers.results()),
	}
}
