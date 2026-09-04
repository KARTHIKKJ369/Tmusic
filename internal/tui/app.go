// Package tui is the root bubbletea application model.
package tui

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/KARTHIKKJ369/Tmusic/internal/audio"
	"github.com/KARTHIKKJ369/Tmusic/internal/config"
	"github.com/KARTHIKKJ369/Tmusic/internal/library"
	"github.com/KARTHIKKJ369/Tmusic/internal/playlist"
	"github.com/KARTHIKKJ369/Tmusic/internal/tui/styles"
	"github.com/KARTHIKKJ369/Tmusic/internal/tui/views"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Tab indices
const (
	TabLibrary    = 0
	TabPlaylists  = 1
	TabFavs       = 2
	TabNowPlaying = 3
)

var tabNames = []string{"1 Library", "2 Playlists", "3 Favourites", "4 Now Playing"}

// tickMsg is fired by the player's tick event.
type tickMsg struct{}

// trackEndMsg signals the current track finished.
type trackEndMsg struct{}

// Model is the root bubbletea model.
type Model struct {
	cfg    config.Config
	player *audio.Player
	index  *library.Index
	pm     *playlist.Manager
	queue  *playlist.Queue

	// TUI state
	activeTab  int
	width      int
	height     int
	showHelp   bool
	showSearch bool
	status     string // transient status message
	tickCount  int64  // counter for dynamic animations
	prevVolume float64

	// Playlist picker modal
	showPlaylistPicker  bool
	playlistPickTrack   audio.Track
	playlistPickCursor  int

	// Time jump modal (g or :)
	showTimeJumpModal bool
	timeJumpInput     string

	// Views
	libView  *views.LibraryView
	plView   *views.PlaylistsView
	favView  *views.FavouritesView
	npView   *views.NowPlayingView
	srchView *views.SearchView

	// Playback state (mirrors player for rendering without locking)
	currentTrack *audio.Track
	currentCover []byte
	position     time.Duration
	paused       bool
}

// New creates and initialises the Model.
func New(cfg config.Config, idx *library.Index, pm *playlist.Manager, player *audio.Player) *Model {
	tracks := idx.All()
	q := playlist.NewQueue(tracks)

	m := &Model{
		cfg:        cfg,
		player:     player,
		index:      idx,
		pm:         pm,
		queue:      q,
		prevVolume: cfg.Volume,

		libView: &views.LibraryView{
			Tracks: tracks,
			FavIDs: pm.Favourites,
		},
		plView: views.NewPlaylistsView(),
		favView: &views.FavouritesView{
			Tracks: tracks,
			FavIDs: pm.Favourites,
		},
		npView: &views.NowPlayingView{
			Repeat:  cfg.Repeat,
			Shuffle: cfg.Shuffle,
			Volume:  cfg.Volume,
		},
		srchView: &views.SearchView{},
	}

	m.plView.Playlists = pm.Playlists
	m.plView.Tracks = tracks
	m.plView.FavIDs = pm.Favourites

	if cfg.Shuffle {
		q.Shuffle()
	}

	return m
}

// Init starts the event listener goroutine.
func (m *Model) Init() tea.Cmd {
	return listenForPlayerEvents(m.player)
}

// listenForPlayerEvents waits for a player event and converts it to a tea.Msg.
func listenForPlayerEvents(p *audio.Player) tea.Cmd {
	return func() tea.Msg {
		evt := <-p.Events
		switch evt.Type {
		case audio.EventTrackEnd:
			return trackEndMsg{}
		case audio.EventTick:
			return tickMsg{}
		}
		return tickMsg{}
	}
}

// Update handles all messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateViewSizes()
		return m, nil

	case tickMsg:
		m.tickCount++
		m.position = m.player.Position()
		m.syncNPView()
		return m, listenForPlayerEvents(m.player)

	case trackEndMsg:
		cmd := m.handleTrackEnd()
		return m, tea.Batch(cmd, listenForPlayerEvents(m.player))

	case tea.MouseMsg:
		if msg.Type == tea.MouseLeft && msg.Y == 0 {
			// Click on tab header area
			colWidth := m.width / 4
			if colWidth > 0 {
				tab := msg.X / colWidth
				if tab >= 0 && tab <= 3 {
					m.activeTab = tab
				}
			}
		}

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *Model) handleTrackEnd() tea.Cmd {
	switch m.cfg.Repeat {
	case config.RepeatTrack:
		return m.playCurrentQueueTrack()
	case config.RepeatQueue:
		if !m.queue.HasNext() {
			m.queue.Unshuffle()
			if m.cfg.Shuffle {
				m.queue.Shuffle()
			}
		}
		return m.advanceQueue()
	default:
		return m.advanceQueue()
	}
}

