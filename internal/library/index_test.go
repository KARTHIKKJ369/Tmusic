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

func TestTasteProfile(t *testing.T) {
	idx := NewIndex()
	tracks := []audio.Track{
		{ID: "t1", Title: "Song 1", Artist: "Beatles", Genre: "Rock"},
		{ID: "t2", Title: "Song 2", Artist: "Beatles", Genre: "Rock"},
		{ID: "t3", Title: "Song 3", Artist: "Queen", Genre: "Rock"},
		{ID: "t4", Title: "Song 4", Artist: "Miles Davis", Genre: "Jazz"},
	}
	idx.set(tracks)

	profile := idx.TasteProfile()
	if profile.TotalTracks != 4 {
		t.Fatalf("expected 4 total tracks, got %d", profile.TotalTracks)
	}
	if len(profile.TopArtists) == 0 || profile.TopArtists[0] != "Beatles" {
		t.Fatalf("expected top artist Beatles, got %v", profile.TopArtists)
	}
	if len(profile.TopGenres) == 0 || profile.TopGenres[0] != "Rock" {
		t.Fatalf("expected top genre Rock, got %v", profile.TopGenres)
	}
}
