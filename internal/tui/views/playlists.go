package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/KARTHIKKJ369/Tmusic/internal/audio"
	"github.com/KARTHIKKJ369/Tmusic/internal/playlist"
	"github.com/KARTHIKKJ369/Tmusic/internal/tui/styles"
)

// PlaylistsView renders a dual-pane playlist manager.
type PlaylistsView struct {
	Playlists      []playlist.Playlist
	Tracks         []audio.Track // full track index for metadata lookup
	FavIDs         map[string]bool
	PlayingID      string
	Cursor         int  // which playlist selected in left pane
	TrackCursor    int  // which track selected in right pane
	FocusRight     bool // whether focus is on the right (track list) pane
	Width, Height  int
	InputMode      bool
	InputValue     string
}

func NewPlaylistsView() *PlaylistsView {
	return &PlaylistsView{}
}

func (v *PlaylistsView) View() string {
	if len(v.Playlists) == 0 {
		msg := "\n  " + styles.Bold.Render("No playlists yet.") + "\n\n" +
			"  Press " + styles.Amber.Render("[c]") + " to create your first playlist.\n" +
			"  In Library, press " + styles.Amber.Render("[a]") + " on any song to add it to a playlist."
		if v.InputMode {
			box := styles.Modal.Render(
				styles.Bold.Render("Create New Playlist") + "\n\n" +
					styles.Input.Render(v.InputValue+"█") + "\n\n" +
					styles.Muted.Render("[Enter] Confirm  [Esc] Cancel"),
			)
			return Center(box, v.Width)
		}
		return msg
	}

	leftW := maxInt(v.Width*32/100, 24)
	rightW := maxInt(v.Width-leftW-3, 30)
	paneH := maxInt(v.Height-2, 5)

	leftContent := v.renderLeftPane(leftW, paneH)
	rightContent := v.renderRightPane(rightW, paneH)

	divider := styles.Muted.Render("│")

	leftLines := strings.Split(leftContent, "\n")
	rightLines := strings.Split(rightContent, "\n")

	var rows []string
	maxLines := maxInt(len(leftLines), len(rightLines))
	for i := 0; i < maxLines; i++ {
		l := ""
		r := ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r = rightLines[i]
		}
		l = Pad(l, leftW)
		rows = append(rows, fmt.Sprintf("%s %s %s", l, divider, r))
	}

	out := strings.Join(rows, "\n")

	if v.InputMode {
		modalBox := styles.Modal.Render(
			styles.Bold.Render(" Create New Playlist ") + "\n\n" +
				styles.Input.Render(v.InputValue+"█") + "\n\n" +
				styles.Muted.Render("[Enter] Confirm  [Esc] Cancel"),
		)
		return out + "\n" + Center(modalBox, v.Width)
	}

	return out
}

