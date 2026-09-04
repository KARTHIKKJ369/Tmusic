package views

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"strings"

	"golang.org/x/image/draw"
)

// SupportsInlineImages detects if the current terminal supports native inline image protocols.
// Supported by iTerm2, Ghostty, WezTerm, VS Code terminal, Tabby, Warp, Kitty, etc.
func SupportsInlineImages() bool {
	termProg := os.Getenv("TERM_PROGRAM")
	if termProg == "iTerm.app" || termProg == "ghostty" || termProg == "WezTerm" ||
		termProg == "vscode" || termProg == "Tabby" || termProg == "WarpTerminal" {
		return true
	}
	if os.Getenv("LC_TERMINAL") == "iTerm2" {
		return true
	}
	if os.Getenv("KITTY_WINDOW_ID") != "" || os.Getenv("TERM") == "xterm-kitty" {
		return true
	}
	if os.Getenv("GHOSTTY_RESOURCES_DIR") != "" || os.Getenv("WEZTERM_EXECUTABLE") != "" {
		return true
	}
	return false
}

// RenderCoverArt renders cover art using native GPU inline images if supported,
// or falls back to enhanced TrueColor ANSI half-blocks (▀) or an animated vinyl disc.
func RenderCoverArt(data []byte, targetWidth, targetHeight int, tick int64, isPlaying bool) []string {
	if len(data) > 0 {
		if SupportsInlineImages() {
			return renderInlineImage(data, targetWidth, targetHeight)
		}
		if lines, err := decodeAndRenderANSI(data, targetWidth, targetHeight); err == nil && len(lines) > 0 {
			return lines
		}
	}
	// Fallback to geometric animated vinyl disc art
	return renderVinylArt(targetWidth, targetHeight, tick, isPlaying)
}

func renderInlineImage(data []byte, targetWidth, targetHeight int) []string {
	b64 := base64.StdEncoding.EncodeToString(data)
	imgSeq := fmt.Sprintf("\x1b]1337;File=inline=1;width=%d;height=%d;preserveAspectRatio=1:%s\a", targetWidth, targetHeight, b64)

	lines := make([]string, targetHeight)
	lines[0] = imgSeq + strings.Repeat(" ", targetWidth)
	for y := 1; y < targetHeight; y++ {
		lines[y] = strings.Repeat(" ", targetWidth)
	}
	return lines
}

func decodeAndRenderANSI(data []byte, targetWidth, targetHeight int) ([]string, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	pixelHeight := targetHeight * 2
	pixelWidth := targetWidth

	// Preserve 1:1 aspect ratio: calculate center-crop rectangle
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	var cropRect image.Rectangle
	if srcW > srcH {
		// Landscape: crop width
		x0 := srcBounds.Min.X + (srcW-srcH)/2
		cropRect = image.Rect(x0, srcBounds.Min.Y, x0+srcH, srcBounds.Max.Y)
	} else {
		// Portrait: crop height
		y0 := srcBounds.Min.Y + (srcH-srcW)/2
		cropRect = image.Rect(srcBounds.Min.X, y0, srcBounds.Max.X, y0+srcW)
	}

	// Create destination canvas
	dst := image.NewRGBA(image.Rect(0, 0, pixelWidth, pixelHeight))

	// Pre-fill canvas with obsidian background (#12121e)
	bgCol := color.RGBA{R: 18, G: 18, B: 30, A: 255}
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: bgCol}, image.Point{}, draw.Src)

	// High quality bicubic scaling with Catmull-Rom resampling
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, cropRect, draw.Over, nil)

	// Apply sharpening & contrast enhancement filter to bring out typography & artwork details
	sharpened := enhanceContrastAndSharpen(dst)

	lines := make([]string, targetHeight)
	for y := 0; y < targetHeight; y++ {
		var sb strings.Builder
		topY := y * 2
		botY := y*2 + 1

		for x := 0; x < targetWidth; x++ {
			topCol := sharpened.RGBAAt(x, topY)
			botCol := sharpened.RGBAAt(x, botY)

			// Alpha blend over background if transparent
			tr, tg, tb := blendAlpha(topCol, 18, 18, 30)
			br, bg, bb := blendAlpha(botCol, 18, 18, 30)

			// ANSI 24-bit TrueColor half block:
			// Foreground color = top pixel, Background color = bottom pixel
			fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀", tr, tg, tb, br, bg, bb)
		}
		sb.WriteString("\x1b[0m")
		lines[y] = sb.String()
	}

	return lines, nil
}

