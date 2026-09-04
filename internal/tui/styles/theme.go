// Package styles defines the lipgloss colour palette, component styles, and layouts.
package styles

import "github.com/charmbracelet/lipgloss"

// Curated dark modern palette (Tokyo Night / Catppuccin inspired)
var (
	ColorBg        = lipgloss.AdaptiveColor{Dark: "#10101a", Light: "#f5f5f9"}
	ColorSurface   = lipgloss.AdaptiveColor{Dark: "#181828", Light: "#e8e8f0"}
	ColorSurfaceAlt= lipgloss.AdaptiveColor{Dark: "#1f1f33", Light: "#dddded"}
	ColorPrimary   = lipgloss.AdaptiveColor{Dark: "#ff4d6d", Light: "#d81159"} // Vibrant Coral Pink
	ColorSecondary = lipgloss.AdaptiveColor{Dark: "#7b2cbf", Light: "#5a189a"} // Deep Purple
	ColorAccent    = lipgloss.AdaptiveColor{Dark: "#00f5d4", Light: "#00bbf9"} // Neon Cyan
	ColorAmber     = lipgloss.AdaptiveColor{Dark: "#ffb703", Light: "#fb8500"} // Warm Amber
	ColorMuted     = lipgloss.AdaptiveColor{Dark: "#52527a", Light: "#9999bb"} // Subtle Grey-Blue
	ColorText      = lipgloss.AdaptiveColor{Dark: "#f0f2f5", Light: "#111118"} // Clean White
	ColorSubtext   = lipgloss.AdaptiveColor{Dark: "#a0a4c0", Light: "#555566"} // Soft Subtitle
	ColorFav       = lipgloss.AdaptiveColor{Dark: "#ff2a6d", Light: "#e60067"} // Glowing Heart Pink
	ColorPlaying   = lipgloss.AdaptiveColor{Dark: "#05ffa1", Light: "#00b471"} // Electric Green
	ColorBorder    = lipgloss.AdaptiveColor{Dark: "#2d2d48", Light: "#cbcbe0"} // Panel outline
	ColorActiveRow = lipgloss.AdaptiveColor{Dark: "#262642", Light: "#d6d6ec"}
)

// Base text styles
var (
	Base = lipgloss.NewStyle().
		Background(ColorBg).
		Foreground(ColorText)

	Bold    = lipgloss.NewStyle().Bold(true)
	Muted   = lipgloss.NewStyle().Foreground(ColorMuted)
	Subtext = lipgloss.NewStyle().Foreground(ColorSubtext)
	Primary = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	Accent  = lipgloss.NewStyle().Foreground(ColorAccent)
	Amber   = lipgloss.NewStyle().Foreground(ColorAmber)
	FavHeart= lipgloss.NewStyle().Foreground(ColorFav).Bold(true)
	Playing = lipgloss.NewStyle().Foreground(ColorPlaying).Bold(true)
	Danger  = lipgloss.NewStyle().Foreground(ColorFav).Bold(true)
)

// Tab styles with pill buttons
var (
	TabBar = lipgloss.NewStyle().
		Background(ColorSurface).
		Padding(0, 1)

	TabLogo = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		PaddingRight(2)

	TabActive = lipgloss.NewStyle().
		Background(ColorPrimary).
		Foreground(lipgloss.Color("#ffffff")).
		Bold(true).
		Padding(0, 1).
		MarginRight(1)

	TabInactive = lipgloss.NewStyle().
		Background(ColorSurfaceAlt).
		Foreground(ColorSubtext).
		Padding(0, 1).
		MarginRight(1)

	TabKeyNumber = lipgloss.NewStyle().
		Foreground(ColorAmber).
		Bold(true)
)

// List styles
var (
	ListHeader = lipgloss.NewStyle().
		Foreground(ColorSubtext).
		Bold(true).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1)

	ListItem = lipgloss.NewStyle().
		Padding(0, 1)

	ListItemSelected = lipgloss.NewStyle().
		Background(ColorActiveRow).
		Foreground(ColorText).
		Bold(true).
		Padding(0, 1)

	ListItemPlaying = lipgloss.NewStyle().
		Foreground(ColorPlaying).
		Bold(true).
		Padding(0, 1)
)

// Status & Progress bar
var (
	StatusBar = lipgloss.NewStyle().
		Background(ColorSurface).
		Foreground(ColorSubtext).
		Padding(0, 1)

	TrackName = lipgloss.NewStyle().
		Foreground(ColorText).
		Bold(true)

	ArtistName = lipgloss.NewStyle().
		Foreground(ColorSubtext)

	ProgressFilled = lipgloss.NewStyle().
		Foreground(ColorPrimary)

	ProgressEmpty = lipgloss.NewStyle().
		Foreground(ColorMuted)

	ProgressKnob = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)
)

// Badges & Pills
var (
	BadgeFormat = lipgloss.NewStyle().
		Background(ColorSecondary).
		Foreground(lipgloss.Color("#ffffff")).
		Bold(true).
		Padding(0, 1)

	BadgeAccent = lipgloss.NewStyle().
		Background(ColorAccent).
		Foreground(lipgloss.Color("#10101a")).
		Bold(true).
		Padding(0, 1)

	BadgeMuted = lipgloss.NewStyle().
		Background(ColorSurfaceAlt).
		Foreground(ColorSubtext).
		Padding(0, 1)

	KeyHint = lipgloss.NewStyle().
		Foreground(ColorAmber).
		Bold(true)
)

// Modal / Dialog / Panel
var (
	Panel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1)

	Modal = lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ColorPrimary).
		Background(ColorSurface).
		Padding(1, 2)

	HelpBar = lipgloss.NewStyle().
		Background(ColorBg).
		Foreground(ColorMuted).
		Padding(0, 1)

	Input = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorAccent).
		Padding(0, 1)
)
