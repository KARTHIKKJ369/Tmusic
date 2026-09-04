package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/KARTHIKKJ369/Tmusic/internal/audio"
	"github.com/KARTHIKKJ369/Tmusic/internal/config"
	"github.com/KARTHIKKJ369/Tmusic/internal/tui/styles"
)

// NowPlayingView is the centered album art & metadata panel with responsive controls.
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
			styles.Muted.Render(" and press ") + styles.Accent.Render("[Enter]") + " to start playback.\n\n" +
			styles.Muted.Render("Or press ") + styles.Accent.Render("[s]") + styles.Muted.Render(" to shuffle and play random music.")
		return Center(msg, v.Width)
	}

	availH := v.Height
	if availH < 5 {
		availH = 5
	}

	var sb strings.Builder

	// Determine if side-by-side or stacked
	isWide := v.Width >= 55 && availH >= 12

	var artH, artW int
	if isWide {
		// Calculate optimal square art dimensions
		// Each character row is 2 vertical pixels, so artW = artH * 2 gives a 1:1 square
		maxArtH := (availH - 6) // leave room for progress bar, controls, status
		if maxArtH > 16 {
			maxArtH = 16
		}
		if maxArtH < 8 {
			maxArtH = 8
		}
		artH = maxArtH
		artW = artH * 2
		if artW > v.Width-25 {
			artW = maxInt(v.Width-25, 16)
			artH = artW / 2
		}
	} else {
		// Compact stacked
		artH = minInt(maxInt((availH-8)/2, 5), 8)
		artW = artH * 2
	}

	artLines := RenderCoverArt(v.CoverData, artW, artH, v.Tick, !v.Paused)
	metaLines := v.buildMetadataLines(v.Width - artW - 6)

	if isWide {
		combinedRows := combineArtAndMeta(artLines, metaLines, artW, v.Width)
		sb.WriteString(combinedRows)
		sb.WriteString("\n\n")
	} else {
		for _, al := range artLines {
			sb.WriteString(Center(al, v.Width) + "\n")
		}
		sb.WriteString("\n")
		for _, ml := range metaLines {
			sb.WriteString(Center(ml, v.Width) + "\n")
		}
		sb.WriteString("\n")
	}

	// Dynamic Equalizer visualizer (only when height >= 20)
	if availH >= 20 {
		eqW := minInt(maxInt(v.Width*45/100, 16), 40)
		if eqW > 8 {
			eq := SpectrumEqualizer(eqW, v.Tick, !v.Paused)
			sb.WriteString(Center(eq, v.Width) + "\n\n")
		}
	}

	// High-resolution Progress Bar
	total := v.Track.Duration
	var ratio float64
	if total > 0 {
		ratio = float64(v.Position) / float64(total)
	}
	barWidth := minInt(maxInt(v.Width*55/100, 16), 50)
	sb.WriteString(Center(ProgressBar(ratio, barWidth, v.Position, total), v.Width) + "\n\n")

	// Sleek playback controls (compact when narrow)
	var playBtn string
	if v.Paused {
		playBtn = styles.BadgeAccent.Render(" PLAY [Space] ")
	} else {
		playBtn = styles.BadgeFormat.Render(" PAUSE [Space] ")
	}

	var controlsRow string
	if v.Width >= 55 {
		prevBtn := styles.BadgeMuted.Render(" PREV [p] ")
		nextBtn := styles.BadgeMuted.Render(" NEXT [n] ")
		seekBack := styles.Muted.Render("[-5s ←]")
		seekFwd := styles.Muted.Render("[+5s →]")
		controlsRow = fmt.Sprintf("%s  %s  %s  %s  %s", prevBtn, seekBack, playBtn, seekFwd, nextBtn)
	} else {
		controlsRow = fmt.Sprintf("%s  %s  %s", styles.BadgeMuted.Render(" [p] "), playBtn, styles.BadgeMuted.Render(" [n] "))
	}
	sb.WriteString(Center(controlsRow, v.Width) + "\n")

	// Status line with Queue, Shuffle, Repeat, Volume (if room permits)
	if availH >= 16 {
		sb.WriteString("\n")
		queuePill := styles.BadgeMuted.Render(fmt.Sprintf("%d/%d", v.QueuePos+1, v.QueueLen))

		var shufflePill string
		if v.Shuffle {
			shufflePill = styles.BadgeAccent.Render("SHUF")
		} else {
			shufflePill = styles.BadgeMuted.Render("SHUF:OFF")
		}

		var repeatPill string
		switch v.Repeat {
		case config.RepeatTrack:
			repeatPill = styles.BadgeAccent.Render("REP:1")
		case config.RepeatQueue:
			repeatPill = styles.BadgeAccent.Render("REP:ALL")
		default:
			repeatPill = styles.BadgeMuted.Render("REP:OFF")
		}

		volPill := VolumeBar(v.Volume, 6)
		statusLine := fmt.Sprintf("%s  %s  %s  %s", queuePill, shufflePill, repeatPill, volPill)
		sb.WriteString(Center(statusLine, v.Width))
	}

	raw := sb.String()
	lines := strings.Split(raw, "\n")
	// Trim empty lines from bottom if overflowing
	for len(lines) > availH {
		if lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		} else {
			break
		}
	}
	if len(lines) > availH {
		lines = lines[:availH]
	}

	return strings.Join(lines, "\n")
}

func (v *NowPlayingView) buildMetadataLines(maxW int) []string {
	if maxW < 18 {
		maxW = 18
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

	// Format / Fav Badges
	formatBadge := FormatBadge(v.Track.Path)
	var favBadge string
	if v.IsFav {
		favBadge = " " + styles.FavHeart.Render("[♥ LIKED]")
	}

	var genreBadge string
	if v.Track.Genre != "" && maxW > 24 {
		genreBadge = " " + styles.BadgeMuted.Render(Trunc(v.Track.Genre, 14))
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
		metaIdx := i - (maxRows-len(metaLines))/2
		if metaIdx >= 0 && metaIdx < len(metaLines) {
			m = metaLines[metaIdx]
		}

		sb.WriteString(fmt.Sprintf("%s%s%s%s\n", marginStr, a, strings.Repeat(" ", artPad), m))
	}

	return strings.TrimRight(sb.String(), "\n")
}