func (m *Model) advanceQueue() tea.Cmd {
	if t, ok := m.queue.Next(); ok {
		return m.playTrack(t)
	}
	m.currentTrack = nil
	m.syncNPView()
	return nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Search overlay intercepts all keys
	if m.showSearch {
		return m.handleSearchKey(msg)
	}

	// Playlist picker dialog
	if m.showPlaylistPicker {
		return m.handlePlaylistPickerKey(msg)
	}

	// Time jump dialog (g or :)
	if m.showTimeJumpModal {
		return m.handleTimeJumpKey(msg)
	}

	// Playlist creation input mode
	if m.activeTab == TabPlaylists && m.plView.InputMode {
		return m.handlePlaylistInput(msg)
	}

	key := msg.String()

	// Direct percentage jumps in Now Playing tab (0..9)
	if m.activeTab == TabNowPlaying && len(key) == 1 && key[0] >= '0' && key[0] <= '9' && m.currentTrack != nil {
		pct := float64(key[0]-'0') * 0.10
		_ = m.player.SeekPercent(pct)
		targetDur := time.Duration(float64(m.currentTrack.Duration) * pct)
		m.status = fmt.Sprintf("Seeked to %d%% (%s)", int(pct*100), views.FormatDur(targetDur))
		m.position = m.player.Position()
		m.syncNPView()
		return m, nil
	}

	switch key {
	// Quit
	case "q", "ctrl+c":
		m.player.Stop()
		_ = m.pm.Save()
		_ = config.Save(m.cfg)
		return m, tea.Quit

	// Direct tab navigation
	case "1":
		m.activeTab = TabLibrary
	case "2":
		m.activeTab = TabPlaylists
	case "3":
		m.activeTab = TabFavs
	case "4":
		m.activeTab = TabNowPlaying

	// Tab cycling
	case "tab", "]":
		if m.activeTab == TabPlaylists {
			m.plView.TogglePane()
		} else {
			m.activeTab = (m.activeTab + 1) % 4
		}
	case "shift+tab", "[":
		if m.activeTab == TabPlaylists {
			m.plView.TogglePane()
		} else {
			m.activeTab = (m.activeTab + 3) % 4
		}

	// Help toggle
	case "?":
		m.showHelp = !m.showHelp

	// Time jump prompt (e.g. g or :)
	case "g", ":":
		if m.currentTrack != nil {
			m.showTimeJumpModal = true
			m.timeJumpInput = ""
		} else {
			m.status = "Start playing a track first"
		}

	// Search
	case "/":
		m.showSearch = true
		m.srchView.Active = true
		m.srchView.Query = ""
		m.srchView.Filter(m.index.All())
		return m, nil

	// Play/Pause
	case " ":
		if m.currentTrack == nil {
			return m, m.playSelected()
		}
		m.paused = m.player.TogglePause()
		m.syncNPView()

	// Enter — play selected
	case "enter":
		return m, m.playSelected()

	// Next / Prev track
	case "n":
		return m, m.advanceQueue()
	case "N", "p":
		if t, ok := m.queue.Prev(); ok {
			return m, m.playTrack(t)
		}

	// Seek & navigation controls
	case "right":
		if m.activeTab == TabPlaylists && !m.plView.FocusRight {
			m.plView.FocusRight = true
		} else {
			_ = m.player.Seek(5 * time.Second)
		}
	case "left":
		if m.activeTab == TabPlaylists && m.plView.FocusRight {
			m.plView.FocusRight = false
		} else {
			_ = m.player.Seek(-5 * time.Second)
		}
	case "h":
		if m.activeTab == TabPlaylists {
			m.plView.FocusRight = false
		}
	case "l":
		if m.activeTab == TabPlaylists {
			m.plView.FocusRight = true
		}
	case "shift+right", "L":
		_ = m.player.Seek(30 * time.Second)
	case "shift+left", "H":
		_ = m.player.Seek(-30 * time.Second)

	// Volume
	case "+", "=":
		v := m.player.Volume() + 0.05
		m.player.SetVolume(v)
		m.cfg.Volume = m.player.Volume()
		m.npView.Volume = m.cfg.Volume
	case "-":
		v := m.player.Volume() - 0.05
		m.player.SetVolume(v)
		m.cfg.Volume = m.player.Volume()
		m.npView.Volume = m.cfg.Volume
	case "m":
		// Mute toggle
		if m.player.Volume() > 0 {
			m.prevVolume = m.player.Volume()
			m.player.SetVolume(0)
			m.status = "Muted"
		} else {
			vol := m.prevVolume
			if vol <= 0 {
				vol = 0.8
			}
			m.player.SetVolume(vol)
			m.status = fmt.Sprintf("Volume: %d%%", int(vol*100))
		}
		m.cfg.Volume = m.player.Volume()
		m.npView.Volume = m.cfg.Volume

	// Shuffle & Random Play
	case "s":
		if m.currentTrack == nil {
			// Nothing playing yet: shuffle library and start playing a fresh random track immediately
			m.queue = playlist.NewQueue(m.index.All())
			m.queue.Shuffle() // Starts from a uniformly random track at pos 0
			if t, ok := m.queue.Current(); ok {
				m.cfg.Shuffle = true
				m.activeTab = TabNowPlaying
				m.status = "SHUFFLE: Started random playback"
				return m, m.playTrack(t)
			}
		} else {
			// Toggle shuffle on current queue preserving the currently playing track
			m.cfg.Shuffle = !m.cfg.Shuffle
			if m.cfg.Shuffle {
				m.queue.Shuffle(m.currentTrack.ID)
				m.status = "SHUFFLE: ON (De-clustered)"
			} else {
				m.queue.Unshuffle(m.currentTrack.ID)
				m.status = "SHUFFLE: OFF"
			}
			m.npView.Shuffle = m.cfg.Shuffle
		}

	// Force pick new random track from library & jump to Now Playing (S or z)
	case "S", "z":
		m.queue = playlist.NewQueue(m.index.All())
		m.queue.Shuffle() // Fresh random starting track
		if t, ok := m.queue.Current(); ok {
			m.cfg.Shuffle = true
			m.activeTab = TabNowPlaying
			m.status = "SHUFFLE: Started random playback"
			return m, m.playTrack(t)
		}

	// Repeat
	case "r":
		switch m.cfg.Repeat {
		case config.RepeatOff:
			m.cfg.Repeat = config.RepeatTrack
			m.status = "REPEAT: TRACK"
		case config.RepeatTrack:
			m.cfg.Repeat = config.RepeatQueue
			m.status = "REPEAT: QUEUE"
		case config.RepeatQueue:
			m.cfg.Repeat = config.RepeatOff
			m.status = "REPEAT: OFF"
		}
		m.npView.Repeat = m.cfg.Repeat

	// Favourite
	case "f":
		m.toggleFavSelected()

	// Open original album art in system photo viewer
	case "o", "O":
		if len(m.npView.CoverData) > 0 {
			title := "cover"
			if m.currentTrack != nil {
				title = m.currentTrack.Title
			}
			if err := openCoverArt(m.npView.CoverData, title); err == nil {
				m.status = "Cover art opened in system photo viewer"
			} else {
				m.status = fmt.Sprintf("Failed to open cover: %v", err)
			}
		} else {
			m.status = "No embedded album art found in track"
		}

	// Add track to playlist
	case "a":
		m.openPlaylistPicker()

	// Create playlist
	case "c":
		if m.activeTab == TabPlaylists {
			m.plView.InputMode = true
			m.plView.InputValue = ""
		} else {
			m.activeTab = TabPlaylists
			m.plView.InputMode = true
			m.plView.InputValue = ""
		}

	// Delete in Playlists tab
	case "x", "d":
		if m.activeTab == TabPlaylists {
			m.handlePlaylistDelete()
		}

	// Navigation
	case "up", "k":
		m.moveUp()
	case "down", "j":
		m.moveDown()

	case "esc":
		m.showHelp = false
		m.showPlaylistPicker = false
		m.showTimeJumpModal = false
	}

	return m, nil
}

