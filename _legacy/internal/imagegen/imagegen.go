package imagegen

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg" // Register JPEG decoder
	"image/png"    // Register PNG decoder
	"log"
	"math"
	"os"

	"github.com/bits-and-blooms/bitset"
)

// GenerateAchievementImage creates a partially revealed image based on the user's provided logic.
// It loads a base image, and for each revealed bit, makes the corresponding square fully opaque.
// The result is then drawn over a dark background.
func GenerateAchievementImage(
	baseImagePath string,
	revealedPixels *bitset.BitSet,
	totalPixelBlocks uint, // Not directly used in user's calculation logic but good for validation
	gridWidthInBlocks uint,
	gridHeightInBlocks uint,
	outputDir string,
) (generatedImagePath string, err error) {

	if gridWidthInBlocks == 0 || gridHeightInBlocks == 0 {
		return "", fmt.Errorf("grid dimensions cannot be zero")
	}

	// Open the base image file
	baseFile, err := os.Open(baseImagePath)
	if err != nil {
		return "", fmt.Errorf("failed to open base image '%s': %w", baseImagePath, err)
	}
	defer baseFile.Close()

	// Decode the base image
	src, _, err := image.Decode(baseFile)
	if err != nil {
		return "", fmt.Errorf("failed to decode base image '%s': %w", baseImagePath, err)
	}

	bounds := src.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	// Create an RGBA buffer of the source image to work with
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, src, bounds.Min, draw.Src)

	// --- Logic adapted from user's UpdateImage function ---

	// Calculate square and gap sizes based on the user's formula, adapted for non-square grids
	sqW := math.Round(float64(imgWidth) / (1.1*float64(gridWidthInBlocks) - 0.1))
	gapW := sqW / 10
	sqH := math.Round(float64(imgHeight) / (1.1*float64(gridHeightInBlocks) - 0.1))
	gapH := sqH / 10

	// Iterate through each bit that is set to '1' (revealed)
	for i, ok := revealedPixels.NextSet(0); ok; i, ok = revealedPixels.NextSet(i + 1) {
		// Calculate the row and column for this block
		row := i / gridWidthInBlocks
		col := i % gridWidthInBlocks

		// Calculate the top-left (x, y) coordinates of the square, including gaps
		y := row * (uint(sqH) + uint(gapH))
		x := col * (uint(sqW) + uint(gapW))

		// Iterate over the pixels of this specific square to make them fully opaque
		for yy := y; yy < y+uint(sqH) && int(yy) < imgHeight; yy++ {
			for xx := x; xx < x+uint(sqW) && int(xx) < imgWidth; xx++ {
				p := rgba.RGBAAt(int(xx), int(yy))
				if p.A != 0 {
					// This is the "un-premultiply" logic from your code
					// to make semi-transparent pixels fully opaque.
					r := uint8(math.Min(255, float64(p.R)*255/float64(p.A)))
					g := uint8(math.Min(255, float64(p.G)*255/float64(p.A)))
					b := uint8(math.Min(255, float64(p.B)*255/float64(p.A)))
					rgba.SetRGBA(int(xx), int(yy), color.RGBA{R: r, G: g, B: b, A: 255})
				} else {
					// Make fully transparent pixels opaque black
					rgba.SetRGBA(int(xx), int(yy), color.RGBA{0, 0, 0, 255})
				}
			}
		}
	}

	// Create a dark-grey background
	bg := image.NewRGBA(bounds)
	draw.Draw(bg, bounds, &image.Uniform{C: color.RGBA{0x33, 0x33, 0x33, 255}}, image.Point{}, draw.Src)

	// Composite the modified rgba image (with some squares now opaque) over the background
	draw.Draw(bg, bounds, rgba, bounds.Min, draw.Over)

	// --- End of adapted logic ---

	// Ensure outputDir exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory '%s': %w", outputDir, err)
	}

	// Create a temporary file for the output image
	tempFile, err := os.CreateTemp(outputDir, "achievement_*.png")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary image file: %w", err)
	}
	defer tempFile.Close()

	// Encode the final composite image (bg) to PNG and save it
	if err := png.Encode(tempFile, bg); err != nil {
		os.Remove(tempFile.Name()) // Attempt to clean up
		return "", fmt.Errorf("failed to encode output image to PNG: %w", err)
	}

	log.Printf("Generated achievement image at: %s", tempFile.Name())
	return tempFile.Name(), nil
}

