package playlist

import (
	"testing"

	"github.com/KARTHIKKJ369/Tmusic/internal/audio"
)

func TestQueueBasic(t *testing.T) {
	tracks := []audio.Track{
		{ID: "1", Title: "Song 1", Artist: "Artist A"},
		{ID: "2", Title: "Song 2", Artist: "Artist B"},
		{ID: "3", Title: "Song 3", Artist: "Artist C"},
	}

	q := NewQueue(tracks)
	if q.Len() != 3 {
		t.Fatalf("expected len 3, got %d", q.Len())
	}

	curr, ok := q.Current()
	if !ok || curr.ID != "1" {
		t.Fatalf("expected current ID '1', got %v", curr.ID)
	}

	next, ok := q.Next()
	if !ok || next.ID != "2" {
		t.Fatalf("expected next ID '2', got %v", next.ID)
	}

	prev, ok := q.Prev()
	if !ok || prev.ID != "1" {
		t.Fatalf("expected prev ID '1', got %v", prev.ID)
	}

	if !q.JumpTo("3") {
		t.Fatal("expected JumpTo to succeed")
	}
	curr, _ = q.Current()
	if curr.ID != "3" {
		t.Fatalf("expected current ID '3', got %v", curr.ID)
	}
}

func TestSpreadByArtist(t *testing.T) {
	tracks := []audio.Track{
		{ID: "1", Artist: "Queen"},
		{ID: "2", Artist: "Queen"},
		{ID: "3", Artist: "Eagles"},
		{ID: "4", Artist: "Queen"},
		{ID: "5", Artist: "Eagles"},
		{ID: "6", Artist: "Pink Floyd"},
	}

	perm := []int{0, 1, 3, 2, 4, 5} // 3 Queens in a row initially
	spread := spreadByArtist(perm, tracks)

	if len(spread) != len(tracks) {
		t.Fatalf("expected spread len %d, got %d", len(tracks), len(spread))
	}

	// Verify no consecutive tracks share the same artist if avoidable
	consecutiveSame := 0
	for i := 0; i < len(spread)-1; i++ {
		if tracks[spread[i]].Artist == tracks[spread[i+1]].Artist {
			consecutiveSame++
		}
	}

	if consecutiveSame > 0 {
		t.Fatalf("expected 0 consecutive same artist tracks, got %d", consecutiveSame)
	}
}

func TestShuffleMaintainsCurrentTrack(t *testing.T) {
	tracks := []audio.Track{
		{ID: "1", Artist: "Queen"},
		{ID: "2", Artist: "Eagles"},
		{ID: "3", Artist: "Pink Floyd"},
		{ID: "4", Artist: "Led Zeppelin"},
	}

	q := NewQueue(tracks)
	q.JumpTo("3")

	q.Shuffle()
	curr, ok := q.Current()
	if !ok || curr.ID != "3" {
		t.Fatalf("expected current track ID '3' after shuffle, got %v", curr.ID)
	}

	q.Unshuffle()
	curr, ok = q.Current()
	if !ok || curr.ID != "3" {
		t.Fatalf("expected current track ID '3' after unshuffle, got %v", curr.ID)
	}
}