func (m *Model) handleTimeJumpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.showTimeJumpModal = false
		m.timeJumpInput = ""
	case "enter":
		if m.currentTrack != nil {
			target, err := views.ParseTimeJump(m.timeJumpInput, m.currentTrack.Duration)
			if err == nil {
				_ = m.player.SeekTo(target)
				m.status = fmt.Sprintf("Jumped to %s", views.FormatDur(target))
				m.position = m.player.Position()
				m.syncNPView()
			} else {
				m.status = fmt.Sprintf("Invalid time jump: %v", err)
			}
		}
		m.showTimeJumpModal = false
		m.timeJumpInput = ""
	case "backspace":
		if len(m.timeJumpInput) > 0 {
			m.timeJumpInput = m.timeJumpInput[:len(m.timeJumpInput)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.timeJumpInput += msg.String()
		}
	}
	return m, nil
}

func (m *Model) openPlaylistPicker() {
	var tr audio.Track
	var ok bool
	switch m.activeTab {
	case TabLibrary:
		tr, ok = m.libView.Selected()
	case TabFavs:
		tr, ok = m.favView.Selected()
	case TabNowPlaying:
		if m.currentTrack != nil {
			tr = *m.currentTrack
			ok = true
		}
	case TabPlaylists:
		tr, ok = m.plView.SelectedTrack()
	}

	if !ok {
		m.status = "Select a track first"
		return
	}

	if len(m.pm.Playlists) == 0 {
		m.status = "No playlists exist. Press [c] to create one first!"
		return
	}

	m.playlistPickTrack = tr
	m.playlistPickCursor = 0
	m.showPlaylistPicker = true
}

