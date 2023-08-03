# Icon-Scraper Package Documentation

`iconscraper` is a Go package that provides a robust solution to get icons from domains.

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

### Get icons from multiple domains

```go
import "github.com/MeVitae/iconscraper"

config := Config{
    SquareOnly:            true,
    TargetHeight:          128,
    MaxConcurrentRequests: 32,
    AllowSvg:              false,
}

domains := []string{"mevitae.com", "example.com", "gov.uk", "golang.org", "rust-lang.org"}

icons := iconscraper.GetIcons(config, domains)

for domain, icon := range icons {
	fmt.Println("Domain: " + domain + ", Icon URL: " + icon.URL)
}
```

### Handle errors and warnings.

Errors related to decoding images or resources not being found on a web server (but the connection
being ok) will be reported as warnings instead of errors.

By default, errors and warnings are only logged to the console. You can handle errors yourself by
adding your own channel in the config, for example:

```go
import "github.com/MeVitae/iconscraper"

config := Config{
    SquareOnly:            true,
    TargetHeight:          128,
    MaxConcurrentRequests: 32,
    AllowSvg:              false,
    Errors:                make(chan error),
}

go func(){
    for err := range config.Errors {
        // Handle err
    }
}()

domains := []string{"mevitae.com", "example.com", "gov.uk", "golang.org", "rust-lang.org"}

icons := iconscraper.GetIcons(config, domains)

for domain, icon := range icons {
	fmt.Println("Domain: " + domain + ", Icon URL: " + icon.URL)
}
```

Warnings can be similarly handled using the `Warnings` field.

### Get icon from a single domain

Icons can be scraped for a single domain using `GetIcon`. Errors and warnings are handled in the
same way.

