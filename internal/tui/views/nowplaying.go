package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/karthikjayan/muse/internal/audio"
	"github.com/karthikjayan/muse/internal/config"
	"github.com/karthikjayan/muse/internal/tui/styles"
)

// NowPlayingView is the centered album art & metadata panel with bottom controls.
type NowPlayingView struct {
	Track     *audio.Track
	CoverData []byte // embedded picture bytes (JPEG / PNG)
	Position  time.Duration
	Paused    bool
	Volume    float64
	Shuffle   bool
	Repeat    config.RepeatMode
	IsFav     bool
	QueuePos  int
	QueueLen  int
	Tick      int64
	Width     int
	Height    int
}

func (v *NowPlayingView) View() string {
	if v.Track == nil {
		msg := "\n\n" + styles.Bold.Render("NO TRACK PLAYING") + "\n\n" +
			styles.Muted.Render("Go to ") + styles.Amber.Render("[1] Library") +
			styles.Muted.Render(" or ") + styles.Amber.Render("[3] Favourites") +
			styles.Muted.Render(" and press ") + styles.Accent.Render("[Enter]") + " to start playback."
		return Center(msg, v.Width)
	}

	var sb strings.Builder

	// Dimensions for artwork
	artW := 24
	artH := 12
	if v.Width < 60 {
		artW = 18
		artH = 9
	}
	artLines := RenderCoverArt(v.CoverData, artW, artH, v.Tick, !v.Paused)

	// Build metadata lines
	metaLines := v.buildMetadataLines(v.Width - artW - 8)

	// If wide enough (>= 60 cols), place Artwork on Left and Metadata on Right
	if v.Width >= 60 {
		combinedRows := combineArtAndMeta(artLines, metaLines, artW, v.Width)
		sb.WriteString("\n")
		sb.WriteString(combinedRows)
		sb.WriteString("\n\n")
	} else {
		// Narrow terminal: stack artwork then metadata centered
		sb.WriteString("\n")
		for _, al := range artLines {
			sb.WriteString(Center(al, v.Width) + "\n")
		}
		sb.WriteString("\n")
		for _, ml := range metaLines {
			sb.WriteString(Center(ml, v.Width) + "\n")
		}
		sb.WriteString("\n")
	}

	// Dynamic Equalizer visualizer
	eqW := minInt(v.Width-16, 44)
	if eqW > 8 {
		eq := SpectrumEqualizer(eqW, v.Tick, !v.Paused)
		sb.WriteString(Center(eq, v.Width) + "\n\n")
	}

	// High-resolution Progress Bar
	total := v.Track.Duration
	var ratio float64
	if total > 0 {
		ratio = float64(v.Position) / float64(total)
	}
	barWidth := minInt(maxInt(v.Width*60/100, 24), 60)
	sb.WriteString(Center(ProgressBar(ratio, barWidth, v.Position, total), v.Width) + "\n\n")

	// Sleek emoji-free playback controls
	var playBtn string
	if v.Paused {
		playBtn = styles.BadgeAccent.Render(" PLAY [Space] ")
	} else {
		playBtn = styles.BadgeFormat.Render(" PAUSE [Space] ")
	}

	prevBtn := styles.BadgeMuted.Render(" PREV [p] ")
	nextBtn := styles.BadgeMuted.Render(" NEXT [n] ")
	seekBack := styles.Muted.Render("[-5s ←]")
	seekFwd := styles.Muted.Render("[+5s →]")

	controlsRow := fmt.Sprintf("%s  %s  %s  %s  %s", prevBtn, seekBack, playBtn, seekFwd, nextBtn)
	sb.WriteString(Center(controlsRow, v.Width) + "\n\n")

	// Status line with Queue, Shuffle, Repeat, Volume
	queuePill := styles.BadgeMuted.Render(fmt.Sprintf("TRACK %d/%d", v.QueuePos+1, v.QueueLen))

	var shufflePill string
	if v.Shuffle {
		shufflePill = styles.BadgeAccent.Render("SHUFFLE: ON")
	} else {
		shufflePill = styles.BadgeMuted.Render("SHUFFLE: OFF")
	}

	var repeatPill string
	switch v.Repeat {
	case config.RepeatTrack:
		repeatPill = styles.BadgeAccent.Render("REPEAT: TRACK")
	case config.RepeatQueue:
		repeatPill = styles.BadgeAccent.Render("REPEAT: QUEUE")
	default:
		repeatPill = styles.BadgeMuted.Render("REPEAT: OFF")
	}

	volPill := VolumeBar(v.Volume, 8)

	statusLine := fmt.Sprintf("%s   %s   %s   %s", queuePill, shufflePill, repeatPill, volPill)
	sb.WriteString(Center(statusLine, v.Width))

	return sb.String()
}

func (v *NowPlayingView) buildMetadataLines(maxW int) []string {
	if maxW < 20 {
		maxW = 20
	}
	var lines []string

	// Track Title
	title := Trunc(v.Track.DisplayTitle(), maxW)
	lines = append(lines, styles.Primary.Bold(true).Render(title))

	// Artist
	if v.Track.Artist != "" {
		lines = append(lines, styles.Accent.Bold(true).Render(Trunc(v.Track.Artist, maxW)))
	} else {
		lines = append(lines, styles.Muted.Render("Unknown Artist"))
	}

	// Album
	if v.Track.Album != "" {
		albumStr := v.Track.Album
		if v.Track.Year > 0 {
			albumStr += fmt.Sprintf(" (%d)", v.Track.Year)
		}
		lines = append(lines, styles.Subtext.Render(Trunc(albumStr, maxW)))
	}

	// Genre / Format Badges
	formatBadge := FormatBadge(v.Track.Path)
	var favBadge string
	if v.IsFav {
		favBadge = " " + styles.FavHeart.Render("[♥ LIKED]")
	}

	var genreBadge string
	if v.Track.Genre != "" {
		genreBadge = " " + styles.BadgeMuted.Render(Trunc(v.Track.Genre, 16))
	}

	lines = append(lines, formatBadge+favBadge+genreBadge)

	return lines
}

func combineArtAndMeta(artLines, metaLines []string, artW, termW int) string {
	maxRows := maxInt(len(artLines), len(metaLines))
	artPad := 4
	totalContentW := artW + artPad + 36

	leftMargin := 0
	if termW > totalContentW {
		leftMargin = (termW - totalContentW) / 2
	}
	marginStr := strings.Repeat(" ", leftMargin)

	var sb strings.Builder
	for i := 0; i < maxRows; i++ {
		var a string
		if i < len(artLines) {
			a = artLines[i]
		} else {
			a = strings.Repeat(" ", artW)
		}

		var m string
		metaIdx := i - (maxRows-len(metaLines))/2 // vertically center metadata relative to art
		if metaIdx >= 0 && metaIdx < len(metaLines) {
			m = metaLines[metaIdx]
		}

		sb.WriteString(fmt.Sprintf("%s%s%s%s\n", marginStr, a, strings.Repeat(" ", artPad), m))
	}

	return strings.TrimRight(sb.String(), "\n")
}