func (m *Model) handlePlaylistPickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.showPlaylistPicker = false
	case "up", "k":
		if m.playlistPickCursor > 0 {
			m.playlistPickCursor--
		}
	case "down", "j":
		if m.playlistPickCursor < len(m.pm.Playlists)-1 {
			m.playlistPickCursor++
		}
	case "enter":
		if m.playlistPickCursor >= 0 && m.playlistPickCursor < len(m.pm.Playlists) {
			m.pm.AddTrack(m.playlistPickCursor, m.playlistPickTrack.ID)
			_ = m.pm.Save()
			m.plView.Playlists = m.pm.Playlists
			m.status = fmt.Sprintf("Added to playlist '%s'", m.pm.Playlists[m.playlistPickCursor].Name)
		}
		m.showPlaylistPicker = false
	}
	return m, nil
}

func (m *Model) handlePlaylistDelete() {
	if m.plView.FocusRight {
		// Delete selected track from playlist
		if m.plView.Cursor >= 0 && m.plView.Cursor < len(m.pm.Playlists) {
			pl := m.pm.Playlists[m.plView.Cursor]
			if m.plView.TrackCursor >= 0 && m.plView.TrackCursor < len(pl.TrackIDs) {
				trackID := pl.TrackIDs[m.plView.TrackCursor]
				m.pm.RemoveTrack(m.plView.Cursor, trackID)
				_ = m.pm.Save()
				m.plView.Playlists = m.pm.Playlists
				m.status = "Removed track from playlist"
			}
		}
	} else {
		// Delete selected playlist
		if m.plView.Cursor >= 0 && m.plView.Cursor < len(m.pm.Playlists) {
			name := m.pm.Playlists[m.plView.Cursor].Name
			m.pm.Delete(m.plView.Cursor)
			_ = m.pm.Save()
			m.plView.Playlists = m.pm.Playlists
			if m.plView.Cursor >= len(m.pm.Playlists) {
				m.plView.Cursor = maxInt(0, len(m.pm.Playlists)-1)
			}
			m.status = fmt.Sprintf("Deleted playlist '%s'", name)
		}
	}
}

func (m *Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.showSearch = false
		m.srchView.Active = false
	case "enter":
		if t, ok := m.srchView.Selected(); ok {
			m.showSearch = false
			m.srchView.Active = false
			return m, m.playTrack(t)
		}
	case "up":
		m.srchView.MoveUp()
	case "down":
		m.srchView.MoveDown()
	case "backspace":
		if len(m.srchView.Query) > 0 {
			m.srchView.Query = m.srchView.Query[:len(m.srchView.Query)-1]
			m.srchView.Filter(m.index.All())
		}
	default:
		if len(msg.String()) == 1 {
			m.srchView.Query += msg.String()
			m.srchView.Filter(m.index.All())
		}
	}
	return m, nil
}

