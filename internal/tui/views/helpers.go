package views

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/KARTHIKKJ369/Tmusic/internal/tui/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// FormatDur formats a duration as M:SS or H:MM:SS.
func FormatDur(d time.Duration) string {
	if d <= 0 {
		return "0:00"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// ProgressBar renders a text progress bar with elapsed and total duration.
func ProgressBar(filled float64, width int, elapsed, total time.Duration) string {
	if width < 12 {
		return ""
	}
	elapsedStr := FormatDur(elapsed)
	totalStr := FormatDur(total)
	
	// Format: " 1:23 ━━━━━●──────── 3:45 "
	barW := width - len(elapsedStr) - len(totalStr) - 4
	if barW < 4 {
		barW = 4
	}

	if filled < 0 {
		filled = 0
	}
	if filled > 1 {
		filled = 1
	}

	n := int(float64(barW) * filled)
	if n < 0 {
		n = 0
	}
	if n > barW {
		n = barW
	}

	filledStr := strings.Repeat("━", n)
	knobStr := "●"
	emptyStr := strings.Repeat("─", maxInt(0, barW-n-1))
	if n >= barW {
		knobStr = "━"
		emptyStr = ""
	}

	bar := styles.ProgressFilled.Render(filledStr) +
		styles.ProgressKnob.Render(knobStr) +
		styles.ProgressEmpty.Render(emptyStr)

	return fmt.Sprintf("%s %s %s",
		styles.Subtext.Render(elapsedStr),
		bar,
		styles.Subtext.Render(totalStr),
	)
}

// SpectrumEqualizer renders an animated 16-band audio equalizer.
func SpectrumEqualizer(width int, tick int64, isPlaying bool) string {
	if width < 6 {
		return ""
	}
	levels := []rune{' ', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	bands := minInt(width/2, 24)
	if bands < 4 {
		bands = 4
	}

	var sb strings.Builder
	for i := 0; i < bands; i++ {
		var lvlIdx int
		if isPlaying {
			// Sine-based rhythmic visual bounce with phase offset per band
			phase := float64(tick)*0.4 + float64(i)*0.65
			amp := (math.Sin(phase) + 1.0) / 2.0
			// Secondary harmonic for realism
			amp2 := (math.Sin(float64(tick)*0.7+float64(i)*1.1) + 1.0) / 2.0
			combined := (amp*0.7 + amp2*0.3)
			lvlIdx = int(combined * float64(len(levels)-1))
			if lvlIdx < 0 {
				lvlIdx = 0
			}
			if lvlIdx >= len(levels) {
				lvlIdx = len(levels) - 1
			}
		} else {
			lvlIdx = 1
		}
		sb.WriteRune(levels[lvlIdx])
		sb.WriteRune(' ')
	}
	return styles.Accent.Render(sb.String())
}

// MiniVU returns a 4-bar mini VU visualizer for the status bar.
func MiniVU(tick int64, isPlaying bool) string {
	if !isPlaying {
		return styles.Muted.Render("❚❚")
	}
	bars := []string{" ▃▆", "▃▆█", "▆█▃", "█▃ ", "▃ ▆"}
	return styles.Playing.Render(bars[int(tick)%len(bars)])
}

// FormatBadge returns a styled format pill badge (e.g., "FLAC", "MP3").
func FormatBadge(path string) string {
	ext := strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), "."))
	if ext == "" {
		ext = "AUDIO"
	}
	return styles.BadgeFormat.Render(" " + ext + " ")
}

// VolumeBar renders a graphical volume bar with percentage.
func VolumeBar(vol float64, width int) string {
	pct := int(vol * 100)
	n := int(vol * float64(width))
	if n < 0 {
		n = 0
	}
	if n > width {
		n = width
	}
	filled := strings.Repeat("█", n)
	empty := strings.Repeat("░", width-n)
	return fmt.Sprintf("%s %s%s %s",
		styles.Subtext.Render("Vol"),
		styles.Accent.Render(filled),
		styles.Muted.Render(empty),
		styles.Bold.Render(fmt.Sprintf("%d%%", pct)),
	)
}

// Center centers a string within the given terminal width, accounting for ANSI escapes.
func Center(s string, width int) string {
	vLen := VisibleLen(s)
	if vLen >= width {
		return s
	}
	pad := (width - vLen) / 2
	return strings.Repeat(" ", pad) + s
}

// VisibleLen counts visible characters by stripping all ANSI escape sequences via lipgloss.
func VisibleLen(s string) int {
	return lipgloss.Width(s)
}

func Pad(s string, width int) string {
	vLen := VisibleLen(s)
	if vLen >= width {
		return ansi.Truncate(s, width, "")
	}
	return s + strings.Repeat(" ", width-vLen)
}

func Trunc(s string, width int) string {
	if width <= 0 {
		return ""
	}
	vLen := VisibleLen(s)
	if vLen > width {
		if width > 3 {
			return ansi.Truncate(s, width, "...")
		}
		return ansi.Truncate(s, width, "")
	}
	return s
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ParseTimeJump parses time jumps like "1:30", "90", "90s", "50%".
func ParseTimeJump(input string, total time.Duration) (time.Duration, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return 0, fmt.Errorf("empty time string")
	}

	if strings.HasSuffix(input, "%") {
		var pct float64
		if _, err := fmt.Sscanf(input, "%f%%", &pct); err == nil {
			if pct < 0 {
				pct = 0
			}
			if pct > 100 {
				pct = 100
			}
			return time.Duration(float64(total) * (pct / 100.0)), nil
		}
	}

	if strings.Contains(input, ":") {
		parts := strings.Split(input, ":")
		if len(parts) == 2 {
			var m, s int
			if _, err := fmt.Sscanf(input, "%d:%d", &m, &s); err == nil {
				return time.Duration(m)*time.Minute + time.Duration(s)*time.Second, nil
			}
		} else if len(parts) == 3 {
			var h, m, s int
			if _, err := fmt.Sscanf(input, "%d:%d:%d", &h, &m, &s); err == nil {
				return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(s)*time.Second, nil
			}
		}
	}

	input = strings.TrimSuffix(input, "s")
	var secs float64
	if _, err := fmt.Sscanf(input, "%f", &secs); err == nil {
		return time.Duration(secs * float64(time.Second)), nil
	}

	return 0, fmt.Errorf("invalid time format (use e.g. 1:30, 90s, 50%%)")
}