// enhanceContrastAndSharpen enhances edge contrast for downsampled album art
func enhanceContrastAndSharpen(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	out := image.NewRGBA(b)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x == 0 || y == 0 || x == w-1 || y == h-1 {
				out.SetRGBA(x, y, src.RGBAAt(x, y))
				continue
			}

			c := src.RGBAAt(x, y)
			up := src.RGBAAt(x, y-1)
			down := src.RGBAAt(x, y+1)
			left := src.RGBAAt(x-1, y)
			right := src.RGBAAt(x+1, y)

			// Unsharp mask 3x3 kernel (Laplacian edge detector)
			r := clampUint8(int(c.R)*5 - int(up.R) - int(down.R) - int(left.R) - int(right.R))
			g := clampUint8(int(c.G)*5 - int(up.G) - int(down.G) - int(left.G) - int(right.G))
			bl := clampUint8(int(c.B)*5 - int(up.B) - int(down.B) - int(left.B) - int(right.B))

			// Blend 50% sharpened with 50% original for clean natural sharpness
			finalR := uint8(float64(r)*0.5 + float64(c.R)*0.5)
			finalG := uint8(float64(g)*0.5 + float64(c.G)*0.5)
			finalB := uint8(float64(bl)*0.5 + float64(c.B)*0.5)

			out.SetRGBA(x, y, color.RGBA{R: finalR, G: finalG, B: finalB, A: c.A})
		}
	}
	return out
}

func clampUint8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func blendAlpha(c color.RGBA, bgR, bgG, bgB uint8) (uint8, uint8, uint8) {
	if c.A == 255 {
		return c.R, c.G, c.B
	}
	if c.A == 0 {
		return bgR, bgG, bgB
	}
	alpha := float64(c.A) / 255.0
	r := uint8(float64(c.R)*alpha + float64(bgR)*(1-alpha))
	g := uint8(float64(c.G)*alpha + float64(bgG)*(1-alpha))
	b := uint8(float64(c.B)*alpha + float64(bgB)*(1-alpha))
	return r, g, b
}

// renderVinylArt renders a vinyl record / disc with rotating grooves.
func renderVinylArt(targetWidth, targetHeight int, tick int64, isPlaying bool) []string {
	pixelHeight := targetHeight * 2
	lines := make([]string, targetHeight)

	centerX := float64(targetWidth) / 2.0
	centerY := float64(pixelHeight) / 2.0
	maxRadius := math.Min(centerX, centerY) - 0.5

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
	// 1.65 character cell aspect ratio correction for perfect round vinyl disc
	dx := (x - cx) * 1.65
	dy := (y - cy)
	dist := math.Sqrt(dx*dx + dy*dy)

	if dist > maxR*1.65 {
		// Outer background
		return 16, 16, 26
	}

	// Center spindle
	if dist < maxR*0.25 {
		if dist < maxR*0.08 {
			return 240, 240, 255 // silver center hole
		}
		return 255, 180, 50 // golden label
	}

	// Inner label ring
	if dist < maxR*0.55 {
		return 220, 30, 80
	}

	// Vinyl grooves with light reflection sheen
	angle := math.Atan2(dy, dx) + angleOffset
	sheen := (math.Sin(angle*2.0) + 1.0) / 2.0

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