func (m *Model) handlePlaylistInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.plView.InputMode = false
		m.plView.InputValue = ""
	case "enter":
		name := strings.TrimSpace(m.plView.InputValue)
		if name != "" {
			m.pm.Create(name)
			_ = m.pm.Save()
			m.plView.Playlists = m.pm.Playlists
			m.status = fmt.Sprintf("Created playlist '%s'", name)
		}
		m.plView.InputMode = false
		m.plView.InputValue = ""
	case "backspace":
		if len(m.plView.InputValue) > 0 {
			m.plView.InputValue = m.plView.InputValue[:len(m.plView.InputValue)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.plView.InputValue += msg.String()
		}
	}
	return m, nil
}

func (m *Model) moveUp() {
	switch m.activeTab {
	case TabLibrary:
		m.libView.MoveUp()
	case TabPlaylists:
		m.plView.MoveUp()
	case TabFavs:
		m.favView.MoveUp()
	}
}

func (m *Model) moveDown() {
	switch m.activeTab {
	case TabLibrary:
		m.libView.MoveDown()
	case TabPlaylists:
		m.plView.MoveDown()
	case TabFavs:
		m.favView.MoveDown()
	}
}

func (m *Model) playSelected() tea.Cmd {
	switch m.activeTab {
	case TabLibrary:
		if t, ok := m.libView.Selected(); ok {
			m.queue = playlist.NewQueue(m.index.All())
			if m.cfg.Shuffle {
				m.queue.Shuffle(t.ID)
			} else {
				m.queue.JumpTo(t.ID)
			}
			return m.playTrack(t)
		}
	case TabFavs:
		favs := m.favView.FavTracks()
		if len(favs) > 0 {
			if t, ok := m.favView.Selected(); ok {
				m.queue = playlist.NewQueue(favs)
				if m.cfg.Shuffle {
					m.queue.Shuffle(t.ID)
				} else {
					m.queue.JumpTo(t.ID)
				}
				return m.playTrack(t)
			}
		}
	case TabPlaylists:
		plTracks := m.plView.SelectedPlaylistTracks()
		if len(plTracks) > 0 {
			var startTrack audio.Track
			if m.plView.FocusRight {
				if tr, ok := m.plView.SelectedTrack(); ok {
					startTrack = tr
				} else {
					startTrack = plTracks[0]
				}
			} else {
				startTrack = plTracks[0]
			}
			m.queue = playlist.NewQueue(plTracks)
			if m.cfg.Shuffle {
				m.queue.Shuffle(startTrack.ID)
			} else {
				m.queue.JumpTo(startTrack.ID)
			}
			return m.playTrack(startTrack)
		}
	}
	return nil
}

func (m *Model) toggleFavSelected() {
	var id string
	switch m.activeTab {
	case TabLibrary:
		if t, ok := m.libView.Selected(); ok {
			id = t.ID
		}
	case TabFavs:
		if t, ok := m.favView.Selected(); ok {
			id = t.ID
		}
	case TabPlaylists:
		if t, ok := m.plView.SelectedTrack(); ok {
			id = t.ID
		}
	default:
		if m.currentTrack != nil {
			id = m.currentTrack.ID
		}
	}
	if id == "" {
		return
	}
	on := m.pm.ToggleFavourite(id)
	if on {
		m.status = "Added to favourites ♥"
	} else {
		m.status = "Removed from favourites"
	}
	_ = m.pm.Save()
}

func (m *Model) playTrack(t audio.Track) tea.Cmd {
	return func() tea.Msg {
		if err := m.player.Load(t.Path); err != nil {
			m.status = fmt.Sprintf("Error: %v", err)
			return trackEndMsg{}
		}
		imgBytes, _ := audio.ExtractPicture(t.Path)
		m.currentTrack = &t
		m.currentCover = imgBytes
		m.paused = false
		m.position = 0
		m.libView.PlayingID = t.ID
		m.favView.PlayingID = t.ID
		m.srchView.PlayingID = t.ID
		m.plView.PlayingID = t.ID
		m.syncNPView()
		m.player.Play()
		return tickMsg{}
	}
}

