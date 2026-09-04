package views

import (
	"fmt"
	"strings"

	"github.com/KARTHIKKJ369/Tmusic/internal/audio"
	"github.com/KARTHIKKJ369/Tmusic/internal/tui/styles"
)

// LibraryView renders the full track list.
type LibraryView struct {
	Tracks    []audio.Track
	Cursor    int
	PlayingID string
	FavIDs    map[string]bool
	Width     int
	Height    int
	Offset    int // scroll offset
}

// View renders the library list.
func (v *LibraryView) View() string {
	if len(v.Tracks) == 0 {
		return styles.Muted.Render("\n  No tracks found in library.\n  Run: ") +
			styles.Accent.Render("muse --set-dir <path>")
	}

	visibleRows := v.Height - 2 // leave room for header
	if visibleRows < 1 {
		visibleRows = 1
	}

	// Adjust scroll offset so cursor is always visible.
	if v.Cursor < v.Offset {
		v.Offset = v.Cursor
	}
	if v.Cursor >= v.Offset+visibleRows {
		v.Offset = v.Cursor - visibleRows + 1
	}

	// Dynamic column calculations
	availW := v.Width - 16 // for badges, numbers, padding
	if availW < 30 {
		availW = 30
	}
	titleW := maxInt(availW*38/100, 16)
	artistW := maxInt(availW*28/100, 12)
	albumW := maxInt(availW*22/100, 10)
	durW := 6

	header := v.renderHeader(titleW, artistW, albumW, durW)

	var rows []string
	rows = append(rows, header)

	end := v.Offset + visibleRows
	if end > len(v.Tracks) {
		end = len(v.Tracks)
	}

	for i := v.Offset; i < end; i++ {
		t := v.Tracks[i]
		rows = append(rows, v.renderRow(i, t, titleW, artistW, albumW, durW))
	}

	return strings.Join(rows, "\n")
}

func (v *LibraryView) renderHeader(titleW, artistW, albumW, durW int) string {
	countStr := fmt.Sprintf("(%d tracks)", len(v.Tracks))
	hdr := fmt.Sprintf("  %s %s  %s  %s  %s  %s",
		Pad("♥", 2),
		Pad("#", 4),
		Pad("TITLE", titleW),
		Pad("ARTIST", artistW),
		Pad("ALBUM", albumW),
		Pad("DURATION", durW),
	)
	_ = countStr
	return styles.ListHeader.Render(hdr)
}

func (v *LibraryView) renderRow(i int, t audio.Track, titleW, artistW, albumW, durW int) string {
	isFav := v.FavIDs[t.ID]
	isPlaying := t.ID == v.PlayingID
	isSelected := i == v.Cursor

	fav := "  "
	if isFav {
		fav = styles.FavHeart.Render("♥ ")
	}

	num := Pad(fmt.Sprintf("%d", i+1), 4)
	title := Trunc(t.DisplayTitle(), titleW)
	artist := Trunc(t.Artist, artistW)
	album := Trunc(t.Album, albumW)
	dur := Pad(FormatDur(t.Duration), durW)

	var playPrefix string
	if isPlaying {
		playPrefix = styles.Playing.Render("▶ ")
	} else if isSelected {
		playPrefix = styles.Accent.Render("• ")
	} else {
		playPrefix = "  "
	}

	line := fmt.Sprintf("%s%s %s  %s  %s  %s  %s",
		playPrefix,
		fav,
		styles.Muted.Render(num),
		title,
		styles.Subtext.Render(artist),
		styles.Muted.Render(album),
		styles.Subtext.Render(dur),
	)

	switch {
	case isPlaying && isSelected:
		return styles.ListItemSelected.Render(line)
	case isPlaying:
		return styles.ListItemPlaying.Render(line)
	case isSelected:
		return styles.ListItemSelected.Render(line)
	default:
		return styles.ListItem.Render(line)
	}
}

// MoveUp moves the cursor up.
func (v *LibraryView) MoveUp() {
	if v.Cursor > 0 {
		v.Cursor--
	}
}

// MoveDown moves the cursor down.
func (v *LibraryView) MoveDown() {
	if v.Cursor < len(v.Tracks)-1 {
		v.Cursor++
	}
}

// Selected returns the currently selected track.
func (v *LibraryView) Selected() (audio.Track, bool) {
	if v.Cursor < 0 || v.Cursor >= len(v.Tracks) {
		return audio.Track{}, false
	}
	return v.Tracks[v.Cursor], true
}
