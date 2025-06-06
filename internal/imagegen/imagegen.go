package imagegen

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg" // Register JPEG decoder
	"image/png"    // Register PNG decoder
	"io/ioutil"
	"log"
	"os"

	"github.com/bits-and-blooms/bitset"
)

// GenerateAchievementImage creates a partially revealed image based on the revealedPixels bitset.
// baseImagePath: Path to the original, fully revealed image.
// revealedPixels: Bitset indicating which pixels/blocks are revealed.
// totalPixelBlocks: Total number of blocks the image is conceptually divided into (e.g., 504).
// gridWidthInBlocks: How many blocks wide the conceptual grid is.
// outputDir: Directory to save the generated temporary image.
// Returns the path to the generated image, or an error.
func GenerateAchievementImage(
	baseImagePath string,
	revealedPixels *bitset.BitSet,
	totalPixelBlocks uint,
	gridWidthInBlocks uint,
	outputDir string,
) (generatedImagePath string, err error) {

	if gridWidthInBlocks == 0 {
		return "", fmt.Errorf("gridWidthInBlocks cannot be zero")
	}
	if totalPixelBlocks == 0 {
		return "", fmt.Errorf("totalPixelBlocks cannot be zero")
	}

	// Open the base image file
	baseFile, err := os.Open(baseImagePath)
	if err != nil {
		return "", fmt.Errorf("failed to open base image '%s': %w", baseImagePath, err)
	}
	defer baseFile.Close()

	// Decode the base image
	baseImg, _, err := image.Decode(baseFile)
	if err != nil {
		return "", fmt.Errorf("failed to decode base image '%s': %w", baseImagePath, err)
	}

	bounds := baseImg.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	// Calculate grid height and individual block dimensions
	gridHeightInBlocks := (totalPixelBlocks + gridWidthInBlocks - 1) / gridWidthInBlocks // Ceiling division
	if gridHeightInBlocks == 0 {                                                         // Should not happen if totalPixelBlocks > 0
		gridHeightInBlocks = 1
	}

	blockWidthPx := imgWidth / int(gridWidthInBlocks)
	blockHeightPx := imgHeight / int(gridHeightInBlocks)

	if blockWidthPx == 0 || blockHeightPx == 0 {
		return "", fmt.Errorf("calculated block dimensions are zero. Image size: %dx%d, Grid: %dx%d", imgWidth, imgHeight, gridWidthInBlocks, gridHeightInBlocks)
	}

	// Create a new RGBA image for the output
	// Option 1: Start with a copy of the base image and obscure parts
	// Option 2: Start with an obscured image and reveal parts (Chosen here for clarity)
	obscuredColor := color.RGBA{R: 50, G: 50, B: 50, A: 255} // Dark gray
	outputImg := image.NewRGBA(bounds)
	draw.Draw(outputImg, outputImg.Bounds(), &image.Uniform{C: obscuredColor}, image.Point{}, draw.Src)

	// Iterate through each conceptual block
	for i := uint(0); i < totalPixelBlocks; i++ {
		if revealedPixels.Test(i) {
			// Calculate the row and column of this block in the grid
			blockRow := i / gridWidthInBlocks
			blockCol := i % gridWidthInBlocks

			// Calculate the pixel coordinates for the top-left of this block
			startX := int(blockCol) * blockWidthPx
			startY := int(blockRow) * blockHeightPx

			// Define the rectangle for this block
			// Ensure block does not go out of image bounds, clip if necessary
			endX := startX + blockWidthPx
			if endX > imgWidth {
				endX = imgWidth
			}
			endY := startY + blockHeightPx
			if endY > imgHeight {
				endY = imgHeight
			}

			blockRect := image.Rect(startX, startY, endX, endY)

			// Copy the corresponding block from the base image to the output image
			// The source point for draw.Draw is relative to the source image's bounds.Min
			// The destination point for draw.Draw is relative to the destination image's bounds.Min
			// For copying a region, we want to draw from baseImg (at blockRect.Min) to outputImg (at blockRect.Min)
			// for the size of blockRect.
			draw.Draw(outputImg, blockRect, baseImg, image.Pt(startX, startY), draw.Src)
		}
	}

	// Create a temporary file for the output image
	// Ensure outputDir exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory '%s': %w", outputDir, err)
	}

	tempFile, err := ioutil.TempFile(outputDir, "achievement_*.png")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary image file: %w", err)
	}
	defer tempFile.Close()

	// Encode the output image to PNG and save it
	if err := png.Encode(tempFile, outputImg); err != nil {
		os.Remove(tempFile.Name()) // Attempt to clean up
		return "", fmt.Errorf("failed to encode output image to PNG: %w", err)
	}

	log.Printf("Generated achievement image at: %s", tempFile.Name())
	return tempFile.Name(), nil
}