func (m *Model) playCurrentQueueTrack() tea.Cmd {
	if t, ok := m.queue.Current(); ok {
		return m.playTrack(t)
	}
	return nil
}

func (m *Model) syncNPView() {
	m.npView.Track = m.currentTrack
	m.npView.CoverData = m.currentCover
	m.npView.Position = m.position
	m.npView.Paused = m.paused
	m.npView.Volume = m.cfg.Volume
	m.npView.Shuffle = m.cfg.Shuffle
	m.npView.Repeat = m.cfg.Repeat
	m.npView.QueuePos = m.queue.Pos()
	m.npView.QueueLen = m.queue.Len()
	m.npView.Tick = m.tickCount
	if m.currentTrack != nil {
		m.npView.IsFav = m.pm.IsFavourite(m.currentTrack.ID)
	}
}

func (m *Model) updateViewSizes() {
	mainH := maxInt(m.height-3, 1)
	m.libView.Width = m.width
	m.libView.Height = mainH
	m.plView.Width = m.width
	m.plView.Height = mainH
	m.favView.Width = m.width
	m.favView.Height = mainH
	m.npView.Width = m.width
	m.npView.Height = mainH
}

// View renders the entire TUI.
func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var rendered string
	switch {
	case m.showSearch:
		rendered = m.renderSearch()
	case m.showHelp:
		rendered = m.renderHelp()
	case m.showTimeJumpModal:
		rendered = m.renderTimeJumpModal()
	case m.showPlaylistPicker:
		rendered = m.renderPlaylistPicker()
	default:
		tabs := m.renderTabs()
		content := m.renderContent()
		statusBar := m.renderStatusBar()
		helpBar := m.renderHelpBar()

		rendered = lipgloss.JoinVertical(lipgloss.Left,
			tabs,
			content,
			statusBar,
			helpBar,
		)
	}

	// Strictly guarantee that the total output never exceeds the terminal height
	lines := strings.Split(rendered, "\n")
	if len(lines) > m.height {
		lines = lines[:m.height]
	}
	for len(lines) < m.height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderTabs() string {
	var tabs []string
	var labels []string
	if m.width >= 70 {
		labels = []string{"[1] Library", "[2] Playlists", "[3] Favourites", "[4] Now Playing"}
	} else if m.width >= 50 {
		labels = []string{"[1]Lib", "[2]Lists", "[3]Favs", "[4]Now"}
	} else {
		labels = []string{"1", "2", "3", "4"}
	}

	for i, name := range labels {
		if i == m.activeTab {
			tabs = append(tabs, styles.TabActive.Render(name))
		} else {
			tabs = append(tabs, styles.TabInactive.Render(name))
		}
	}

	title := ""
	if m.width >= 70 {
		title = styles.TabLogo.Render("MUSE // AUDIO") + " "
	} else if m.width >= 45 {
		title = styles.TabLogo.Render("MUSE") + " "
	}

	bar := title + strings.Join(tabs, " ")
	return styles.TabBar.Width(m.width).MaxHeight(1).Render(bar)
}

