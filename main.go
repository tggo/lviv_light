package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/go-resty/resty/v2"
	"gocv.io/x/gocv"
)

func main() {
	url := getUrlToImage()

	fmt.Println("URL to image: ", url)

	imagePath := "/tmp/image.jpg"

	downloadImageFromSite(url, imagePath)

	img := gocv.IMRead(imagePath, gocv.IMReadColor)
	if img.Empty() {
		fmt.Println("Error reading image")
		return
	}
	defer img.Close()

	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	// Apply Gaussian Blur to reduce noise
	gaussian := gocv.NewMat()
	defer gaussian.Close()
	gocv.GaussianBlur(gray, &gaussian, image.Pt(1, 1), 0, 0, gocv.BorderDefault)

	// Apply adaptive thresholding
	thresh := gocv.NewMat()
	defer thresh.Close()
	gocv.AdaptiveThreshold(gaussian, &thresh, 255, gocv.AdaptiveThresholdMean, gocv.ThresholdBinaryInv, 15, 10)

	// Detect horizontal lines
	horizontalKernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(40, 1))
	defer horizontalKernel.Close()
	horizontal := gocv.NewMat()
	defer horizontal.Close()
	gocv.MorphologyEx(thresh, &horizontal, gocv.MorphOpen, horizontalKernel)
	horizontalLines := detectLines(horizontal)

	// Detect vertical lines
	verticalKernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(1, 40))
	defer verticalKernel.Close()
	vertical := gocv.NewMat()
	defer vertical.Close()
	gocv.MorphologyEx(thresh, &vertical, gocv.MorphOpen, verticalKernel)
	verticalLines := detectLines(vertical)

	// Extract the points of intersection
	horizontalPositions := extractLinePositions(horizontalLines, true)
	verticalPositions := extractLinePositions(verticalLines, false)

	// Initialize the matrix
	var matrix [][]string

	// Define color ranges (with some flexibility)
	greenLower := color.RGBA{R: 35, G: 40, B: 40, A: 255}
	greenUpper := color.RGBA{R: 85, G: 255, B: 255, A: 255}
	orangeLower := color.RGBA{R: 10, G: 100, B: 100, A: 255}
	orangeUpper := color.RGBA{R: 25, G: 255, B: 255, A: 255}

	gaussianImg := gocv.NewMat()
	gocv.GaussianBlur(img, &gaussianImg, image.Pt(5, 5), 0, 0, gocv.BorderDefault)
	defer gaussianImg.Close()

	// Extract and detect color from each cell
	for i := 0; i < len(horizontalPositions)-1; i++ {
		var row []string
		for j := 0; j < len(verticalPositions)-1; j++ {
			cell := gaussianImg.Region(image.Rect(verticalPositions[j], horizontalPositions[i], verticalPositions[j+1], horizontalPositions[i+1]))
			colorDetected := detectColor(cell, greenLower, greenUpper, orangeLower, orangeUpper)
			row = append(row, colorDetected)
		}
		matrix = append(matrix, row)
	}

	// delete rows 0 and 1
	matrix = append(matrix[:0], matrix[2:]...)

	// delete column 0
	// for i := 0; i < len(matrix); i++ {
	// 	matrix[i] = append(matrix[i][:0], matrix[i][1:]...)
	// }

	if len(matrix[0]) != 24 {
		panic("wrong parsing column")
	}

	// Print the matrix
	for i, row := range matrix {
		fmt.Printf("Row %d: ", i+1)
		fmt.Print(strings.Join(row, ""))
		fmt.Println()
	}

	// work with row 1 - is 1.2 group
	// each element array is one hour of the day
	// if the element is green, then the hour is available
	// if the element is orange, then the hour is not available
	// write a function that takes the matrix and returns the available hours

	availableHours := getAvailableHours(matrix[1])
	fmt.Printf("Available hours: %v is %dh of 24h\n", availableHours, len(availableHours))

	// concatenate the available hours into ranges
	// e.g. 0, 1, 2, 3, 4, 5, 6, 7, 8, 9 to 0-10
	// e.g. 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 11, 12 to 0-10, 11-13
	// e.g. 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 11, 12, 13, 14 to 0-10, 11-15
	// e.g. 0, 5, 6, 7, 8, 9, 11, 12, 13, 14 to 0-1, 5-10, 11-15 not 0-0, 5-9, 11-14
	// e.g. 0 4 5 6 7 8 9 10 11 12 17 18 19 20 21 23 to 0-1, 4-13, 17-22, 23-0

	// write a function that takes the available hours and returns the ranges
	ranges := getRanges(availableHours)
	fmt.Println("Available ranges: ", ranges)
}

