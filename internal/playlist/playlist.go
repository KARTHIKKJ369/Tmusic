// Package playlist manages playlists and favourites persistence.
package playlist

import (
	"time"

	"github.com/KARTHIKKJ369/Tmusic/internal/config"
)

const (
	playlistsFile  = "playlists.json"
	favouritesFile = "favourites.json"
)

// Playlist is an ordered list of track IDs.
type Playlist struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	TrackIDs  []string  `json:"track_ids"`
}

// Manager manages all playlists and favourites.
type Manager struct {
	Playlists  []Playlist
	Favourites map[string]bool // track ID → true
}

// NewManager returns an empty Manager.
func NewManager() *Manager {
	return &Manager{Favourites: make(map[string]bool)}
}

// Load reads playlists and favourites from disk.
func (m *Manager) Load() error {
	if err := config.LoadJSON(playlistsFile, &m.Playlists); err != nil {
		return err
	}
	if m.Playlists == nil {
		m.Playlists = []Playlist{}
	}
	var favList []string
	if err := config.LoadJSON(favouritesFile, &favList); err != nil {
		return err
	}
	m.Favourites = make(map[string]bool, len(favList))
	for _, id := range favList {
		m.Favourites[id] = true
	}
	return nil
}

// Save writes playlists and favourites to disk.
func (m *Manager) Save() error {
	if err := config.SaveJSON(playlistsFile, m.Playlists); err != nil {
		return err
	}
	favList := make([]string, 0, len(m.Favourites))
	for id := range m.Favourites {
		favList = append(favList, id)
	}
	return config.SaveJSON(favouritesFile, favList)
}

// ToggleFavourite flips the favourite state of a track.
func (m *Manager) ToggleFavourite(id string) bool {
	if m.Favourites[id] {
		delete(m.Favourites, id)
		return false
	}
	m.Favourites[id] = true
	return true
}

// IsFavourite returns whether a track is a favourite.
func (m *Manager) IsFavourite(id string) bool {
	return m.Favourites[id]
}

// Create adds a new empty playlist.
func (m *Manager) Create(name string) {
	m.Playlists = append(m.Playlists, Playlist{
		Name:      name,
		CreatedAt: time.Now(),
	})
}

// AddTrack appends a track to a playlist by index.
func (m *Manager) AddTrack(playlistIdx int, trackID string) {
	if playlistIdx < 0 || playlistIdx >= len(m.Playlists) {
		return
	}
	pl := &m.Playlists[playlistIdx]
	for _, id := range pl.TrackIDs {
		if id == trackID {
			return // already in playlist
		}
	}
	pl.TrackIDs = append(pl.TrackIDs, trackID)
}

// RemoveTrack removes a track from a playlist by index.
func (m *Manager) RemoveTrack(playlistIdx int, trackID string) {
	if playlistIdx < 0 || playlistIdx >= len(m.Playlists) {
		return
	}
	pl := &m.Playlists[playlistIdx]
	for i, id := range pl.TrackIDs {
		if id == trackID {
			pl.TrackIDs = append(pl.TrackIDs[:i], pl.TrackIDs[i+1:]...)
			return
		}
	}
}

// Delete removes a playlist by index.
func (m *Manager) Delete(idx int) {
	if idx < 0 || idx >= len(m.Playlists) {
		return
	}
	m.Playlists = append(m.Playlists[:idx], m.Playlists[idx+1:]...)
}
