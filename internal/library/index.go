// Package library scans a music directory and builds a track index.
package library

import (
	"io/fs"
	"path/filepath"
	"sync"

	"github.com/KARTHIKKJ369/Tmusic/internal/audio"
	"github.com/KARTHIKKJ369/Tmusic/internal/config"
)

const indexFile = "index.json"

// Index holds all known tracks.
type Index struct {
	mu     sync.RWMutex
	tracks []audio.Track
	byID   map[string]*audio.Track
}

// NewIndex returns an empty index.
func NewIndex() *Index {
	return &Index{byID: make(map[string]*audio.Track)}
}

// All returns a copy of all tracks.
func (idx *Index) All() []audio.Track {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	out := make([]audio.Track, len(idx.tracks))
	copy(out, idx.tracks)
	return out
}

// ByID returns a track by its ID.
func (idx *Index) ByID(id string) (audio.Track, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	t, ok := idx.byID[id]
	if !ok {
		return audio.Track{}, false
	}
	return *t, true
}

// Len returns the number of indexed tracks.
func (idx *Index) Len() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.tracks)
}

func (idx *Index) set(tracks []audio.Track) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.tracks = tracks
	idx.byID = make(map[string]*audio.Track, len(tracks))
	for i := range tracks {
		idx.byID[tracks[i].ID] = &idx.tracks[i]
	}
}

// AddTrack reads metadata for a newly added audio file and appends it to the index.
func (idx *Index) AddTrack(path string) error {
	t := audio.ReadMetadata(path)
	idx.mu.Lock()
	defer idx.mu.Unlock()
	for i, existing := range idx.tracks {
		if existing.Path == t.Path {
			idx.tracks[i] = t
			idx.byID[t.ID] = &idx.tracks[i]
			return nil
		}
	}
	idx.tracks = append(idx.tracks, t)
	idx.byID[t.ID] = &idx.tracks[len(idx.tracks)-1]
	return nil
}

// Load attempts to restore the index from the cache file.
func (idx *Index) Load() error {
	var tracks []audio.Track
	if err := config.LoadJSON(indexFile, &tracks); err != nil {
		return err
	}
	if tracks != nil {
		idx.set(tracks)
	}
	return nil
}

// Save persists the index to the cache file.
func (idx *Index) Save() error {
	idx.mu.RLock()
	tracks := make([]audio.Track, len(idx.tracks))
	copy(tracks, idx.tracks)
	idx.mu.RUnlock()
	return config.SaveJSON(indexFile, tracks)
}

// Scan walks dir concurrently, reads metadata, and populates the index.
func (idx *Index) Scan(dir string, progress func(done, total int)) error {
	// Collect all audio paths first.
	var paths []string
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if audio.SupportedExt(path) {
			paths = append(paths, path)
		}
		return nil
	})

	total := len(paths)
	if total == 0 {
		return nil
	}

	tracks := make([]audio.Track, total)
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		done    int
		workers = 8
		jobs    = make(chan int, total)
	)

	for i := 0; i < total; i++ {
		jobs <- i
	}
	close(jobs)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				t := audio.ReadMetadata(paths[i])
				tracks[i] = t
				mu.Lock()
				done++
				if progress != nil {
					progress(done, total)
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	idx.set(tracks)
	return nil
}
