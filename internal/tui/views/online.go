package views

import (
	"fmt"
	"strings"

	"github.com/KARTHIKKJ369/Tmusic/internal/online"
	"github.com/KARTHIKKJ369/Tmusic/internal/tui/styles"
	"github.com/charmbracelet/lipgloss"
)

// OnlineView provides an on-demand online search & stream interface.
type OnlineView struct {
	Query          string
	LastQuery      string
	InputMode      bool
	HasSearched    bool
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
	if width < 30 {
		width = 30
	}
	var sb strings.Builder

	// 1. Single-line, non-wrapping Search Bar
	prompt := styles.Primary.Bold(true).Render(" ♫ ONLINE SEARCH: ")
	var searchBar string
	if v.InputMode {
		qText := v.Query
		if qText == "" {
			qText = " "
		}
		box := lipgloss.NewStyle().
			Background(styles.ColorSurfaceAlt).
			Foreground(styles.ColorAccent).
			Bold(true).
			Padding(0, 1).
			Render(qText + "█")
		hints := styles.Muted.Render("  [Enter] Search iTunes   [Esc] Browse Results")
		searchBar = prompt + box + hints
	} else {
		var queryDisplay string
		if v.LastQuery != "" {
			queryDisplay = styles.Accent.Bold(true).Render("\"" + v.LastQuery + "\"")
		} else if v.Query != "" {
			queryDisplay = styles.Accent.Bold(true).Render("\"" + v.Query + "\"")
		} else {
			queryDisplay = styles.Muted.Render("(press [/] to search)")
		}
		hints := styles.Muted.Render("  [/] Search   [Enter] Stream   [a] Add to Playlist   [s] Shuffle")
		searchBar = prompt + queryDisplay + hints
	}
	sb.WriteString(Trunc(searchBar, width) + "\n\n")

	// 2. Loading State
	if v.Loading {
		msg := v.LoadingMsg
		if msg == "" {
			msg = "Searching iTunes catalog..."
		}
		sb.WriteString(styles.Accent.Render(fmt.Sprintf("  ⠋ %s\n", msg)))
		return sb.String()
	}

	// 3. Error State
	if v.ErrorMsg != "" {
		sb.WriteString(styles.Danger.Render(fmt.Sprintf("  ⚠ %s\n\n", v.ErrorMsg)))
		sb.WriteString(styles.Muted.Render("  Press [/] to edit query and try again.\n"))
		return sb.String()
	}

	// 4. Empty Results or Initial Screen
	if len(v.Tracks) == 0 {
		if v.HasSearched && v.LastQuery != "" && !v.InputMode {
			sb.WriteString(styles.Muted.Render("  No tracks found matching \""+v.LastQuery+"\". Press [/] to try another search.\n\n"))
		} else {
			sb.WriteString(styles.Bold.Render("  Search & Stream Millions of Songs Keyless\n\n"))
			sb.WriteString(styles.Subtext.Render("  • Discovery: Apple iTunes Catalog (Official Metadata & 600×600 Artwork)\n"))
			sb.WriteString(styles.Subtext.Render("  • Streaming: Direct high-bitrate Opus/AAC audio via yt-dlp\n"))
			sb.WriteString(styles.Subtext.Render("  • Smart Cache: Instant 0ms playback on repeat listening\n\n"))
			if v.InputMode {
				sb.WriteString(styles.Amber.Render("  Type any artist, song, or album name above and press [Enter].\n"))
			} else {
				sb.WriteString(styles.Muted.Render("  Press [/] to begin searching.\n"))
			}
		}
		return sb.String()
	}

	// 5. Results Table
	availRows := height - 4
	if availRows < 4 {
		availRows = 4
	}

	tableW := width - 8
	if tableW < 30 {
		tableW = 30
	}

	titleW := tableW * 42 / 100
	if titleW < 12 {
		titleW = 12
	}
	artistW := tableW * 28 / 100
	if artistW < 10 {
		artistW = 10
	}
	durW := 6
	albumW := tableW - titleW - artistW - durW - 6
	if albumW < 8 {
		albumW = 8
	}

	// Header row
	header := fmt.Sprintf("    %-*s  %-*s  %-*s  %*s",
		titleW, "TITLE",
		artistW, "ARTIST",
		albumW, "ALBUM",
		durW, "TIME",
	)
	sb.WriteString(styles.Muted.Render(Trunc(header, width)) + "\n")

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

		line := fmt.Sprintf("  %s%-*s  %-*s  %-*s  %*s",
			prefix,
			titleW, titleTrunc,
			artistW, artistTrunc,
			albumW, albumTrunc,
			durW, durStr,
		)

		if i == v.Cursor {
			sb.WriteString(styles.ListItemSelected.Render(Trunc(line, width-2)) + "\n")
		} else {
			sb.WriteString(styles.ListItem.Render(Trunc(line, width-2)) + "\n")
		}
	}

	// Footer indicator
	if len(v.Tracks) > availRows {
		footer := fmt.Sprintf("  Showing %d-%d of %d online results", v.ScrollOffset+1, endIdx, len(v.Tracks))
		sb.WriteString(styles.Muted.Render(Trunc(footer, width)))
	}

	return sb.String()
}
