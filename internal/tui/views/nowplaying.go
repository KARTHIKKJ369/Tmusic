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

	// Calculate optimal artwork dimensions (aspect ratio 2:1 columns per row for exact 1:1 square)
	maxArtH := availH - 9
	if maxArtH > 18 {
		maxArtH = 18
	}
	if maxArtH < 4 {
		maxArtH = 4
	}

	artH := maxArtH
	artW := int(float64(artH) * 2.0)
	if artW > v.Width-8 {
		artW = maxInt(v.Width-8, 10)
		artH = maxInt(artW/2, 4)
	}

	// 1. Centered Cover Art
	artLines := RenderCoverArt(v.CoverData, artW, artH, v.Tick, !v.Paused)
	for _, line := range artLines {
		sb.WriteString(Center(line, v.Width) + "\n")
	}
	sb.WriteString("\n")

	// 2. Track Metadata Centered directly below the artwork
	maxTextW := minInt(maxInt(v.Width-10, 20), 60)

	// Title (large, bold, vibrant coral pink)
	title := Trunc(v.Track.DisplayTitle(), maxTextW)
	sb.WriteString(Center(styles.Primary.Bold(true).Render(title), v.Width) + "\n")

	// Artist (bold, neon cyan)
	artist := v.Track.Artist
	if artist == "" {
		artist = "Unknown Artist"
	}
	sb.WriteString(Center(styles.Accent.Bold(true).Render(Trunc(artist, maxTextW)), v.Width) + "\n")

	// Album & Badges
	var metaParts []string
	if v.Track.Album != "" {
		albumStr := v.Track.Album
		if v.Track.Year > 0 {
			albumStr += fmt.Sprintf(" (%d)", v.Track.Year)
		}
		metaParts = append(metaParts, styles.Subtext.Render(Trunc(albumStr, 30)))
	}
	metaParts = append(metaParts, FormatBadge(v.Track.Path))
	if v.IsFav {
		metaParts = append(metaParts, styles.FavHeart.Render("♥"))
	}
	if v.Track.Genre != "" && maxTextW > 35 {
		metaParts = append(metaParts, styles.BadgeMuted.Render(Trunc(v.Track.Genre, 14)))
	}
	sb.WriteString(Center(strings.Join(metaParts, "  "), v.Width) + "\n\n")

	// 3. Dynamic Audio Spectrum Visualizer Equalizer (if height permits)
	if availH >= 20 {
		eqW := minInt(maxInt(v.Width*45/100, 16), 36)
		if eqW > 8 {
			eq := SpectrumEqualizer(eqW, v.Tick, !v.Paused)
			sb.WriteString(Center(eq, v.Width) + "\n\n")
		}
	}

	// 4. High-Precision Progress Bar
	total := v.Track.Duration
	var ratio float64
	if total > 0 {
		ratio = float64(v.Position) / float64(total)
	}
	barWidth := minInt(maxInt(v.Width*50/100, 16), 46)
	sb.WriteString(Center(ProgressBar(ratio, barWidth, v.Position, total), v.Width) + "\n\n")

	// 5. Playback Controls
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

	// 6. Queue Position / Shuffle / Repeat / Volume Status Line
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
	// Trim trailing empty lines if overflowing
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
