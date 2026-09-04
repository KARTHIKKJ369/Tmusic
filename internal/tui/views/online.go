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
	Query            string
	LastQuery        string
	InputMode        bool
	HasSearched      bool
	Tracks           []online.OnlineTrack
	Cursor           int
	ScrollOffset     int
	Loading          bool
	LoadingMsg       string
	ErrorMsg         string
	PlayingTrackID   string
	Suggestions      []string
	SuggestionCursor int // -1 when none selected
}

// NewOnlineView creates an initialised OnlineView.
func NewOnlineView() *OnlineView {
	return &OnlineView{
		InputMode:        true,
		SuggestionCursor: -1,
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

// SuggestionUp moves selection up in the suggestions dropdown.
func (v *OnlineView) SuggestionUp() {
	if len(v.Suggestions) == 0 {
		return
	}
	if v.SuggestionCursor > 0 {
		v.SuggestionCursor--
	} else {
		v.SuggestionCursor = len(v.Suggestions) - 1
	}
}

// SuggestionDown moves selection down in the suggestions dropdown.
func (v *OnlineView) SuggestionDown() {
	if len(v.Suggestions) == 0 {
		return
	}
	if v.SuggestionCursor < len(v.Suggestions)-1 {
		v.SuggestionCursor++
	} else {
		v.SuggestionCursor = 0
	}
}

// SelectedSuggestion returns the currently highlighted suggestion text, or empty if none.
func (v *OnlineView) SelectedSuggestion() string {
	if v.SuggestionCursor >= 0 && v.SuggestionCursor < len(v.Suggestions) {
		return v.Suggestions[v.SuggestionCursor]
	}
	return ""
}

// Selected returns the currently highlighted track.
func (v *OnlineView) Selected() *online.OnlineTrack {
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

	// 1. Sleek Search Bar Header
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
		hints := styles.Muted.Render("  [Enter] Search   [↑↓] Pick Suggestion   [Tab] Fill   [Esc] Browse")
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
		hints := styles.Muted.Render("  [/] Search   [Enter] Stream   [d] Download   [a] Add to Playlist   [s] Shuffle")
		searchBar = prompt + queryDisplay + hints
	}
	sb.WriteString(Trunc(searchBar, width) + "\n\n")

	// 2. Suggestions Dropdown (when typing and suggestions are available)
	if v.InputMode && len(v.Suggestions) > 0 {
		availSugs := height - 4
		if availSugs > len(v.Suggestions) {
			availSugs = len(v.Suggestions)
		}
		if availSugs > 6 {
			availSugs = 6
		}
		sugHeader := styles.Muted.Render("  SUGGESTIONS (↑↓ navigate, Tab autocomplete, Enter search):")
		sb.WriteString(Trunc(sugHeader, width) + "\n")
		for i := 0; i < availSugs; i++ {
			sug := v.Suggestions[i]
			var line string
			if i == v.SuggestionCursor {
				line = styles.ListItemSelected.Render(fmt.Sprintf("  ▸ %-*s", width-8, sug))
			} else {
				line = styles.ListItem.Render(fmt.Sprintf("    %-*s", width-8, sug))
			}
			sb.WriteString(Trunc(line, width) + "\n")
		}
		sb.WriteString("\n")
		return sb.String()
	}

	// 3. Loading State
	if v.Loading {
		msg := v.LoadingMsg
		if msg == "" {
			msg = "Searching YouTube Music..."
		}
		sb.WriteString(styles.Accent.Render(fmt.Sprintf("  ⠋ %s\n", msg)))
		return sb.String()
	}

	// 4. Error State
	if v.ErrorMsg != "" {
		sb.WriteString(styles.Danger.Render(fmt.Sprintf("  ⚠ %s\n\n", v.ErrorMsg)))
		sb.WriteString(styles.Muted.Render("  Press [/] to try another search.\n"))
		return sb.String()
	}

	// 5. Empty Results or Initial Screen (Clean Left-Aligned Orientation)
	if len(v.Tracks) == 0 {
		if v.HasSearched && v.LastQuery != "" && !v.InputMode {
			sb.WriteString(styles.Muted.Render(fmt.Sprintf("  No tracks found matching \"%s\". Press [/] to try another search.\n\n", v.LastQuery)))
		} else {
			sb.WriteString(styles.Bold.Render("  Search & Stream Millions of Songs Keyless\n\n"))
			sb.WriteString(styles.Subtext.Render("  • Discovery: YouTube Music & Apple iTunes Catalog\n"))
			sb.WriteString(styles.Subtext.Render("  • Streaming: Direct high-bitrate Opus/AAC audio via yt-dlp\n"))
			sb.WriteString(styles.Subtext.Render("  • Continuous Play: Auto-queues related songs & radio\n"))
			sb.WriteString(styles.Subtext.Render("  • Offline Download: Press [d] on any song to save to your local library\n\n"))
			if v.InputMode {
				sb.WriteString(styles.Amber.Render("  Type any artist, song, or album name above and press [Enter].\n"))
			} else {
				sb.WriteString(styles.Muted.Render("  Press [/] to begin searching.\n"))
			}
		}
		return sb.String()
	}

	// 6. Results Table
	availRows := height - 4
	if availRows < 4 {
		availRows = 4
	}

	tableW := width - 8
	if tableW < 30 {
		tableW = 30
	}

	titleW := tableW * 44 / 100
	if titleW < 12 {
		titleW = 12
	}
	artistW := tableW * 30 / 100
	if artistW < 10 {
		artistW = 10
	}
	durW := 6
	albumW := tableW - titleW - artistW - durW - 6
	if albumW < 8 {
		albumW = 8
	}

	// Header row
	header := fmt.Sprintf("    %s  %s  %s  %s",
		Pad("TITLE", titleW),
		Pad("ARTIST", artistW),
		Pad("ALBUM", albumW),
		Pad("TIME", durW),
	)
	sb.WriteString(styles.Muted.Render(Trunc(header, width)) + "\n")

	endIdx := v.ScrollOffset + availRows
	if endIdx > len(v.Tracks) {
		endIdx = len(v.Tracks)
	}

	for i := v.ScrollOffset; i < endIdx; i++ {
		t := v.Tracks[i]
		durStr := FormatDur(t.Duration)

		titleTrunc := Trunc(t.Title, titleW)
		artistTrunc := Trunc(t.Artist, artistW)
		albumTrunc := Trunc(t.Album, albumW)

		isPlaying := v.PlayingTrackID != "" && v.PlayingTrackID == t.ID

		var prefix string
		if isPlaying {
			prefix = styles.Accent.Render("▶ ")
		} else if i == v.Cursor {
			prefix = styles.Primary.Render("● ")
		} else {
			prefix = "  "
		}

		line := fmt.Sprintf("  %s%s  %s  %s  %s",
			prefix,
			Pad(titleTrunc, titleW),
			Pad(artistTrunc, artistW),
			Pad(albumTrunc, albumW),
			Pad(durStr, durW),
		)

		if i == v.Cursor {
			sb.WriteString(styles.ListItemSelected.Render(Trunc(line, width-2)) + "\n")
		} else {
			sb.WriteString(styles.ListItem.Render(Trunc(line, width-2)) + "\n")
		}
	}

	// Footer indicator
	if len(v.Tracks) > availRows {
		footer := fmt.Sprintf("  Showing %d-%d of %d online results   [Enter] Stream   [d] Download to Library", v.ScrollOffset+1, endIdx, len(v.Tracks))
		sb.WriteString(styles.Muted.Render(Trunc(footer, width)))
	}

	return sb.String()
}
