package library

import "github.com/KARTHIKKJ369/Tmusic/internal/audio"

// TasteProfile encapsulates user listening preferences derived from their local library.
type TasteProfile struct {
	TopArtists  []string
	TopGenres   []string
	TotalTracks int
}

// LocalRepository provides an abstraction over local music index and storage.
type LocalRepository interface {
	All() []audio.Track
	ByID(id string) (audio.Track, bool)
	Len() int
	AddTrack(path string) error
	Load() error
	Save() error
	Scan(dir string, progress func(done, total int)) error
	TasteProfile() TasteProfile
}
