package memento

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"sort"
)

// A simple paletted image is a paletted image where:
// - Index 0 is transparent.
// - Max 255 distinct opaque colors (color.RGBA64{A: 65535}).
// - No partially-opaque colors.
// - Palette sorted by A then R G B. (No particular reason.)
// - Not empty (has positive width and height).

// These functions may panic on extreme cases such as:
// - r.Empty()
// - !sp.In(src.Rect)
// - r != r.Intersect(dst.Rect).Intersect(
//      src.Rect.Add(dst.Rect.Min.Sub(src.Rect.Min)))

// All simple paletted images passed to a function must share the same palette.
// These functions run faster than Go image/draw by doing less:
// - No out-of-bounds checking.
// - No color remapping, it just copies indexes.
// - Index 0 is always transparent.
// - All other indexes are fully opaque.
// - No overlap checking.

// --- functions here ---

// Collect distinct opaque RGBA64 colors in the image.
// Error if image has partial-alpha.
func collectDistinctOpaqueColors(src image.RGBA64Image) (map[color.RGBA64]struct{}, error) {
	bounds := src.Bounds()
	colors := make(map[color.RGBA64]struct{})
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := src.RGBA64At(x, y)
			if c.A == 65535 {
				colors[c] = struct{}{}
			} else if c.A != 0 {
				return nil, fmt.Errorf("found partial-alpha color %v", c)
			}
		}
	}
	return colors, nil
}

// Load image, return it and its distinct opaque colors.
func loadImageAndDistinctOpaqueColors(imgBytes []byte) (image.RGBA64Image, map[color.RGBA64]struct{}, error) {
	src, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, nil, err
	}
	img64, ok := src.(image.RGBA64Image)
	if !ok {
		return nil, nil, fmt.Errorf("cannot cast %T to RGBA64Image", src)
	}
	if img64.Bounds().Empty() || img64.Bounds().Min != image.Pt(0, 0) {
		return nil, nil, fmt.Errorf("unexpected image bounds: %v", img64.Bounds())
	}
	colors, err := collectDistinctOpaqueColors(img64)
	if err != nil {
		return nil, nil, err
	}
	return img64, colors, nil
}

// Serialize a canonical palette with 0 as Transparent and 1..255 as colors.
func serializePalette(colors map[color.RGBA64]struct{}) ([]color.Color, error) {
	pal := make([]color.Color, 0, len(colors)+1)
	// Always put color.Transparent first.
	pal = append(pal, color.Transparent)
	for k := range colors {
		pal = append(pal, k)
	}
	if len(pal) > 256 {
		return nil, fmt.Errorf("gif cannot support %d colors", len(pal))
	}

	// Sort deterministically, exclude the color.Transparent.
	// No need to compara alpha, assume pal[1:] has the same alpha.
	sort.Slice(pal[1:], func(i, j int) bool {
		ri, gi, bi, _ := pal[i+1].RGBA()
		rj, gj, bj, _ := pal[j+1].RGBA()
		if ri != rj {
			return ri < rj
		}
		if gi != gj {
			return gi < gj
		}
		return bi < bj
	})

	return pal, nil
}

// Clone any image into an image.Paletted with a specific palette.
// (Slow if src is already an image.Paletted of the correct palette because
// remapping the color indexes would be unnecessary.)
func cloneToPaletted(src image.Image, pal []color.Color) *image.Paletted {
	dst := image.NewPaletted(src.Bounds(), pal)
	draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Src)
	return dst
}

// Fill non-empty image.Paletted with dst.Pix[0] (the top-left pixel color).
func fillPaletted(dst *image.Paletted) {
	// https://gist.github.com/taylorza/df2f89d5f9ab3ffd06865062a4cf015d#fill-the-slice-using-the-builtin-copy-function-to-incrementally-duplicate-the-data-through-the-array
	for j := 1; j < len(dst.Pix); j <<= 1 {
		copy(dst.Pix[j:], dst.Pix[:j])
	}
}

