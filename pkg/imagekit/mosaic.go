package imagekit

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strconv"
	"strings"

	"github.com/bits-and-blooms/bitset"
)

// --- Default Configuration Constants ---
const (
	// Visual properties for the generated mosaic.
	// Can be defined in pixels ("2px") or as a percentage of cell width ("10%").
	defaultGridSpacing = "10%" // The space between each cell.
	defaultBorderSize  = "5%"  // The border thickness, drawn inside the cell.
	defaultCellRadius  = "0px" // The corner radius for each cell.

	// Dimming effect for "unrevealed" cells
	dimmedOpacity = 0.1 // Value between 0.0 (fully transparent) and 1.0 (fully opaque)

	// Colors in Hex format (RRGGBBAA). Use "00" for alpha for full transparency.
	defaultBackgroundColorHex      = "#333333FF" // Dark grey
	defaultBorderColorHex          = "#000000FF" // Black
	recentlyRevealedBorderColorHex = "#4CAF50FF" // Bright Green
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

// parseDimension parses a string that can be a pixel value (e.g., "2px") or a
// percentage (e.g., "10%") relative to a base value.
func parseDimension(dimStr string, base int) (int, error) {
	if strings.HasSuffix(dimStr, "%") {
		percentStr := strings.TrimSuffix(dimStr, "%")
		percent, err := strconv.ParseFloat(percentStr, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid percentage format: %s", dimStr)
		}
		// Calculate value and convert to integer pixels
		return int((percent / 100.0) * float64(base)), nil
	}

	if strings.HasSuffix(dimStr, "px") {
		pixelStr := strings.TrimSuffix(dimStr, "px")
		pixels, err := strconv.Atoi(pixelStr)
		if err != nil {
			return 0, fmt.Errorf("invalid pixel format: %s", dimStr)
		}
		return pixels, nil
	}

	return 0, fmt.Errorf("invalid dimension format: must end with 'px' or '%%'")
}

// createRoundedRectMask creates an alpha mask for a rounded rectangle.
func createRoundedRectMask(r image.Rectangle, radius int) *image.Alpha {
	mask := image.NewAlpha(r)
	if radius <= 0 { // If no radius, just fill the rectangle (much faster)
		draw.Draw(mask, r, image.Opaque, image.Point{}, draw.Src)
		return mask
	}

	// Clamp radius to half of the smallest dimension
	maxRadius := r.Dx() / 2
	if r.Dy()/2 < maxRadius {
		maxRadius = r.Dy() / 2
	}
	if radius > maxRadius {
		radius = maxRadius
	}

	// Corner centers
	c1 := image.Point{r.Min.X + radius, r.Min.Y + radius}         // Top-left
	c2 := image.Point{r.Max.X - radius - 1, r.Min.Y + radius}     // Top-right
	c3 := image.Point{r.Min.X + radius, r.Max.Y - radius - 1}     // Bottom-left
	c4 := image.Point{r.Max.X - radius - 1, r.Max.Y - radius - 1} // Bottom-right
	radiusSq := float64(radius * radius)

	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			// In a straight section
			if (x >= c1.X && x <= c2.X) || (y >= c1.Y && y <= c3.Y) {
				mask.SetAlpha(x, y, color.Alpha{A: 255})
				continue
			}

			var distSq float64
			// Top-left corner
			if x < c1.X && y < c1.Y {
				distSq = math.Pow(float64(c1.X-x), 2) + math.Pow(float64(c1.Y-y), 2)
			} else if x > c2.X && y < c2.Y { // Top-right corner
				distSq = math.Pow(float64(x-c2.X), 2) + math.Pow(float64(c2.Y-y), 2)
			} else if x < c3.X && y > c3.Y { // Bottom-left corner
				distSq = math.Pow(float64(c3.X-x), 2) + math.Pow(float64(y-c3.Y), 2)
			} else if x > c4.X && y > c4.Y { // Bottom-right corner
				distSq = math.Pow(float64(x-c4.X), 2) + math.Pow(float64(y-c4.Y), 2)
			}

			if distSq <= radiusSq {
				mask.SetAlpha(x, y, color.Alpha{A: 255})
			}
		}
	}
	return mask
}

