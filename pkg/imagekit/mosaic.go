package imagekit

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strconv"
	"strings"

	"github.com/bits-and-blooms/bitset"
)

// --- Default Configuration Constants ---
const (
	// Visual properties for the generated mosaic
	defaultGridSpacing = 2 // The space between each cell.
	defaultBorderSize  = 2 // The border thickness, drawn inside the cell.

	// Dimming effect for "unrevealed" cells
	dimmedOpacity = 0.2 // Value between 0.0 (fully transparent) and 1.0 (fully opaque)

	// Colors in Hex format (RRGGBBAA). Use "00" for alpha for full transparency.
	defaultBackgroundColorHex      = "#333333FF" // Dark grey
	defaultBorderColorHex          = "#000000FF" // Black
	recentlyRevealedBorderColorHex = "#00FF00FF" // Bright Green
)

// parseHexColor converts a hex string like "#RRGGBBAA" to a color.RGBA object.
func parseHexColor(s string) (color.RGBA, error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 8 {
		return color.RGBA{}, fmt.Errorf("invalid hex color format, must be RRGGBBAA (e.g., #FF0000FF for red)")
	}

	r, err := strconv.ParseInt(s[0:2], 16, 0)
	if err != nil {
		return color.RGBA{}, err
	}
	g, err := strconv.ParseInt(s[2:4], 16, 0)
	if err != nil {
		return color.RGBA{}, err
	}
	b, err := strconv.ParseInt(s[4:6], 16, 0)
	if err != nil {
		return color.RGBA{}, err
	}
	a, err := strconv.ParseInt(s[6:8], 16, 0)
	if err != nil {
		return color.RGBA{}, err
	}

	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}, nil
}

// GenerateMosaic creates a mosaic image. It derives the grid layout from the source image's
// dimensions and uses the provided cellWidth and cellHeight for the output cell size.
func GenerateMosaic(src image.Image, revealedPixels, recentlyRevealedPixels *bitset.BitSet, cellWidth, cellHeight int) image.Image {
	// --- 1. Derive Grid Dimensions and Parse Colors ---
	bounds := src.Bounds()
	gridWidth := uint(bounds.Dx())
	gridHeight := uint(bounds.Dy())

	if gridWidth == 0 || gridHeight == 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}

	// Calculate the total number of cells in the grid.
	totalCells := gridWidth * gridHeight

	bgColor, _ := parseHexColor(defaultBackgroundColorHex)
	defaultBorderColor, _ := parseHexColor(defaultBorderColorHex)
	recentBorderColor, _ := parseHexColor(recentlyRevealedBorderColorHex)

	// Read all pixel colors from the source image.
	cellColors := make([][]color.Color, gridHeight)
	for r := uint(0); r < gridHeight; r++ {
		cellColors[r] = make([]color.Color, gridWidth)
		for c := uint(0); c < gridWidth; c++ {
			cellColors[r][c] = src.At(bounds.Min.X+int(c), bounds.Min.Y+int(r))
		}
	}

	// Calculate the total dimensions for the new image using the provided cell dimensions.
	outputWidth := (int(gridWidth) * cellWidth) + ((int(gridWidth) + 1) * defaultGridSpacing)
	outputHeight := (int(gridHeight) * cellHeight) + ((int(gridHeight) + 1) * defaultGridSpacing)

	finalImage := image.NewRGBA(image.Rect(0, 0, outputWidth, outputHeight))
	draw.Draw(finalImage, finalImage.Bounds(), image.NewUniform(bgColor), image.Point{}, draw.Src)

	// --- 2. Generate the full grid (in a temporary, dimmed state) ---
	dimmedGrid := image.NewRGBA(finalImage.Bounds())
	for r := uint(0); r < gridHeight; r++ {
		for c := uint(0); c < gridWidth; c++ {
			cellColor := cellColors[r][c]
			cellX0 := defaultGridSpacing + int(c)*(cellWidth+defaultGridSpacing)
			cellY0 := defaultGridSpacing + int(r)*(cellHeight+defaultGridSpacing)
			drawCell(dimmedGrid, cellX0, cellY0, cellWidth, cellHeight, cellColor, defaultBorderColor, dimmedOpacity)
		}
	}
	draw.Draw(finalImage, finalImage.Bounds(), dimmedGrid, image.Point{}, draw.Over)

	// --- 3. "Turn on" revealed cells with full opacity ---
	if revealedPixels != nil {
		for i, ok := revealedPixels.NextSet(0); ok; i, ok = revealedPixels.NextSet(i + 1) {
			// --- SAFETY CHECK ---
			// Ensure the index from the bitset is not out of bounds for our grid.
			if i >= totalCells {
				continue // Skip this invalid index and move to the next one.
			}

			row := i / gridWidth
			col := i % gridWidth

			cellColor := cellColors[row][col]
			cellX0 := defaultGridSpacing + int(col)*(cellWidth+defaultGridSpacing)
			cellY0 := defaultGridSpacing + int(row)*(cellHeight+defaultGridSpacing)

			borderColor := defaultBorderColor
			if recentlyRevealedPixels != nil && recentlyRevealedPixels.Test(i) {
				borderColor = recentBorderColor
			}

			drawCell(finalImage, cellX0, cellY0, cellWidth, cellHeight, cellColor, borderColor, 1.0)
		}
	}

	return finalImage
}

// drawCell is a helper to draw a single cell with a border onto a destination image.
func drawCell(dst draw.Image, x, y, cellWidth, cellHeight int, cellColor, borderColor color.Color, opacity float64) {
	opaqBorder := applyOpacity(borderColor, opacity)
	opaqCell := applyOpacity(cellColor, opacity)

	borderRect := image.Rect(x, y, x+cellWidth, y+cellHeight)
	draw.Draw(dst, borderRect, image.NewUniform(opaqBorder), image.Point{}, draw.Over)

	innerRect := image.Rect(
		x+defaultBorderSize,
		y+defaultBorderSize,
		x+cellWidth-defaultBorderSize,
		y+cellHeight-defaultBorderSize,
	)

	if innerRect.Min.X < innerRect.Max.X && innerRect.Min.Y < innerRect.Max.Y {
		draw.Draw(dst, innerRect, image.NewUniform(opaqCell), image.Point{}, draw.Over)
	}
}

// applyOpacity creates a new color with a modified alpha channel.
func applyOpacity(c color.Color, opacity float64) color.Color {
	r, g, b, a := c.RGBA()
	newAlpha := uint8(float64(a>>8) * opacity)
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: newAlpha}
}