// Fast paint single color. Slower than fillPaletted when used on the full image.
func fillPalettedRect(dst *image.Paletted, r image.Rectangle, c byte) {
	dstP := dst.PixOffset(r.Min.X, r.Min.Y)
	dx, dy := r.Dx(), r.Dy()
	for y := 0; y < dy; y++ {
		for x := 0; x < dx; x++ {
			dst.Pix[dstP+x] = c
		}
		dstP += dst.Stride
	}
}

// Fast draw.Draw(..., draw.Src) for two distinct images sharing the same simple palette.
func fastDrawSrc(dst *image.Paletted, r image.Rectangle, src *image.Paletted, sp image.Point) {
	dstP := dst.PixOffset(r.Min.X, r.Min.Y)
	srcP := src.PixOffset(sp.X, sp.Y)
	dx, dy := r.Dx(), r.Dy()
	for y := 0; y < dy; y++ {
		copy(dst.Pix[dstP:dstP+dx], src.Pix[srcP:srcP+dx])
		dstP += dst.Stride
		srcP += src.Stride
	}
}

// Fast draw.Draw(..., draw.Over) for two distinct images sharing the same simple palette.
func fastDrawOver(dst *image.Paletted, r image.Rectangle, src *image.Paletted, sp image.Point) {
	dstP := dst.PixOffset(r.Min.X, r.Min.Y)
	srcP := src.PixOffset(sp.X, sp.Y)
	dx, dy := r.Dx(), r.Dy()
	for y := 0; y < dy; y++ {
		for x := 0; x < dx; x++ {
			c := src.Pix[srcP+x]
			if c != 0 {
				dst.Pix[dstP+x] = c
			}
		}
		dstP += dst.Stride
		srcP += src.Stride
	}
}

// Given two distinct images sharing the same simple palette, turn within-bound
// pixels in dst image to transparent if it matches the pixel in src image.
func fastUndrawOver(dst *image.Paletted, r image.Rectangle, src *image.Paletted, sp image.Point) {
	dstP := dst.PixOffset(r.Min.X, r.Min.Y)
	srcP := src.PixOffset(sp.X, sp.Y)
	dx, dy := r.Dx(), r.Dy()
	for y := 0; y < dy; y++ {
		for x := 0; x < dx; x++ {
			if dst.Pix[dstP+x] == src.Pix[srcP+x] {
				dst.Pix[dstP+x] = 0
			}
		}
		dstP += dst.Stride
		srcP += src.Stride
	}
}

// Given a simple paletted image, return the bounding box of the opaque pixels.
func croppedBounds(src *image.Paletted) image.Rectangle {
	dx, dy := src.Rect.Dx(), src.Rect.Dy()

	// Top
	firstY := -1
	srcP := 0
findFirstY:
	for y := 0; y < dy; y++ {
		for x := 0; x < dx; x++ {
			if src.Pix[srcP+x] != 0 {
				firstY = y
				break findFirstY
			}
		}
		srcP += src.Stride
	}
	if firstY < 0 {
		// Image is entirely transparent.
		return image.Rectangle{}
	}

	// Bottom
	lastY := -1
	srcP = dy * src.Stride
findLastY:
	for y := dy - 1; y >= 0; y-- {
		srcP -= src.Stride
		for x := 0; x < dx; x++ {
			if src.Pix[srcP+x] != 0 {
				lastY = y
				break findLastY
			}
		}
	}

	// Left
	srcPY := firstY * src.Stride
	firstX := -1
findFirstX:
	for x := 0; x < dx; x++ {
		srcP = srcPY + x
		for y := firstY; y <= lastY; y++ {
			if src.Pix[srcP] != 0 {
				firstX = x
				break findFirstX
			}
			srcP += src.Stride
		}
	}

	// Right
	lastX := -1
findLastX:
	for x := dx - 1; x >= 0; x-- {
		srcP = srcPY + x
		for y := firstY; y <= lastY; y++ {
			if src.Pix[srcP] != 0 {
				lastX = x
				break findLastX
			}
			srcP += src.Stride
		}
	}

	return image.Rect(src.Rect.Min.X+firstX, src.Rect.Min.Y+firstY, src.Rect.Min.X+lastX+1, src.Rect.Min.Y+lastY+1)
}

