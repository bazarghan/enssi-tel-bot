package utils

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"math/rand"
	"os"

	"github.com/bits-and-blooms/bitset"
)

func UpdateImage(imagePath string, state *bitset.BitSet, dim uint, cnt uint) {
	outputFilename := "output.png"

	imageFile, err := os.Open(imagePath)
	if err != nil {
		fmt.Printf("Error opening input file: %v\n", err)
		return
	}
	defer imageFile.Close()

	imageSrc, _, err := image.Decode(imageFile)
	if err != nil {
		fmt.Printf("Error decoding image: %v\n", err)
		return
	}

	bounds := imageSrc.Bounds()

	imageSize := bounds.Dx()

	// Convert to RGBA for easy pixel manipulation.
	rgbaImage := image.NewRGBA(bounds)
	draw.Draw(rgbaImage, bounds, imageSrc, bounds.Min, draw.Src)

	squareSize := math.Round(float64(imageSize) / (1.1*float64(dim) - 0.1))
	intervalSize := squareSize / 10

	i := 0
	for i < int(cnt) {
		randNum := rand.Intn(int(dim * dim))

		if !state.Test(uint(randNum)) {
			state.Set(uint(randNum))
			i++
		}
	}

	for i, ok := state.NextSet(0); ok; i, ok = state.NextSet(i + 1) {
		y := (i / dim) * (uint(squareSize) + uint(intervalSize))
		x := (i % dim) * (uint(squareSize) + uint(intervalSize))

		// Verify that the square lies completely within the image bounds
		if x+uint(squareSize) > uint(bounds.Max.X) || y+uint(squareSize) > uint(bounds.Max.Y) {
			fmt.Println("Error: The specified square exceeds the image bounds.")
			return
		}

		for j := y; j <= y+uint(squareSize); j++ {
			for k := x; k <= x+uint(squareSize); k++ {
				p := rgbaImage.RGBAAt(int(k), int(j))

				if p.A != 0 {
					r := uint8(math.Min(255, float64(p.R)*255/float64(p.A)))
					g := uint8(math.Min(255, float64(p.G)*255/float64(p.A)))
					b := uint8(math.Min(255, float64(p.B)*255/float64(p.A)))
					rgbaImage.SetRGBA(int(k), int(j), color.RGBA{R: r, G: g, B: b, A: 255})
				} else {
					// Handle fully transparent pixels (set to opaque black, for example)
					rgbaImage.SetRGBA(int(k), int(j), color.RGBA{R: 0, G: 0, B: 0, A: 255})
				}
			}
		}
	}

	// Create a new background image filled with the color #333333.
	background := image.NewRGBA(bounds)
	bgColor := color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 255}
	draw.Draw(background, bounds, &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// Composite the entire (modified) image over the background.
	// This respects any transparency (outside our square, the reduced opacity remains).
	draw.Draw(background, bounds, rgbaImage, bounds.Min, draw.Over)

	// Create and save the output file as PNG
	outputFile, err := os.Create(outputFilename)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return
	}
	defer outputFile.Close()

	if err = png.Encode(outputFile, background); err != nil {
		fmt.Printf("Error encoding output image: %v\n", err)
		return
	}

	fmt.Printf("Output image saved as: %s\n", outputFilename)
}
