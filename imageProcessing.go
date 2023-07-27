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

	_ "github.com/mat/besticon/ico"
)

// getImage fetches the image data from the specified URL, decodes it, and returns information about the image.
//
// It supports both regular image formats and ICO files.
//
// If the URL is not a valid image or an error occurs during the process, the function returns without an image or an error.
//
// Example:
//
//	img, err := getImage("https://example.com/image.jpg")
//	if err != nil {
//	    // there was an error fetching the image
//	} else if img == nil {
//	    // couldn't decode the image
//	} else {
//	    // all good :)
//	}
func (workers *imageWorkers) getImage(url string) {
	if !isURL(url) {
		url = "https://" + url
	}

	httpResult := workers.http.get(url)
	if httpResult.err != nil {
		workers.errors <- httpResult.err
		workers.failureChan <- struct{}{}
		return
	}
	body := httpResult.body
	if httpResult.status != 200 || !isImage(body) {
		workers.failureChan <- struct{}{}
		return
	}

	// Check if the image is an ICO file
	if isICOFile(body) {
		// FIXME
		path := GenerateRandomString(10)
		saveImageToFile(body, path+"tmpIco.png")
		width, height, err := getImageInfo(path + "tmpIco.png")
		os.Remove(path + "tmpIco.png")
		if err != nil {
			workers.errors <- fmt.Errorf("failed to get ico dimensions of %s: %w", url, err)
			workers.failureChan <- struct{}{}
			return
		}
		size := size{width, height}
		workers.resultChan <- imageData{domain: workers.domain, src: url, size: size, data: body}
		return
	}

	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		// TODO: maybe these should be warnings not errors
		workers.errors <- fmt.Errorf("failed to decode image %s: %w", url, err)
		workers.failureChan <- struct{}{}
		return
	}
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	size := size{width, height}
	workers.resultChan <- imageData{domain: workers.domain, src: url, size: size, img: img, data: body}
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