func downloadImageFromSite(url string, path string) {
	client := resty.New()
	_, err := client.R().SetOutput(path).Get(url)
	if err != nil {
		panic(err)
	}
}

func getUrlToImage() string {

	client := resty.New()
	resp, err := client.R().Get("https://api.loe.lviv.ua/api/pages?page=1&synonym=power-top")
	if err != nil {
		panic(err)
	}

	// u003Cp\u003E\u003Cimg src=\u0022https:\/\/api.loe.lviv.ua\/media\/6696ae15ed274_IMG_20240716_194813_632.jpg\u0022 alt=\u0022\u0022
	// get image from response use Regular Expression
	exp := `https:\\\/\\\/api.loe.lviv.ua\\\/media\\\/[a-zA-Z0-9_]+.jpg`
	r := regexp.MustCompile(exp)
	urls := r.FindAllString(string(resp.Body()), -1)

	url := ""
	if len(urls) > 0 {
		url = urls[0]
	}

	url = strings.ReplaceAll(url, "\\", "")

	return url
}

func getLastImageToParse() string {
	// get a list of files in the directory by mask *.jpg
	fileList, err := filepath.Glob("*.jpg")
	if err != nil {
		panic(err)
	}

	// sort-by-date, the newest file is the first
	sort.Slice(fileList, func(i, j int) bool {
		infoI, errStat := os.Stat(fileList[i])
		if errStat != nil {
			return false
		}

		infoJ, errStat := os.Stat(fileList[j])
		if errStat != nil {
			return false
		}

		return infoI.ModTime().After(infoJ.ModTime())
	})

	// get the first file from the list
	return fileList[0]
}

func getRanges(availableHours []int) []string {
	var ranges []string
	var start int
	var end int
	for i, hour := range availableHours {
		if i == 0 {
			start = hour
			end = hour
			continue
		}

		if hour == end+1 {
			end = hour
		} else {
			ranges = append(ranges, fmt.Sprintf("%d-%d", start, end+1))
			start = hour
			end = hour
		}
	}

	ranges = append(ranges, fmt.Sprintf("%d-%d", start, end+1))
	return ranges
}

func getAvailableHours(row []string) []int {
	var availableHours []int
	for i, cell := range row {
		if cell == "+" {
			availableHours = append(availableHours, i)
		}
	}

	return availableHours
}

// Function to detect color
// Function to detect color
func detectColor(img gocv.Mat, greenLower, greenUpper, orangeLower, orangeUpper color.RGBA) string {
	hsv := gocv.NewMat()
	defer hsv.Close()
	gocv.CvtColor(img, &hsv, gocv.ColorBGRToHSV)
	greenMask := gocv.NewMat()
	defer greenMask.Close()
	orangeMask := gocv.NewMat()
	defer orangeMask.Close()
	gocv.InRangeWithScalar(hsv,
		gocv.NewScalar(float64(greenLower.R), float64(greenLower.G), float64(greenLower.B), 0),
		gocv.NewScalar(float64(greenUpper.R), float64(greenUpper.G), float64(greenUpper.B), 0),
		&greenMask)
	gocv.InRangeWithScalar(hsv,
		gocv.NewScalar(float64(orangeLower.R), float64(orangeLower.G), float64(orangeLower.B), 0),
		gocv.NewScalar(float64(orangeUpper.R), float64(orangeUpper.G), float64(orangeUpper.B), 0),
		&orangeMask)
	if gocv.CountNonZero(greenMask) > 0 {
		return "+"
	}
	if gocv.CountNonZero(orangeMask) > 0 {
		return "-"
	}
	return "."
}

func displayImage(img gocv.Mat) {
	window := gocv.NewWindow("Display Image")
	defer window.Close()
	window.IMShow(img)
	gocv.WaitKey(0)
}

// Function to detect lines using contours
func detectLines(binary gocv.Mat) gocv.PointsVector {
	contours := gocv.FindContours(binary, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	return contours
}

// Function to extract line positions from contours
func extractLinePositions(lines gocv.PointsVector, isHorizontal bool) []int {
	var positions []int
	for i := 0; i < lines.Size(); i++ {
		rect := gocv.BoundingRect(lines.At(i))
		if isHorizontal {
			positions = append(positions, rect.Min.Y)
		} else {
			positions = append(positions, rect.Min.X)
		}
	}
	sort.Ints(positions)
	return positions
}