func (m *Model) renderContent() string {
	h := maxInt(m.height-3, 1)

	var content string
	switch m.activeTab {
	case TabLibrary:
		content = m.libView.View()
	case TabPlaylists:
		content = m.plView.View()
	case TabFavs:
		content = m.favView.View()
	case TabNowPlaying:
		content = m.npView.View()
	}

	lines := strings.Split(content, "\n")
	for len(lines) < h {
		lines = append(lines, "")
	}
	if len(lines) > h {
		lines = lines[:h]
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderStatusBar() string {
	isPlaying := m.currentTrack != nil && !m.paused
	miniVU := views.MiniVU(m.tickCount, isPlaying)

	var left string
	if m.currentTrack != nil {
		playStatus := styles.Playing.Render("PLAY")
		if m.paused {
			playStatus = styles.Amber.Render("PAUSE")
		}

		fav := ""
		if m.pm.IsFavourite(m.currentTrack.ID) {
			fav = styles.FavHeart.Render("♥ ")
		}

		titleMax := maxInt(m.width*30/100, 10)
		artistMax := maxInt(m.width*20/100, 8)
		title := styles.TrackName.Render(views.Trunc(m.currentTrack.DisplayTitle(), titleMax))
		artist := styles.ArtistName.Render(views.Trunc(m.currentTrack.Artist, artistMax))

		total := m.currentTrack.Duration
		var ratio float64
		if total > 0 {
			ratio = float64(m.position) / float64(total)
		}
		barW := minInt(maxInt(m.width*20/100, 10), 24)
		prog := views.ProgressBar(ratio, barW, m.position, total)

		if m.width >= 75 {
			left = fmt.Sprintf(" %s [%s] %s%s · %s  %s", miniVU, playStatus, fav, title, artist, prog)
		} else if m.width >= 50 {
			left = fmt.Sprintf(" %s [%s] %s%s  %s", miniVU, playStatus, fav, title, prog)
		} else {
			left = fmt.Sprintf(" %s %s%s", miniVU, fav, title)
		}
	} else {
		left = fmt.Sprintf(" %s %s", miniVU, styles.Muted.Render("STOPPED"))
	}

	if m.status != "" && m.width >= 60 {
		statusMax := m.width - views.VisibleLen(left) - 6
		if statusMax > 6 {
			left += "  " + styles.Accent.Render("│ "+views.Trunc(m.status, statusMax))
		}
	}

	return styles.StatusBar.Width(m.width).MaxHeight(1).Render(left)
}

func (m *Model) renderHelpBar() string {
	var hints string
	if m.width >= 90 {
		hints = "[1-4/Tab]view  [s]huffle/play  [0-9/g]jump  [Space]play  [n/p]track  [←→]seek  [o]pen cover  [+/-]vol  [/]search  [?]help  [q]uit"
	} else if m.width >= 65 {
		hints = "[1-4]views  [s]huffle  [Space]play  [n/p]track  [←→]seek  [o]pen cover  [?]help  [q]uit"
	} else {
		hints = "[s]Shuffle  [Space]Play  [n/p]Track  [o]Cover  [?]Help  [q]Quit"
	}
	return styles.HelpBar.Width(m.width).MaxHeight(1).Render(hints)
}

func (m *Model) renderSearch() string {
	var sb strings.Builder
	sb.WriteString(styles.TabBar.Width(m.width).MaxHeight(1).Render(styles.Primary.Render(" SEARCH // LIBRARY")))
	sb.WriteString("\n")
	sb.WriteString(m.srchView.View(m.width))
	sb.WriteString("\n")
	sb.WriteString(styles.HelpBar.Width(m.width).MaxHeight(1).Render("[↑↓]navigate  [Enter]play  [Esc]close"))
	return sb.String()
}

func (m *Model) renderTimeJumpModal() string {
	var sb strings.Builder
	sb.WriteString(styles.Bold.Render("Jump to Time Position:"))
	sb.WriteString("\n\n")
	sb.WriteString(styles.Input.Render(m.timeJumpInput + "█"))
	sb.WriteString("\n\n")
	sb.WriteString(styles.Subtext.Render("Examples: 1:30 (minute:sec) · 90s (seconds) · 50% (percentage)\n\n"))
	sb.WriteString(styles.Muted.Render("[Enter] Jump  [Esc] Cancel"))
	modal := styles.Modal.Render(sb.String())
	return views.Center(modal, m.width)
}

func (m *Model) renderPlaylistPicker() string {
	var sb strings.Builder
	sb.WriteString(styles.Bold.Render(fmt.Sprintf("Add '%s' to Playlist:", views.Trunc(m.playlistPickTrack.DisplayTitle(), 30))))
	sb.WriteString("\n\n")

	for i, pl := range m.pm.Playlists {
		prefix := "  "
		if i == m.playlistPickCursor {
			prefix = styles.Accent.Render("▶ ")
		}
		line := fmt.Sprintf("%s%s (%d tracks)", prefix, pl.Name, len(pl.TrackIDs))
		if i == m.playlistPickCursor {
			sb.WriteString(styles.ListItemSelected.Render(line) + "\n")
		} else {
			sb.WriteString(styles.ListItem.Render(line) + "\n")
		}
	}

	sb.WriteString("\n" + styles.Muted.Render("[↑↓]select  [Enter]add  [Esc]cancel"))
	modal := styles.Modal.Render(sb.String())
	return views.Center(modal, m.width)
}

func (m *Model) renderHelp() string {
	var sb strings.Builder
	sb.WriteString(styles.Bold.Render("MUSE // KEYBOARD SHORTCUTS") + "\n\n")

	if m.height >= 30 {
		type shortcut struct {
			key  string
			desc string
		}

		type section struct {
			title string
			items []shortcut
		}

		sections := []section{
			{
				title: "NAVIGATION",
				items: []shortcut{
					{"1, 2, 3, 4", "Switch to Library, Playlists, Favs, Now Playing"},
					{"Tab / Shift+Tab", "Cycle focus between sections & panes"},
					{"↑ / ↓ (j / k)", "Navigate song & playlist lists"},
					{"← / → (h / l)", "Switch playlist panes (left/right)"},
				},
			},
			{
				title: "PLAYBACK & CONTROLS",
				items: []shortcut{
					{"s", "Shuffle & play random (moves to Now Playing)"},
					{"S / z", "Force pick new random track from library"},
					{"Space", "Play / Pause"},
					{"Enter", "Play selected track or playlist"},
					{"n / p", "Next / Previous track in queue"},
					{"→ / ←", "Seek ±5s (Shift+→ / Shift+← for ±30s)"},
					{"0 - 9", "Jump to 0% - 90% position in track"},
					{"g or :", "Jump to exact time (e.g. 1:30, 90s, 50%)"},
					{"o", "Open full album cover in photo viewer"},
					{"+ / -", "Volume up / down (5% steps)"},
					{"m", "Mute / Unmute"},
					{"r", "Cycle Repeat (Off → Track → Queue)"},
				},
			},
			{
				title: "PLAYLISTS & SEARCH",
				items: []shortcut{
					{"f", "Toggle favourite heart (♥)"},
					{"a", "Add selected track to playlist"},
					{"c", "Create new playlist"},
					{"d / x", "Delete playlist or remove track"},
					{"/", "Live fuzzy search filter"},
					{"?", "Close this help cheatsheet"},
					{"q / Ctrl+C", "Quit and save player state"},
				},
			},
		}

		for _, sec := range sections {
			sb.WriteString(styles.Primary.Render("  "+sec.title) + "\n")
			for _, item := range sec.items {
				keyPill := styles.KeyHint.Render(fmt.Sprintf("%-18s", item.key))
				desc := styles.Subtext.Render(item.desc)
				sb.WriteString(fmt.Sprintf("    %s %s\n", keyPill, desc))
			}
			sb.WriteString("\n")
		}
	} else {
		// Compact cheatsheet for small terminal heights (like 82x25 or 58x27)
		shortcuts := [][2]string{
			{"1 - 4", "Switch views (Library, Playlists, Favs, Now)"},
			{"Tab", "Cycle sections / playlist panes"},
			{"s / S", "Shuffle & play random (moves to Now Playing)"},
			{"Space / Enter", "Play / Pause / Play selection"},
			{"n / p", "Next / Previous track in queue"},
			{"→ / ←", "Seek ±5s (Shift+→ / ← for ±30s)"},
			{"0 - 9 / g", "Jump to 0%-90% or exact time (1:30, 50%)"},
			{"o", "Open full album art in photo viewer"},
			{"+ / - / m", "Volume up / down / Mute toggle"},
			{"f / a / c", "Favourite (♥) / Add to Playlist / Create"},
			{"/ / ?", "Live search / Close this help guide"},
			{"q", "Quit and save player state"},
		}

		for _, item := range shortcuts {
			keyPill := styles.KeyHint.Render(fmt.Sprintf("%-14s", item[0]))
			desc := styles.Subtext.Render(item[1])
			sb.WriteString(fmt.Sprintf("  %s %s\n", keyPill, desc))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(styles.Muted.Render("  [?] or [Esc] to close guide"))
	modal := styles.Modal.Render(sb.String())
	return views.Center(modal, m.width)
}

func openCoverArt(data []byte, title string) error {
	tmpDir := os.TempDir()
	ext := ".jpg"
	if len(data) >= 8 && bytes.Equal(data[:8], []byte("\x89PNG\r\n\x1a\n")) {
		ext = ".png"
	}
	clean := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, title)
	if clean == "" {
		clean = "cover"
	}
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("muse_%s%s", clean, ext))
	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		return err
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", tmpFile)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", tmpFile)
	default:
		cmd = exec.Command("xdg-open", tmpFile)
	}
	return cmd.Start()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
