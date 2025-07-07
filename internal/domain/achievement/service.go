package achievement

import "github.com/bits-and-blooms/bitset"

// ImageGenerator is a port for a service that can generate progressive achievement images.
type ImageGenerator interface {
	// Generate takes the path to a base image and returns the path to a newly generated temp image.
	Generate(baseImagePath string, revealed *bitset.BitSet, recentlyRevealed *bitset.BitSet, gridWidth, gridHeight uint) (generatedImagePath string, err error)
}

