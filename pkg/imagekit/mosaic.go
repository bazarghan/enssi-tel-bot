package imagekit

import (
	"github.com/bits-and-blooms/bitset"
	"image"
	"image/color"
	"image/draw"
	"math"
)

// GenerateMosaic applies a mosaic effect to a source image, revealing parts based on a bitset.
// This function is pure and performs no file I/O.
func GenerateMosaic(src image.Image, revealedPixels *bitset.BitSet, gridWidth, gridHeight uint) image.Image {
	if gridWidth == 0 || gridHeight == 0 {
		return src // Return original image if grid is invalid
	}

	bounds := src.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	// Create a new RGBA image from the source to allow modifications.
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, src, bounds.Min, draw.Src)

	// Calculate square and gap sizes.
	sqW := math.Round(float64(imgWidth) / (1.1*float64(gridWidth) - 0.1))
	gapW := sqW / 10
	sqH := math.Round(float64(imgHeight) / (1.1*float64(gridHeight) - 0.1))
	gapH := sqH / 10

	// Iterate through each revealed bit and make the corresponding square fully opaque.
	for i, ok := revealedPixels.NextSet(0); ok; i, ok = revealedPixels.NextSet(i + 1) {
		row := i / gridWidth
		col := i % gridWidth

		y := row * (uint(sqH) + uint(gapH))
		x := col * (uint(sqW) + uint(gapW))

		// Iterate over the pixels of this specific square.
		for yy := y; yy < y+uint(sqH) && int(yy) < imgHeight; yy++ {
			for xx := x; xx < x+uint(sqW) && int(xx) < imgWidth; xx++ {
				p := rgba.RGBA64At(int(xx), int(yy))
				if p.A != 0 {
					// "Un-premultiply" alpha to make semi-transparent pixels fully opaque.
					r := uint8(math.Min(255, float64(p.R)*255/float64(p.A)))
					g := uint8(math.Min(255, float64(p.G)*255/float64(p.A)))
					b := uint8(math.Min(255, float64(p.B)*255/float64(p.A)))
					rgba.SetRGBA(int(xx), int(yy), color.RGBA{R: r, G: g, B: b, A: 255})
				}
			}
		}
	}

	// Create a dark-grey background.
	finalImage := image.NewRGBA(bounds)
	draw.Draw(finalImage, bounds, &image.Uniform{C: color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 255}}, image.Point{}, draw.Src)

	// Composite the modified rgba image (with revealed squares) over the background.
	draw.Draw(finalImage, bounds, rgba, bounds.Min, draw.Over)

	return finalImage
}
