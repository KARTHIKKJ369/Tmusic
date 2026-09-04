package library

import (
	"testing"

	"github.com/KARTHIKKJ369/Tmusic/internal/audio"
)

func TestIndexOperations(t *testing.T) {
	idx := NewIndex()

	tracks := []audio.Track{
		{ID: "track-1", Title: "Song 1", Artist: "Artist 1"},
		{ID: "track-2", Title: "Song 2", Artist: "Artist 2"},
	}

	idx.set(tracks)

	if idx.Len() != 2 {
		t.Fatalf("expected len 2, got %d", idx.Len())
	}

	tr, ok := idx.ByID("track-2")
	if !ok || tr.Title != "Song 2" {
		t.Fatalf("expected Song 2, got %v", tr)
	}

	all := idx.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 tracks, got %d", len(all))
	}
}
