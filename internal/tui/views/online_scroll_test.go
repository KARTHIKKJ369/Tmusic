package views

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/KARTHIKKJ369/Tmusic/internal/online"
	"github.com/charmbracelet/lipgloss"
)

func TestOnlineViewScrolling(t *testing.T) {
	v := NewOnlineView()
	v.InputMode = false

	// Create 25 mock tracks, including Indic/Malayalam titles
	for i := 0; i < 25; i++ {
		title := fmt.Sprintf("Track %d English Title", i+1)
		if i == 9 || i == 10 {
			title = "പുതുമഴയായ് പൊഴിയാം"
		}
		v.Tracks = append(v.Tracks, online.OnlineTrack{
			ID:       fmt.Sprintf("track_%d", i+1),
			Title:    title,
			Artist:   "Millennium Musics & Artists",
			Album:    "YouTube Music",
			Duration: 4 * time.Minute,
		})
	}

	termW := 80
	termH := 20
	// In app.go: content = m.onlineView.View(m.width, h) where h = maxInt(m.height-3, 1)
	contentH := termH - 3 // 17 lines

	// Initial render
	rendered := v.View(termW, contentH)
	lines := strings.Split(rendered, "\n")
	if len(lines) > contentH {
		t.Fatalf("rendered lines count %d exceeds contentH %d", len(lines), contentH)
	}
	for idx, l := range lines {
		if lipgloss.Width(l) > termW {
			t.Fatalf("Line %d width %d exceeds termW %d: %q", idx, lipgloss.Width(l), termW, l)
		}
	}

	// Move down 15 times
	for i := 0; i < 15; i++ {
		v.MoveDown()
		out := v.View(termW, contentH)
		outLines := strings.Split(out, "\n")
		if len(outLines) > contentH {
			t.Fatalf("Step down %d: rendered lines count %d exceeds contentH %d", i, len(outLines), contentH)
		}
		for lineIdx, l := range outLines {
			if lipgloss.Width(l) > termW {
				t.Fatalf("Step down %d: Line %d width %d exceeds termW %d", i, lineIdx, lipgloss.Width(l), termW)
			}
		}
	}

	if v.Cursor != 15 {
		t.Fatalf("Expected Cursor=15, got %d", v.Cursor)
	}

	// Move up 10 times
	for i := 0; i < 10; i++ {
		v.MoveUp()
		out := v.View(termW, contentH)
		outLines := strings.Split(out, "\n")
		if len(outLines) > contentH {
			t.Fatalf("Step up %d: rendered lines count %d exceeds contentH %d", i, len(outLines), contentH)
		}
		for lineIdx, l := range outLines {
			if lipgloss.Width(l) > termW {
				t.Fatalf("Step up %d: Line %d width %d exceeds termW %d", i, lineIdx, lipgloss.Width(l), termW)
			}
		}
	}

	if v.Cursor != 5 {
		t.Fatalf("Expected Cursor=5, got %d", v.Cursor)
	}

	// Move all the way to 0
	for i := 0; i < 10; i++ {
		v.MoveUp()
	}
	if v.Cursor != 0 {
		t.Fatalf("Expected Cursor=0, got %d", v.Cursor)
	}
	if v.ScrollOffset != 0 {
		t.Fatalf("Expected ScrollOffset=0, got %d", v.ScrollOffset)
	}
}

func TestMalayalamPadAndTrunc(t *testing.T) {
	s := "പുതുമഴയായ് പൊഴിയാം"
	padded := Pad(s, 40)
	if VisibleLen(padded) > 40 {
		t.Fatalf("padded visible len %d > 40: %q", VisibleLen(padded), padded)
	}
	t.Logf("Pad(40): %q (VisibleLen=%d)", padded, VisibleLen(padded))

	truncated := Trunc(s, 10)
	if VisibleLen(truncated) > 10 {
		t.Fatalf("truncated visible len %d > 10: %q", VisibleLen(truncated), truncated)
	}
	t.Logf("Trunc(10): %q (VisibleLen=%d)", truncated, VisibleLen(truncated))
}




