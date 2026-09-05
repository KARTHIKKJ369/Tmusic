package tui

import (
	"sync"
	"testing"
	"time"

	"github.com/KARTHIKKJ369/Tmusic/internal/audio"
	"github.com/KARTHIKKJ369/Tmusic/internal/config"
	"github.com/KARTHIKKJ369/Tmusic/internal/library"
	"github.com/KARTHIKKJ369/Tmusic/internal/online"
	"github.com/KARTHIKKJ369/Tmusic/internal/playlist"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	testPlayer     *audio.Player
	testPlayerErr  error
	testPlayerOnce sync.Once
)

func getSharedTestPlayer(t *testing.T) *audio.Player {
	testPlayerOnce.Do(func() {
		testPlayer, testPlayerErr = audio.NewPlayer(0.8)
	})
	if testPlayerErr != nil {
		t.Skipf("audio device not available in test environment: %v", testPlayerErr)
	}
	return testPlayer
}

func TestPlaySelectionSwitchesToNowPlaying(t *testing.T) {
	p := getSharedTestPlayer(t)
	cfg := config.Config{}
	idx := library.NewIndex()
	pm := playlist.NewManager()

	onlineRepo := online.NewRepository()
	m := New(cfg, idx, pm, p, onlineRepo)

	// Switch to online tab
	m.activeTab = TabOnline
	m.onlineView.InputMode = true

	// Simulate search results arriving
	tracks := []online.OnlineTrack{
		{
			ID:       "vid12345",
			Title:    "Puthu Mazha",
			Artist:   "Artist Name",
			Duration: 3 * time.Minute,
		},
	}
	updated, _ := m.Update(onlineSearchMsg{query: "puthu mazha", tracks: tracks})
	m = updated.(*Model)

	if m.onlineView.InputMode {
		t.Errorf("expected onlineView.InputMode to be false after search results arrive")
	}
	if len(m.onlineView.Tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(m.onlineView.Tracks))
	}

	// Pressing Enter to play selected online track
	_ = m.playSelected()

	// Should immediately switch activeTab to TabNowPlaying
	if m.activeTab != TabNowPlaying {
		t.Errorf("expected activeTab to be TabNowPlaying (5) immediately upon hitting play, got %d", m.activeTab)
	}
	if m.currentTrack == nil {
		t.Fatalf("expected currentTrack to be set immediately upon hitting play")
	}
	if m.currentTrack.Title != "Puthu Mazha" {
		t.Errorf("expected currentTrack title 'Puthu Mazha', got %q", m.currentTrack.Title)
	}

	// When onlineStreamMsg arrives with resolved stream
	updated, _ = m.Update(onlineStreamMsg{
		track: tracks[0],
		result: online.StreamResult{
			Track:    tracks[0],
			FilePath: "/fake/cached.mp3",
		},
	})
	m = updated.(*Model)

	if m.activeTab != TabNowPlaying {
		t.Errorf("expected activeTab to be TabNowPlaying after onlineStreamMsg, got %d", m.activeTab)
	}
}

func TestOnlineInputKeyTransitions(t *testing.T) {
	p := getSharedTestPlayer(t)
	cfg := config.Config{}
	idx := library.NewIndex()
	pm := playlist.NewManager()

	onlineRepo := online.NewRepository()
	m := New(cfg, idx, pm, p, onlineRepo)

	m.activeTab = TabOnline
	m.onlineView.InputMode = true
	m.onlineView.Tracks = []online.OnlineTrack{
		{ID: "t1", Title: "Sample", Artist: "Test"},
	}

	// Pressing KeyDown while in input mode with tracks available should exit input mode
	updated, _ := m.handleOnlineInput(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(*Model)
	if m.onlineView.InputMode {
		t.Errorf("expected InputMode=false after KeyDown with tracks available")
	}

	// Re-enter input mode
	m.onlineView.InputMode = true
	m.onlineView.Query = "new search"

	// Pressing KeyEnter should exit input mode and submit search
	updated, _ = m.handleOnlineInput(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(*Model)
	if m.onlineView.InputMode {
		t.Errorf("expected InputMode=false after KeyEnter")
	}
}
