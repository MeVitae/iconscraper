package scraper

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mat/besticon/ico"
)

// getImage fetches the image data from the specified URL, decodes it, and returns information about the image.
//
// It supports both regular image formats and ICO files.
//
// If the URL is not a valid image or an error occurs during the process, the function returns without an image or an error.
//
// Example:
//
//		img, err := getImage("https://example.com/image.jpg")
//	 if err != nil {
//	     // there was an error fetching the image
//	 } else if img == nil {
//	     // couldn't decode the image
//	 } else {
//	     // all good :)
//	 }
func getImage(url string, domain *string, receiveCh chan imageData, errorChan chan error) (imageData, bool) {

	if !isURL(url) {
		url = "https://" + url
	}
	body, ok, err := httpGet(url)
	if err != nil || !ok || !isImage(body) {
		errorChan <- err
		return imageData{}, false
	}

	// Check if the image is an ICO file
	if isICOFile(body) {
		// FIXME
		path := GenerateRandomString(10)
		saveImageToFile(body, path+"tmpIco.png")
		width, height, err := getImageInfo(path + "tmpIco.png")
		os.Remove(path + "tmpIco.png")
		if err == nil {
			size := size{width, height}
			receiveCh <- imageData{domain: *domain, src: url, size: size, data: body}

		}
		return imageData{}, false
	}
	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		errorChan <- err
		return imageData{}, false
	}
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	size := size{width, height}
	receiveCh <- imageData{domain: *domain, src: url, size: size, img: img, data: body}
	return imageData{src: url, size: size, img: img, data: body}, true
}

func encodeImageAsPNG(imagePath string) ([]byte, error) {
	// Read the ICO file data from file
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Decode the ICO file
	icoImage, err := ico.Decode(file)
	if err != nil {
		return nil, err
	}

	img := icoImage

	// Encode the image as PNG and get it as bytes
	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func saveImageToPNGBytes(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
func getImageInfo(filePath string) (width, height int, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	img, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}

	return img.Width, img.Height, nil
}
func saveImageToFile(imageData []byte, filename string) error {
	// Write the image data to the file
	err := ioutil.WriteFile(filename, imageData, 0644)
	if err != nil {
		return err
	}

	return nil
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
	if len(data) < 4 {
		return false
	}

	// Check the magic number for ICO files (0x00 0x00 0x01 0x00)
	return data[0] == 0x00 && data[1] == 0x00 && data[2] == 0x01 && data[3] == 0x00
}

// isImage checks whether the provided `data` represents an image.
//
// It uses http.DetectContentType to identify the content type of the data and checks if it starts with the prefix "image/".
//
// If the data represents an image, the function returns true; otherwise, it returns false.
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
	return strings.HasPrefix(http.DetectContentType(data), "image/")
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
//
// It chooses the smallest image taller than `targetHeight` or, if none exists, the largest image.
// If there are no input images, or `squareOnly` is true and none are square, returns `nil`.
//
//		images := []imageData{
//		    {name: "image1.jpg", size: size{1200, 800}},
//		    {name: "image2.jpg", size: size{1920, 1080}},
//		    {name: "image3.jpg", size: size{800, 600}},
//		}
//		targetHeight := 700
//		bestImage := pickBestImage(squareOnly, targetHeight, images)
//	 // bestImage.size.height == 800
func pickBestImage(squareOnly bool, targetHeight int, images []imageData) *imageData {
	// Track the largest image
	var largestImage *imageData
	// Track the smallest image larger than `targetHeight`
	var smallestOkImage *imageData

	for idx := range images {
		image := &images[idx]
		// Maybe skip non-square images
		if squareOnly && image.size.width != image.size.height {
			continue
		}

		// Update `smallestOkImage`
		diff := image.size.height - targetHeight
		if diff >= 0 {
			if smallestOkImage == nil || image.size.height < smallestOkImage.size.height {
				smallestOkImage = image
			}
		}

		// Update `largestImage`
		if largestImage == nil || image.size.height > largestImage.size.height {
			largestImage = image
		}
	}

	if smallestOkImage != nil {
		return smallestOkImage
	}
	return largestImage
}

func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}

	return string(b)
}