func (v *PlaylistsView) renderLeftPane(width, height int) string {
	hdr := styles.ListHeader.Render(Pad(" PLAYLISTS", width))
	var lines []string
	lines = append(lines, hdr)

	for i, pl := range v.Playlists {
		isSelected := i == v.Cursor && !v.FocusRight
		isCurrent := i == v.Cursor

		badge := styles.BadgeMuted.Render(fmt.Sprintf("%d", len(pl.TrackIDs)))
		nameW := width - VisibleLen(badge) - 5
		name := Trunc(pl.Name, nameW)

		var line string
		if isCurrent {
			prefix := styles.Accent.Render("▶ ")
			line = fmt.Sprintf("%s%s %s", prefix, name, badge)
		} else {
			line = fmt.Sprintf("  %s %s", name, badge)
		}

		if isSelected {
			lines = append(lines, styles.ListItemSelected.Render(Pad(line, width)))
		} else if isCurrent {
			lines = append(lines, styles.ListItem.Bold(true).Render(Pad(line, width)))
		} else {
			lines = append(lines, styles.ListItem.Render(Pad(line, width)))
		}
	}

	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (v *PlaylistsView) renderRightPane(width, height int) string {
	if v.Cursor < 0 || v.Cursor >= len(v.Playlists) {
		return styles.Muted.Render("  No playlist selected")
	}

	pl := v.Playlists[v.Cursor]
	totalDur := v.playlistDuration(pl)
	summary := fmt.Sprintf("%s · %s", styles.Bold.Render(pl.Name), styles.Subtext.Render(FormatDur(totalDur)))
	hdr := styles.ListHeader.Render(Pad(summary, width))

	var lines []string
	lines = append(lines, hdr)

	if len(pl.TrackIDs) == 0 {
		lines = append(lines, styles.Muted.Render("\n  Playlist is empty. Press [a] on songs in Library to add."))
		for len(lines) < height {
			lines = append(lines, "")
		}
		return strings.Join(lines, "\n")
	}

	titleW := maxInt(width*50/100, 16)
	artistW := maxInt(width*30/100, 12)

	for j, id := range pl.TrackIDs {
		t := v.trackByID(id)
		isSelected := j == v.TrackCursor && v.FocusRight
		isPlaying := t.ID == v.PlayingID

		num := styles.Muted.Render(Pad(fmt.Sprintf("%d", j+1), 3))
		title := Trunc(t.DisplayTitle(), titleW)
		artist := Trunc(t.Artist, artistW)
		dur := styles.Subtext.Render(Pad(FormatDur(t.Duration), 6))

		var prefix string
		if isPlaying {
			prefix = styles.Playing.Render("▶ ")
		} else if isSelected {
			prefix = styles.Accent.Render("• ")
		} else {
			prefix = "  "
		}

		row := fmt.Sprintf("%s%s %s  %s  %s", prefix, num, title, styles.Subtext.Render(artist), dur)

		if isPlaying && isSelected {
			lines = append(lines, styles.ListItemSelected.Render(Pad(row, width)))
		} else if isPlaying {
			lines = append(lines, styles.ListItemPlaying.Render(Pad(row, width)))
		} else if isSelected {
			lines = append(lines, styles.ListItemSelected.Render(Pad(row, width)))
		} else {
			lines = append(lines, styles.ListItem.Render(Pad(row, width)))
		}
	}

	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (v *PlaylistsView) playlistDuration(pl playlist.Playlist) time.Duration {
	var total time.Duration
	for _, id := range pl.TrackIDs {
		t := v.trackByID(id)
		total += t.Duration
	}
	return total
}

func (v *PlaylistsView) trackByID(id string) audio.Track {
	for _, t := range v.Tracks {
		if t.ID == id {
			return t
		}
	}
	return audio.Track{ID: id, Title: id}
}

func (v *PlaylistsView) MoveUp() {
	if v.FocusRight {
		if v.TrackCursor > 0 {
			v.TrackCursor--
		}
	} else {
		if v.Cursor > 0 {
			v.Cursor--
			v.TrackCursor = 0
		}
	}
}

func (v *PlaylistsView) MoveDown() {
	if v.FocusRight {
		if v.Cursor >= 0 && v.Cursor < len(v.Playlists) {
			if v.TrackCursor < len(v.Playlists[v.Cursor].TrackIDs)-1 {
				v.TrackCursor++
			}
		}
	} else {
		if v.Cursor < len(v.Playlists)-1 {
			v.Cursor++
			v.TrackCursor = 0
		}
	}
}

func (v *PlaylistsView) TogglePane() {
	v.FocusRight = !v.FocusRight
}

func (v *PlaylistsView) SelectedTrack() (audio.Track, bool) {
	if v.Cursor < 0 || v.Cursor >= len(v.Playlists) {
		return audio.Track{}, false
	}
	pl := v.Playlists[v.Cursor]
	if len(pl.TrackIDs) == 0 {
		return audio.Track{}, false
	}
	if v.TrackCursor < 0 || v.TrackCursor >= len(pl.TrackIDs) {
		v.TrackCursor = 0
	}
	trackID := pl.TrackIDs[v.TrackCursor]
	return v.trackByID(trackID), true
}

func (v *PlaylistsView) SelectedPlaylistTracks() []audio.Track {
	if v.Cursor < 0 || v.Cursor >= len(v.Playlists) {
		return nil
	}
	pl := v.Playlists[v.Cursor]
	var out []audio.Track
	for _, id := range pl.TrackIDs {
		out = append(out, v.trackByID(id))
	}
	return out
}
