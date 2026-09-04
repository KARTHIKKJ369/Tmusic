package views

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"strings"

	"golang.org/x/image/draw"
)

// RenderCoverArt converts image bytes into ANSI truecolor half-block art (▀).
// targetWidth is character columns, targetHeight is character rows (each row = 2 pixel rows).
func RenderCoverArt(data []byte, targetWidth, targetHeight int, tick int64, isPlaying bool) []string {
	if len(data) > 0 {
		if lines, err := decodeAndRenderANSI(data, targetWidth, targetHeight); err == nil && len(lines) > 0 {
			return lines
		}
	}
	// Fallback to geometric vinyl disc / synthwave art
	return renderVinylArt(targetWidth, targetHeight, tick, isPlaying)
}

func decodeAndRenderANSI(data []byte, targetWidth, targetHeight int) ([]string, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	pixelHeight := targetHeight * 2
	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, pixelHeight))

	// Scale image cleanly with BiLinear interpolation
	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	lines := make([]string, targetHeight)
	for y := 0; y < targetHeight; y++ {
		var sb strings.Builder
		topY := y * 2
		botY := y*2 + 1

		for x := 0; x < targetWidth; x++ {
			topR, topG, topB, _ := dst.At(x, topY).RGBA()
			botR, botG, botB, _ := dst.At(x, botY).RGBA()

			// Convert 16-bit RGBA (0..65535) to 8-bit (0..255)
			tr, tg, tb := uint8(topR>>8), uint8(topG>>8), uint8(topB>>8)
			br, bg, bb := uint8(botR>>8), uint8(botG>>8), uint8(botB>>8)

			// ▀ has foreground = top pixel, background = bottom pixel
			fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀", tr, tg, tb, br, bg, bb)
		}
		sb.WriteString("\x1b[0m")
		lines[y] = sb.String()
	}

	return lines, nil
}

// renderVinylArt renders a vinyl record / disc with rotating grooves.
func renderVinylArt(targetWidth, targetHeight int, tick int64, isPlaying bool) []string {
	pixelHeight := targetHeight * 2
	lines := make([]string, targetHeight)

	centerX := float64(targetWidth) / 2.0
	centerY := float64(pixelHeight) / 2.0
	maxRadius := math.Min(centerX, centerY) - 1.0

	var angleOffset float64
	if isPlaying {
		angleOffset = float64(tick) * 0.25
	}

	for y := 0; y < targetHeight; y++ {
		var sb strings.Builder
		topY := float64(y * 2)
		botY := float64(y*2 + 1)

		for x := 0; x < targetWidth; x++ {
			fx := float64(x)
			tr, tg, tb := vinylPixel(fx, topY, centerX, centerY, maxRadius, angleOffset)
			br, bg, bb := vinylPixel(fx, botY, centerX, centerY, maxRadius, angleOffset)

			fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀", tr, tg, tb, br, bg, bb)
		}
		sb.WriteString("\x1b[0m")
		lines[y] = sb.String()
	}

	return lines
}

func vinylPixel(x, y, cx, cy, maxR, angleOffset float64) (uint8, uint8, uint8) {
	// Correct for character aspect ratio (terminal characters are roughly twice as tall as wide)
	dx := (x - cx) * 1.8
	dy := (y - cy)
	dist := math.Sqrt(dx*dx + dy*dy)

	if dist > maxR*1.8 {
		// Outer background (dark sleek obsidian)
		return 18, 18, 28
	}

	// Center spindle (gold / amber hub)
	if dist < maxR*0.25 {
		if dist < maxR*0.08 {
			return 240, 240, 255 // silver center hole
		}
		return 255, 180, 50 // golden label
	}

	// Inner label ring (crimson / magenta)
	if dist < maxR*0.55 {
		return 220, 30, 80
	}

	// Vinyl grooves with light reflection sheen
	angle := math.Atan2(dy, dx) + angleOffset
	sheen := (math.Sin(angle*2.0) + 1.0) / 2.0 // 2-quadrant glossy sheen

	// Groove ripples
	ripple := (math.Sin(dist*3.5) + 1.0) / 2.0

	baseVal := 22.0 + (sheen * 35.0) + (ripple * 12.0)
	if baseVal > 255 {
		baseVal = 255
	}

	r := uint8(baseVal * 0.9)
	g := uint8(baseVal * 0.95)
	b := uint8(baseVal * 1.1)

	return r, g, b
}
