// Wallpaper tint extraction.
//
// We pre-compute a representative tint for wallpapers on upload so the
// frontend doesn't have to fetch and downsample the (potentially huge) image
// on first paint. The result is shipped back in the upload response and
// stored on board.theme_custom.glass_tint by the SPA when the user saves the
// theme.
package api

import (
	"bytes"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
)

// WallpaperTint is the JSON shape mirrored on the frontend's
// composables/useTheme.ts ExtractedTint.
type WallpaperTint struct {
	GlassBg   string `json:"glass_bg"`
	Border    string `json:"border"`
	Glow      string `json:"glow"`
	Highlight string `json:"highlight"`
	// AverageHex is purely informational (debug / future themes).
	AverageHex string `json:"average_hex"`
}

// extractWallpaperTint downsamples the image to a small grid and averages the
// pixel colors. Lossy by design — we only need a representative chroma to seed
// the Aero glass tint.
func extractWallpaperTint(data []byte) (WallpaperTint, bool) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return WallpaperTint{}, false
	}
	bounds := img.Bounds()
	if bounds.Dx() == 0 || bounds.Dy() == 0 {
		return WallpaperTint{}, false
	}

	const samples = 64
	stepX := math.Max(1, float64(bounds.Dx())/float64(samples))
	stepY := math.Max(1, float64(bounds.Dy())/float64(samples))

	var sumR, sumG, sumB uint64
	var count uint64
	for fy := 0.0; fy < float64(bounds.Dy()); fy += stepY {
		for fx := 0.0; fx < float64(bounds.Dx()); fx += stepX {
			c := img.At(bounds.Min.X+int(fx), bounds.Min.Y+int(fy))
			r, g, b, _ := c.RGBA()
			sumR += uint64(r >> 8)
			sumG += uint64(g >> 8)
			sumB += uint64(b >> 8)
			count++
		}
	}
	if count == 0 {
		return WallpaperTint{}, false
	}

	avg := color.RGBA{
		R: uint8(sumR / count),
		G: uint8(sumG / count),
		B: uint8(sumB / count),
		A: 255,
	}
	return tintFromRGB(avg), true
}

func tintFromRGB(c color.RGBA) WallpaperTint {
	// Slightly darker, semi-transparent for the glass body.
	bg := alphaHex(scale(c, 0.55), 0.62)
	border := alphaHex(scale(c, 1.15), 0.34)
	glow := alphaHex(scale(c, 1.25), 0.18)
	highlight := alphaHex(scale(c, 1.45), 0.22)
	return WallpaperTint{
		GlassBg:    bg,
		Border:     border,
		Glow:       glow,
		Highlight:  highlight,
		AverageHex: hexOf(c),
	}
}

func scale(c color.RGBA, k float64) color.RGBA {
	clamp := func(f float64) uint8 {
		if f < 0 {
			return 0
		}
		if f > 255 {
			return 255
		}
		return uint8(f)
	}
	return color.RGBA{
		R: clamp(float64(c.R) * k),
		G: clamp(float64(c.G) * k),
		B: clamp(float64(c.B) * k),
		A: c.A,
	}
}

func alphaHex(c color.RGBA, alpha float64) string {
	a := alpha
	if a < 0 {
		a = 0
	} else if a > 1 {
		a = 1
	}
	return rgbaCSS(c.R, c.G, c.B, a)
}

func hexOf(c color.RGBA) string {
	const hex = "0123456789abcdef"
	out := make([]byte, 7)
	out[0] = '#'
	out[1] = hex[c.R>>4]
	out[2] = hex[c.R&0xf]
	out[3] = hex[c.G>>4]
	out[4] = hex[c.G&0xf]
	out[5] = hex[c.B>>4]
	out[6] = hex[c.B&0xf]
	return string(out)
}

func rgbaCSS(r, g, b uint8, a float64) string {
	// Compact rgba() string with two-decimal alpha.
	return "rgba(" +
		itoa(int(r)) + "," + itoa(int(g)) + "," + itoa(int(b)) + "," +
		twoDecimal(a) + ")"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	buf := make([]byte, 0, 4)
	for n > 0 {
		buf = append([]byte{byte('0') + byte(n%10)}, buf...)
		n /= 10
	}
	if negative {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

func twoDecimal(f float64) string {
	if f < 0 {
		f = 0
	}
	if f > 1 {
		f = 1
	}
	scaled := int(math.Round(f * 100))
	if scaled >= 100 {
		return "1"
	}
	if scaled < 10 {
		return "0.0" + itoa(scaled)
	}
	return "0." + itoa(scaled)
}
