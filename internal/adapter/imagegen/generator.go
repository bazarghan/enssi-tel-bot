package imagegen

import (
	"fmt"
	"github.com/2000ostd/enssi-tel-bot/pkg/imagekit"
	"github.com/bits-and-blooms/bitset"
	"image"
	_ "image/jpeg" // Register JPEG decoder
	"image/png"    // Register PNG decoder
	"os"
)

// Generator implements the domain's ImageGenerator service port.
type Generator struct {
	OutputDir string
}

// NewGenerator creates a new image generator, ensuring the output directory exists.
func NewGenerator(outputDir string) (*Generator, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory '%s': %w", outputDir, err)
	}
	return &Generator{OutputDir: outputDir}, nil
}

// Generate creates a progressive image and saves it to a temporary file.
func (g *Generator) Generate(baseImagePath string, revealed *bitset.BitSet, recentlyRevealed *bitset.BitSet, gridWidth, gridHeight uint) (string, error) {
	baseFile, err := os.Open(baseImagePath)
	if err != nil {
		return "", fmt.Errorf("failed to open base image '%s': %w", baseImagePath, err)
	}
	defer baseFile.Close()

	srcImage, _, err := image.Decode(baseFile)
	if err != nil {
		return "", fmt.Errorf("failed to decode base image '%s': %w", baseImagePath, err)
	}

	// Call the pure image processing function from the pkg.
	generatedImage := imagekit.GenerateMosaic(srcImage, revealed, recentlyRevealed, int(gridWidth), int(gridHeight))

	// Create a temporary file for the output.
	tempFile, err := os.CreateTemp(g.OutputDir, "achievement_*.png")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary image file: %w", err)
	}
	defer tempFile.Close()

	// Encode the final composite image to PNG and save it.
	if err := png.Encode(tempFile, generatedImage); err != nil {
		os.Remove(tempFile.Name()) // Attempt to clean up on failure
		return "", fmt.Errorf("failed to encode output image to PNG: %w", err)
	}

	return tempFile.Name(), nil
}

