package views

import (
	"strings"

	"github.com/KARTHIKKJ369/Tmusic/internal/audio"
	"github.com/KARTHIKKJ369/Tmusic/internal/tui/styles"
)

// SearchView is an overlay for fuzzy track search.
type SearchView struct {
	Query     string
	Results   []audio.Track
	Cursor    int
	PlayingID string
	FavIDs    map[string]bool
	Active    bool
}

// Filter updates Results by matching Query against tracks.
func (v *SearchView) Filter(all []audio.Track) {
	if v.Query == "" {
		v.Results = all
		v.Cursor = 0
		return
	}
	q := strings.ToLower(v.Query)
	var out []audio.Track
	for _, t := range all {
		if strings.Contains(strings.ToLower(t.DisplayTitle()), q) ||
			strings.Contains(strings.ToLower(t.Artist), q) ||
			strings.Contains(strings.ToLower(t.Album), q) {
			out = append(out, t)
		}
	}
	v.Results = out
	v.Cursor = 0
}

func (v *SearchView) View(width int) string {
	var sb strings.Builder

	// Search input
	sb.WriteString(styles.Input.Render("/ " + v.Query + "█"))
	sb.WriteString("\n")

	if len(v.Results) == 0 {
		sb.WriteString(styles.Muted.Render("  No results"))
		return sb.String()
	}

	limit := 12
	if len(v.Results) < limit {
		limit = len(v.Results)
	}

	for i := 0; i < limit; i++ {
		t := v.Results[i]
		fav := "  "
		if v.FavIDs[t.ID] {
			fav = styles.FavHeart.Render("♥ ")
		}
		line := fav + Trunc(t.DisplayTitle(), width/3) + "  " +
			styles.Subtext.Render(Trunc(t.Artist, width/4))
		if i == v.Cursor {
			sb.WriteString(styles.ListItemSelected.Render("  "+line) + "\n")
		} else {
			sb.WriteString(styles.ListItem.Render("  "+line) + "\n")
		}
	}

	return sb.String()
}

func (v *SearchView) MoveUp() {
	if v.Cursor > 0 {
		v.Cursor--
	}
}

func (v *SearchView) MoveDown() {
	if v.Cursor < len(v.Results)-1 {
		v.Cursor++
	}
}

func (v *SearchView) Selected() (audio.Track, bool) {
	if v.Cursor < 0 || v.Cursor >= len(v.Results) {
		return audio.Track{}, false
	}
	return v.Results[v.Cursor], true
}