// Given a simple paletted image, return the bounding box of the opaque pixels different from dst.
func croppedBoundsDiff(src *image.Paletted, r image.Rectangle, dst *image.Paletted, dp image.Point) image.Rectangle {
	srcPY := src.PixOffset(r.Min.X, r.Min.Y)
	dstPY := dst.PixOffset(dp.X, dp.Y)
	dx, dy := r.Dx(), r.Dy()

	// Top
	firstY := -1
	srcP := srcPY
	dstP := dstPY
findFirstY:
	for y := 0; y < dy; y++ {
		for x := 0; x < dx; x++ {
			if src.Pix[srcP+x] != 0 && src.Pix[srcP+x] != dst.Pix[dstP+x] {
				firstY = y
				break findFirstY
			}
		}
		srcP += src.Stride
		dstP += dst.Stride
	}
	if firstY < 0 {
		// Image is entirely transparent.
		return image.Rectangle{}
	}

	// Bottom
	lastY := -1
	srcP = srcPY + dy*src.Stride
	dstP = dstPY + dy*dst.Stride
findLastY:
	for y := dy - 1; y >= 0; y-- {
		srcP -= src.Stride
		dstP -= dst.Stride
		for x := 0; x < dx; x++ {
			if src.Pix[srcP+x] != 0 && src.Pix[srcP+x] != dst.Pix[dstP+x] {
				lastY = y
				break findLastY
			}
		}
	}

	// Left
	srcPY += firstY * src.Stride
	dstPY += firstY * dst.Stride
	firstX := -1
findFirstX:
	for x := 0; x < dx; x++ {
		srcP = srcPY + x
		dstP = dstPY + x
		for y := firstY; y <= lastY; y++ {
			if src.Pix[srcP] != 0 && src.Pix[srcP] != dst.Pix[dstP] {
				firstX = x
				break findFirstX
			}
			srcP += src.Stride
			dstP += dst.Stride
		}
	}

	// Right
	lastX := -1
findLastX:
	for x := dx - 1; x >= 0; x-- {
		srcP = srcPY + x
		dstP = dstPY + x
		for y := firstY; y <= lastY; y++ {
			if src.Pix[srcP] != 0 && src.Pix[srcP] != dst.Pix[dstP] {
				lastX = x
				break findLastX
			}
			srcP += src.Stride
			dstP += dst.Stride
		}
	}

	return image.Rect(r.Min.X+firstX, r.Min.Y+firstY, r.Min.X+lastX+1, r.Min.Y+lastY+1)
}

// fastDrawSrc() with a different argument order. src is a SubImage.
func fastSpriteDrawSrc(dst *image.Paletted, dp image.Point, src *image.Paletted) {
	dstP := dst.PixOffset(dp.X, dp.Y)
	srcP := src.PixOffset(src.Rect.Min.X, src.Rect.Min.Y)
	dx, dy := src.Rect.Dx(), src.Rect.Dy()
	for y := 0; y < dy; y++ {
		copy(dst.Pix[dstP:dstP+dx], src.Pix[srcP:srcP+dx])
		dstP += dst.Stride
		srcP += src.Stride
	}
}

// fastDrawOver() with a different argument order. src is a SubImage.
func fastSpriteDrawOver(dst *image.Paletted, dp image.Point, src *image.Paletted) {
	dstP := dst.PixOffset(dp.X, dp.Y)
	srcP := src.PixOffset(src.Rect.Min.X, src.Rect.Min.Y)
	dx, dy := src.Rect.Dx(), src.Rect.Dy()
	for y := 0; y < dy; y++ {
		for x := 0; x < dx; x++ {
			c := src.Pix[srcP+x]
			if c != 0 {
				dst.Pix[dstP+x] = c
			}
		}
		dstP += dst.Stride
		srcP += src.Stride
	}
}
