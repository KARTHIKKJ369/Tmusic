// Package audio provides the playback engine, format decoder and metadata reader.
package audio

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/dhowden/tag"
)

// Track represents a single audio file with its metadata.
type Track struct {
	ID       string        // sha-style unique id (path-derived)
	Path     string        // absolute file path
	Title    string
	Artist   string
	Album    string
	Genre    string
	Year     int
	Duration time.Duration
}

// DisplayTitle returns title or filename if title is empty.
func (t Track) DisplayTitle() string {
	if t.Title != "" {
		return t.Title
	}
	return strings.TrimSuffix(filepath.Base(t.Path), filepath.Ext(t.Path))
}

// ReadMetadata reads tags from path. Duration is filled separately after decoding.
func ReadMetadata(path string) Track {
	t := Track{
		ID:   pathID(path),
		Path: path,
	}

	f, err := openFile(path)
	if err != nil {
		return t
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		return t
	}

	t.Title = m.Title()
	t.Artist = m.Artist()
	t.Album = m.Album()
	t.Genre = m.Genre()
	t.Year = m.Year()
	if dur, err := Duration(path); err == nil {
		t.Duration = dur
	}
	return t
}

// ExtractPicture returns raw picture bytes and MIME type from path if present.
func ExtractPicture(path string) ([]byte, string) {
	f, err := openFile(path)
	if err != nil {
		return nil, ""
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		return nil, ""
	}

	p := m.Picture()
	if p != nil && len(p.Data) > 0 {
		return p.Data, p.MIMEType
	}
	return nil, ""
}

// SupportedExt returns true for audio extensions we handle.
func SupportedExt(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3", ".flac", ".wav", ".ogg":
		return true
	}
	return false
}

// pathID converts a file path to a stable short ID.
func pathID(path string) string {
	// Use cleaned absolute path as id — simple and collision-free for local files.
	return filepath.Clean(path)
}
