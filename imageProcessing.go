package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/disintegration/imaging"
)

func getImageSize(url string, images *[]Images) {
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
			*images = append(*images, Images{src: url, size: size, data: img})

		}
		return
	}

	img, err := imaging.Decode(bytes.NewReader(body))
	if err != nil {
		return
	}

	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	if err == nil {
		size := [2]int{width, height}
		*images = append(*images, Images{src: url, size: size, data: img})
	}

	return
}

func isICOFile(data []byte) bool {
	return len(data) > 2 && data[0] == 0 && data[1] == 0x01
}

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

func isImage(data []byte) bool {
	contentType := http.DetectContentType(data)
	if strings.HasPrefix(contentType, "image/") {
		return true
	}

	return false
}

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

func saveImageAsPNG(img image.Image, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Encode the image as PNG and write it to the file
	err = png.Encode(file, img)
	if err != nil {
		return err
	}

	return nil
}

func pickBestImage(target int, images []Images) Images {
	bestImage := Images{}
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
