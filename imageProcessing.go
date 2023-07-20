package scraper

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"net/http"
	"net/url"
	"strings"
)

// getImageSize fetches the image data from the specified URL, decodes it, and appends information about the image
// to the provided list of images ([]imagesStruct). It supports both regular image formats and ICO files.
// If the URL is not a valid image or an error occurs during the process, the function returns without appending any image data.
//
// Parameters:
//
//	url (string): The URL from which to fetch the image data.
//	images (*[]imagesStruct): A pointer to a list of imagesStruct to which the image information will be appended.
//
// Example:
//
//	var images []imagesStruct
//	url := "https://example.com/image.jpg"
//	getImageSize(url, &images)
//	// Now the images list contains information about the image fetched from the URL.
func getImageSize(url string, images *[]imagesStruct) {
	if !isURL(url) {
		url = "https://" + url
	}
	body, err := getImageData(url)
	if !isImage(body) {
		return
	}
	if err != nil {
		return
	}

	// Check if the image is an ICO file
	if isICOFile(body) {
		width, height, data, err := getICOSize(body)

		if err == nil {
			size := [2]int{width, height}
			img := image.NewAlpha(image.Rect(0, 0, width, height))
			for i := 0; i < len(data); i++ {
				img.Pix[i] = data[i]
			}
			*images = append(*images, imagesStruct{src: url, size: size, data: img, Source: data})

		}
		return
	}

	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return
	}

	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	if err == nil {
		size := [2]int{width, height}
		*images = append(*images, imagesStruct{src: url, size: size, data: img, Source: body})
	}

	return
}

// isICOFile checks whether the provided byte list `data` represents an ICO file.
// It returns true if the byte list has at least two elements (len(data) > 2) and the first two elements match the ICO file signature (0 and 0x01).
// Otherwise, it returns false.
//
// Parameters:
//
//	data ([]byte): The byte list to check for an ICO file signature.
//
// Returns:
//
//	(bool): True if the byte list represents an ICO file, false otherwise.
//
// Example:
//
//	data := []byte{0, 0x01, ...}
//	isICO := isICOFile(data)
//	if isICO {
//	    fmt.Println("The data represents an ICO file.")
//	} else {
//	    fmt.Println("The data is not an ICO file.")
//	}
func isICOFile(data []byte) bool {
	return len(data) > 2 && data[0] == 0 && data[1] == 0x01
}

// isURL checks whether the provided string `str` is a valid URL.
// It uses Go's url.Parse and checks if it returns any error to determine if the URL is valid.
// If the URL is valid and contains both a scheme and a host, the function returns true; otherwise, it returns false.
//
// Parameters:
//
//	str (string): The string to check if it is a valid URL.
//
// Returns:
//
//	(bool): True if the string is a valid URL, false otherwise.
//
// Example:
//
//	urlStr := "https://example.com"
//	isValid := isURL(urlStr)
//	if isValid {
//	    fmt.Println("The URL is valid.")
//	} else {
//	    fmt.Println("The URL is not valid.")
//	}
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

// isImage checks whether the provided byte list `data` represents an image.
// It uses http.DetectContentType to identify the content type of the data and checks if it starts with the prefix "image/".
// If the data represents an image, the function returns true; otherwise, it returns false.
//
// Parameters:
//
//	data ([]byte): The byte list to check for an image content type.
//
// Returns:
//
//	(bool): True if the byte list represents an image, false otherwise.
//
// Example:
//
//	imageData := []byte{255, 216, ...}
//	isImg := isImage(imageData)
//	if isImg {
//	    fmt.Println("The data represents an image.")
//	} else {
//	    fmt.Println("The data is not an image.")
//	}
func isImage(data []byte) bool {
	contentType := http.DetectContentType(data)
	if strings.HasPrefix(contentType, "image/") {
		return true
	}

	return false
}

// getICOSize extracts the size and alpha channel data of the first valid icon entry from the given ICO file data.
//
// Parameters:
//
//	data ([]byte): The ICO file data as a byte list.
//
// Returns:
//
//	(int, int, []byte, error): The width and height of the icon, the alpha channel data as a byte list, and any error encountered during extraction.
//
// Example:
//
//	data := []byte{0, 0, 1, 0, 3, 0, 0, 0, 16, 0, 16, 0, 1, 0, 32, 32, ...}
//	width, height, alphaData, err := getICOSize(data)
//	if err != nil {
//	    fmt.Println("Error:", err)
//	} else {
//	    fmt.Printf("Icon size: %dx%d\n", width, height)
//	    // Use alphaData to manipulate the icon's alpha channel.
//	}
func getICOSize(data []byte) (int, int, []byte, error) {
	// ICO file header
	const (
		iconDirEntrySize         = 16
		iconDirEntryWidthOffset  = 6
		iconDirEntryHeightOffset = 8
	)

	// Check ICO file signature
	if len(data) < 6 || data[0] != 0 || data[1] != 0 || data[2] != 1 || data[3] != 0 {
		return 0, 0, nil, fmt.Errorf("Invalid ICO file format")
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
			// Find the offset of the alpha channel data
			imgOffset := int(data[offset+12])
			imgSize := int(data[offset+8]) // Size of the image data
			alphaData := data[imgOffset : imgOffset+imgSize]

			return width, height, alphaData, nil
		}
	}

	return 0, 0, nil, fmt.Errorf("No valid icon size found")
}

// pickBestImage picks the image from the given list that best matches the target size.
// It calculates the absolute difference between each image's width and the target size and selects the image
// with the smallest difference as the best match.
//
// Parameters:
//
//	target (int): The target width that the best image should match.
//	images ([]imagesStruct): A list of imagesStruct representing a collection of images with their sizes.
//
// Returns:
//
//	(imagesStruct): The best-matching image based on the target size.
//
// Example:
//
//	images := []imagesStruct{
//	    {name: "image1.jpg", size: [2]int{1200, 800}},
//	    {name: "image2.jpg", size: [2]int{1920, 1080}},
//	    {name: "image3.jpg", size: [2]int{800, 600}},
//	}
//	targetSize := 1280
//	bestImage := pickBestImage(targetSize, images)
//	fmt.Println("Best image:", bestImage.name)
func pickBestImage(target int, images []imagesStruct) imagesStruct {
	bestImage := imagesStruct{}
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