// GenerateMosaic creates a mosaic image. It derives the grid layout from the source image's
// dimensions and uses the provided cellWidth and cellHeight for the output cell size.
func GenerateMosaic(src image.Image, revealedPixels, recentlyRevealedPixels *bitset.BitSet, cellWidth, cellHeight int) image.Image {
	// --- 1. Derive Grid Dimensions and Parse Colors/Dimensions ---
	bounds := src.Bounds()
	gridWidth := uint(bounds.Dx())
	gridHeight := uint(bounds.Dy())

	if gridWidth == 0 || gridHeight == 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}

	// Parse string-based dimensions into concrete integer values.
	// A malformed constant is a fatal error.
	gridSpacing, err := parseDimension(defaultGridSpacing, cellWidth)
	if err != nil {
		panic(fmt.Sprintf("could not parse defaultGridSpacing: %v", err))
	}
	borderSize, err := parseDimension(defaultBorderSize, cellWidth)
	if err != nil {
		panic(fmt.Sprintf("could not parse defaultBorderSize: %v", err))
	}
	cellRadius, err := parseDimension(defaultCellRadius, cellWidth)
	if err != nil {
		panic(fmt.Sprintf("could not parse defaultCellRadius: %v", err))
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
	outputWidth := (int(gridWidth) * cellWidth) + ((int(gridWidth) + 1) * gridSpacing)
	outputHeight := (int(gridHeight) * cellHeight) + ((int(gridHeight) + 1) * gridSpacing)

	finalImage := image.NewRGBA(image.Rect(0, 0, outputWidth, outputHeight))
	draw.Draw(finalImage, finalImage.Bounds(), image.NewUniform(bgColor), image.Point{}, draw.Src)

	// --- 2. Generate the full grid (in a temporary, dimmed state) ---
	dimmedGrid := image.NewRGBA(finalImage.Bounds())
	for r := uint(0); r < gridHeight; r++ {
		for c := uint(0); c < gridWidth; c++ {
			cellColor := cellColors[r][c]
			cellX0 := gridSpacing + int(c)*(cellWidth+gridSpacing)
			cellY0 := gridSpacing + int(r)*(cellHeight+gridSpacing)
			drawCell(dimmedGrid, cellX0, cellY0, cellWidth, cellHeight, borderSize, cellRadius, cellColor, defaultBorderColor, dimmedOpacity)
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
			cellX0 := gridSpacing + int(col)*(cellWidth+gridSpacing)
			cellY0 := gridSpacing + int(row)*(cellHeight+gridSpacing)

			borderColor := defaultBorderColor
			if recentlyRevealedPixels != nil && recentlyRevealedPixels.Test(i) {
				borderColor = recentBorderColor
			}

			drawCell(finalImage, cellX0, cellY0, cellWidth, cellHeight, borderSize, cellRadius, cellColor, borderColor, 1.0)
		}
	}

	return finalImage
}

// drawCell is a helper to draw a single cell with a border onto a destination image.
func drawCell(dst draw.Image, x, y, cellWidth, cellHeight, borderSize, cellRadius int, cellColor, borderColor color.Color, opacity float64) {
	opaqBorder := applyOpacity(borderColor, opacity)
	opaqCell := applyOpacity(cellColor, opacity)

	// Draw the outer border shape
	outerRect := image.Rect(x, y, x+cellWidth, y+cellHeight)
	if outerRect.Empty() {
		return
	}
	outerMask := createRoundedRectMask(outerRect, cellRadius)
	draw.DrawMask(dst, outerRect, image.NewUniform(opaqBorder), image.Point{}, outerMask, outerRect.Min, draw.Over)

	// Calculate the inner rectangle for the cell fill
	innerRect := image.Rect(
		x+borderSize,
		y+borderSize,
		x+cellWidth-borderSize,
		y+cellHeight-borderSize,
	)

	// Draw the inner cell shape if it's a valid rectangle
	if !innerRect.Empty() && innerRect.Min.X < innerRect.Max.X && innerRect.Min.Y < innerRect.Max.Y {
		innerRadius := cellRadius - borderSize
		if innerRadius < 0 {
			innerRadius = 0
		}
		innerMask := createRoundedRectMask(innerRect, innerRadius)
		draw.DrawMask(dst, innerRect, image.NewUniform(opaqCell), image.Point{}, innerMask, innerRect.Min, draw.Over)
	}
}

// applyOpacity creates a new color with a modified alpha channel.
func applyOpacity(c color.Color, opacity float64) color.Color {
	r, g, b, a := c.RGBA()
	newAlpha := uint8(float64(a>>8) * opacity)
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: newAlpha}
}
