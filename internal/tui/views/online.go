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
	IsRecommended    bool
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

// MoveDown moves the track selection down.
func (v *OnlineView) MoveDown() {
	if len(v.Tracks) == 0 {
		return
	}
	if v.Cursor < len(v.Tracks)-1 {
		v.Cursor++
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
	if width < 10 {
		width = 10
	}
	var lines []string

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
		} else if v.IsRecommended {
			queryDisplay = styles.Accent.Bold(true).Render("Recommended for You")
		} else {
			queryDisplay = styles.Muted.Render("(press [/] to search)")
		}
		hints := styles.Muted.Render("  [/] Search   [Enter] Stream   [d] Download   [a] Add to Playlist   [s] Shuffle")
		searchBar = prompt + queryDisplay + hints
	}
	lines = append(lines, Trunc(searchBar, width))
	lines = append(lines, "")

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
		lines = append(lines, Trunc(sugHeader, width))
		for i := 0; i < availSugs; i++ {
			sug := v.Suggestions[i]
			var rendered string
			if i == v.SuggestionCursor {
				rendered = styles.ListItemSelected.MaxWidth(width).MaxHeight(1).Render(Trunc(fmt.Sprintf("  ▸ %s", sug), width-4))
			} else {
				rendered = styles.ListItem.MaxWidth(width).MaxHeight(1).Render(Trunc(fmt.Sprintf("    %s", sug), width-4))
			}
			lines = append(lines, Trunc(rendered, width))
		}
		return finishLines(lines, width, height)
	}

	// 3. Loading State
	if v.Loading {
		msg := v.LoadingMsg
		if msg == "" {
			msg = "Searching YouTube Music..."
		}
		lines = append(lines, Trunc(styles.Accent.Render(fmt.Sprintf("  ⠋ %s", msg)), width))
		return finishLines(lines, width, height)
	}

	// 4. Error State (never shown when search bar is empty or in input mode)
	if strings.TrimSpace(v.Query) == "" && strings.TrimSpace(v.LastQuery) == "" {
		v.ErrorMsg = ""
	}
	if v.ErrorMsg != "" && !v.InputMode && (strings.TrimSpace(v.Query) != "" || strings.TrimSpace(v.LastQuery) != "") {
		cleanErr := v.ErrorMsg
		if idx := strings.Index(cleanErr, "\n"); idx != -1 {
			cleanErr = cleanErr[:idx]
		}
		lines = append(lines, Trunc(styles.Danger.Render(fmt.Sprintf("  ⚠ %s", cleanErr)), width))
		lines = append(lines, "")
		if len(v.Tracks) == 0 {
			lines = append(lines, Trunc(styles.Muted.Render("  Press [/] to try another search."), width))
			return finishLines(lines, width, height)
		}
	}

	// 5. Empty Results or Initial Screen
	if len(v.Tracks) == 0 {
		if v.HasSearched && v.LastQuery != "" && !v.InputMode && v.ErrorMsg == "" {
			lines = append(lines, Trunc(styles.Muted.Render(fmt.Sprintf("  No tracks found matching \"%s\". Press [/] to try another search.", v.LastQuery)), width))
		}
		return finishLines(lines, width, height)
	}

	// 6. Results Table
	availRows := height - 4
	if availRows < 1 {
		availRows = 1
	}

	// Clamp cursor within bounds
	if v.Cursor < 0 {
		v.Cursor = 0
	}
	if len(v.Tracks) > 0 && v.Cursor >= len(v.Tracks) {
		v.Cursor = len(v.Tracks) - 1
	}

	// Dynamic scroll offset so cursor is always visible
	if v.Cursor < v.ScrollOffset {
		v.ScrollOffset = v.Cursor
	}
	if v.Cursor >= v.ScrollOffset+availRows {
		v.ScrollOffset = v.Cursor - availRows + 1
	}
	if v.ScrollOffset+availRows > len(v.Tracks) && len(v.Tracks) > availRows {
		v.ScrollOffset = len(v.Tracks) - availRows
	}
	if v.ScrollOffset < 0 {
		v.ScrollOffset = 0
	}

	availW := width - 14
	if availW < 24 {
		availW = 24
	}

	titleW := availW * 40 / 100
	if titleW < 12 {
		titleW = 12
	}
	artistW := availW * 28 / 100
	if artistW < 10 {
		artistW = 10
	}
	durW := 6
	albumW := availW - titleW - artistW - durW - 4
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
	lines = append(lines, styles.Muted.MaxWidth(width).MaxHeight(1).Render(Trunc(header, width-4)))

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

		var rowStyle lipgloss.Style
		var prefix string
		if isPlaying && i == v.Cursor {
			rowStyle = styles.ListItemSelected
			prefix = styles.Accent.Render("▶ ")
		} else if isPlaying {
			rowStyle = styles.ListItemPlaying
			prefix = styles.Accent.Render("▶ ")
		} else if i == v.Cursor {
			rowStyle = styles.ListItemSelected
			prefix = styles.Primary.Render("● ")
		} else {
			rowStyle = styles.ListItem
			prefix = "  "
		}

		line := fmt.Sprintf("  %s%s  %s  %s  %s",
			prefix,
			Pad(titleTrunc, titleW),
			Pad(artistTrunc, artistW),
			Pad(albumTrunc, albumW),
			Pad(durStr, durW),
		)

		renderedRow := rowStyle.MaxWidth(width).MaxHeight(1).Render(Trunc(line, width-4))
		lines = append(lines, Trunc(renderedRow, width))
	}

	// Footer indicator
	if len(v.Tracks) > availRows {
		label := "online results"
		if v.IsRecommended {
			label = "recommendations"
		}
		footer := fmt.Sprintf("  Showing %d-%d of %d %s   [Enter] Stream   [d] Download to Library", v.ScrollOffset+1, endIdx, len(v.Tracks), label)
		lines = append(lines, styles.Muted.MaxWidth(width).MaxHeight(1).Render(Trunc(footer, width)))
	}

	return finishLines(lines, width, height)
}

func finishLines(lines []string, width, height int) string {
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	for i, l := range lines {
		lines[i] = Trunc(l, width)
	}
	return strings.Join(lines, "\n")
}

