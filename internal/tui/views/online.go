package views

import (
	"fmt"
	"strings"

	"github.com/KARTHIKKJ369/Tmusic/internal/online"
	"github.com/KARTHIKKJ369/Tmusic/internal/tui/styles"
)

// OnlineView provides an on-demand online search & stream interface.
type OnlineView struct {
	Query          string
	InputMode      bool
	Tracks         []online.ITunesTrack
	Cursor         int
	ScrollOffset   int
	Loading        bool
	LoadingMsg     string
	ErrorMsg       string
	PlayingTrackID int64
}

// NewOnlineView creates an initialised OnlineView.
func NewOnlineView() *OnlineView {
	return &OnlineView{
		InputMode: true,
	}
}

// MoveUp moves the track selection up.
func (v *OnlineView) MoveUp() {
	if len(v.Tracks) == 0 {
		return
	}
	if v.Cursor > 0 {
		v.Cursor--
		if v.Cursor < v.ScrollOffset {
			v.ScrollOffset = v.Cursor
		}
	}
}

// MoveDown moves the track selection down, constrained by visible window.
func (v *OnlineView) MoveDown(visibleHeight int) {
	if len(v.Tracks) == 0 {
		return
	}
	if v.Cursor < len(v.Tracks)-1 {
		v.Cursor++
		if v.Cursor >= v.ScrollOffset+visibleHeight {
			v.ScrollOffset = v.Cursor - visibleHeight + 1
		}
	}
}

// Selected returns the currently highlighted track.
func (v *OnlineView) Selected() *online.ITunesTrack {
	if len(v.Tracks) == 0 || v.Cursor < 0 || v.Cursor >= len(v.Tracks) {
		return nil
	}
	return &v.Tracks[v.Cursor]
}

// View renders the Online search and results interface.
func (v *OnlineView) View(width, height int) string {
	var sb strings.Builder

	// 1. Search Bar Header
	var searchBar string
	prompt := styles.Primary.Bold(true).Render(" 󰠃 ONLINE SEARCH: ")
	if v.InputMode {
		queryText := v.Query + "█"
		searchBar = prompt + styles.Input.Render(queryText) + "  " + styles.Muted.Render("[Enter] Search  [Esc] Results")
	} else {
		queryText := v.Query
		if queryText == "" {
			queryText = styles.Muted.Render("press [/] to search millions of tracks...")
		}
		searchBar = prompt + queryText + "  " + styles.Muted.Render("[/] Search  [Enter] Stream  [a] Add to Playlist")
	}
	sb.WriteString(searchBar + "\n\n")

	// 2. Loading State or Error
	if v.Loading {
		msg := v.LoadingMsg
		if msg == "" {
			msg = "Searching iTunes..."
		}
		sb.WriteString(styles.Accent.Render(fmt.Sprintf("  ⠋ %s\n", msg)))
		return sb.String()
	}

	if v.ErrorMsg != "" {
		sb.WriteString(styles.Danger.Render(fmt.Sprintf("  ⚠ %s\n\n", v.ErrorMsg)))
		sb.WriteString(styles.Muted.Render("  Press [/] to edit query and try again.\n"))
		return sb.String()
	}

	if len(v.Tracks) == 0 {
		if v.Query != "" {
			sb.WriteString(styles.Muted.Render("  No tracks found matching \""+v.Query+"\". Try another search.\n\n"))
		} else {
			sb.WriteString(styles.Bold.Render("  Search & Stream Millions of Songs Keyless\n\n"))
			sb.WriteString(styles.Subtext.Render("  • Discovery: iTunes Catalog (Official Metadata & 600×600 Artwork)\n"))
			sb.WriteString(styles.Subtext.Render("  • Streaming: Direct high-bitrate Opus/AAC audio via yt-dlp\n"))
			sb.WriteString(styles.Subtext.Render("  • Smart Cache: Instant 0ms playback on repeat listening\n\n"))
			sb.WriteString(styles.Muted.Render("  Type any artist, song, or album name above and hit [Enter].\n"))
		}
		return sb.String()
	}

	// 3. Results Table
	availRows := height - 4
	if availRows < 4 {
		availRows = 4
	}

	// Column widths
	titleW := width * 38 / 100
	if titleW < 16 {
		titleW = 16
	}
	artistW := width * 26 / 100
	if artistW < 12 {
		artistW = 12
	}
	albumW := width * 20 / 100
	if albumW < 10 {
		albumW = 10
	}

	// Header row
	header := fmt.Sprintf("    %-*s  %-*s  %-*s  %6s",
		titleW, "TITLE",
		artistW, "ARTIST",
		albumW, "ALBUM",
		"TIME",
	)
	sb.WriteString(styles.Muted.Render(header) + "\n")

	endIdx := v.ScrollOffset + availRows
	if endIdx > len(v.Tracks) {
		endIdx = len(v.Tracks)
	}

	for i := v.ScrollOffset; i < endIdx; i++ {
		t := v.Tracks[i]
		durStr := FormatDur(t.Duration())

		titleTrunc := Trunc(t.TrackName, titleW)
		artistTrunc := Trunc(t.ArtistName, artistW)
		albumTrunc := Trunc(t.CollectionName, albumW)

		isPlaying := v.PlayingTrackID > 0 && v.PlayingTrackID == t.TrackID

		var prefix string
		if isPlaying {
			prefix = styles.Accent.Render("▶ ")
		} else if i == v.Cursor {
			prefix = styles.Primary.Render("● ")
		} else {
			prefix = "  "
		}

		line := fmt.Sprintf("%s%-*s  %-*s  %-*s  %6s",
			prefix,
			titleW, titleTrunc,
			artistW, artistTrunc,
			albumW, albumTrunc,
			durStr,
		)

		if i == v.Cursor {
			sb.WriteString(styles.ListItemSelected.Render(" "+line) + "\n")
		} else {
			sb.WriteString(styles.ListItem.Render(" "+line) + "\n")
		}
	}

	// Footer indicator
	if len(v.Tracks) > availRows {
		footer := fmt.Sprintf("  Showing %d-%d of %d online results", v.ScrollOffset+1, endIdx, len(v.Tracks))
		sb.WriteString("\n" + styles.Muted.Render(footer))
	}

	return sb.String()
}
