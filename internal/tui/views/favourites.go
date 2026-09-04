package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/karthikjayan/muse/internal/audio"
	"github.com/karthikjayan/muse/internal/tui/styles"
)

// FavouritesView renders the favourites list.
type FavouritesView struct {
	Tracks    []audio.Track // all tracks
	FavIDs    map[string]bool
	PlayingID string
	Cursor    int
	Width     int
	Height    int
	Offset    int
}

// FavTracks returns only the favourited tracks.
func (v *FavouritesView) FavTracks() []audio.Track {
	var out []audio.Track
	for _, t := range v.Tracks {
		if v.FavIDs[t.ID] {
			out = append(out, t)
		}
	}
	return out
}

func (v *FavouritesView) TotalDuration() time.Duration {
	var total time.Duration
	for _, t := range v.FavTracks() {
		total += t.Duration
	}
	return total
}

func (v *FavouritesView) View() string {
	favs := v.FavTracks()
	if len(favs) == 0 {
		return styles.Muted.Render("\n  No favourites yet.\n\n") +
			"  In Library, press " + styles.FavHeart.Render("[f]") + " to heart any track and save it here!"
	}

	visibleRows := v.Height - 2
	if visibleRows < 1 {
		visibleRows = 1
	}

	if v.Cursor < v.Offset {
		v.Offset = v.Cursor
	}
	if v.Cursor >= v.Offset+visibleRows {
		v.Offset = v.Cursor - visibleRows + 1
	}

	availW := v.Width - 16
	if availW < 30 {
		availW = 30
	}
	titleW := maxInt(availW*40/100, 16)
	artistW := maxInt(availW*30/100, 12)
	albumW := maxInt(availW*20/100, 10)
	durW := 6

	summary := fmt.Sprintf("♥ %d FAVOURITED TRACKS · %s", len(favs), FormatDur(v.TotalDuration()))
	hdr := fmt.Sprintf("  %s %s  %s  %s  %s  %s",
		Pad("♥", 2),
		Pad("#", 4),
		Pad("TITLE", titleW),
		Pad("ARTIST", artistW),
		Pad("ALBUM", albumW),
		Pad("DURATION", durW),
	)
	_ = summary

	var rows []string
	rows = append(rows, styles.ListHeader.Render(hdr))

	end := v.Offset + visibleRows
	if end > len(favs) {
		end = len(favs)
	}

	for i := v.Offset; i < end; i++ {
		t := favs[i]
		isPlaying := t.ID == v.PlayingID
		isSelected := i == v.Cursor

		num := Pad(fmt.Sprintf("%d", i+1), 4)
		title := Trunc(t.DisplayTitle(), titleW)
		artist := Trunc(t.Artist, artistW)
		album := Trunc(t.Album, albumW)
		dur := Pad(FormatDur(t.Duration), durW)

		var prefix string
		if isPlaying {
			prefix = styles.Playing.Render("▶ ")
		} else if isSelected {
			prefix = styles.Accent.Render("• ")
		} else {
			prefix = "  "
		}

		line := fmt.Sprintf("%s%s %s  %s  %s  %s  %s",
			prefix,
			styles.FavHeart.Render("♥ "),
			styles.Muted.Render(num),
			title,
			styles.Subtext.Render(artist),
			styles.Muted.Render(album),
			styles.Subtext.Render(dur),
		)

		switch {
		case isPlaying && isSelected:
			rows = append(rows, styles.ListItemSelected.Render(line))
		case isPlaying:
			rows = append(rows, styles.ListItemPlaying.Render(line))
		case isSelected:
			rows = append(rows, styles.ListItemSelected.Render(line))
		default:
			rows = append(rows, styles.ListItem.Render(line))
		}
	}

	return strings.Join(rows, "\n")
}

func (v *FavouritesView) MoveUp() {
	if v.Cursor > 0 {
		v.Cursor--
	}
}

func (v *FavouritesView) MoveDown() {
	favs := v.FavTracks()
	if v.Cursor < len(favs)-1 {
		v.Cursor++
	}
}

func (v *FavouritesView) Selected() (audio.Track, bool) {
	favs := v.FavTracks()
	if v.Cursor < 0 || v.Cursor >= len(favs) {
		return audio.Track{}, false
	}
	return favs[v.Cursor], true
}
