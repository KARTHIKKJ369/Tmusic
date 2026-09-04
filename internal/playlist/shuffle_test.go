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

	// Verify consecutive same artist count is minimized
	consecutiveSame := 0
	for i := 0; i < len(spread)-1; i++ {
		if tracks[spread[i]].Artist == tracks[spread[i+1]].Artist {
			consecutiveSame++
		}
	}

	if consecutiveSame > 1 {
		t.Fatalf("expected <= 1 consecutive same artist tracks, got %d", consecutiveSame)
	}
}

func TestShuffleRandomDistribution(t *testing.T) {
	tracks := []audio.Track{
		{ID: "1", Title: "Song 1", Artist: "Artist A"},
		{ID: "2", Title: "Song 2", Artist: "Artist B"},
		{ID: "3", Title: "Song 3", Artist: "Artist C"},
		{ID: "4", Title: "Song 4", Artist: "Artist D"},
		{ID: "5", Title: "Song 5", Artist: "Artist E"},
		{ID: "6", Title: "Song 6", Artist: "Artist F"},
		{ID: "7", Title: "Song 7", Artist: "Artist G"},
		{ID: "8", Title: "Song 8", Artist: "Artist H"},
	}

	firstTrackCounts := make(map[string]int)
	trials := 100

	for i := 0; i < trials; i++ {
		q := NewQueue(tracks)
		q.Shuffle() // No activeTrackID -> must pick a random starting track!
		curr, ok := q.Current()
		if !ok {
			t.Fatal("expected current track")
		}
		firstTrackCounts[curr.ID]++
	}

	// With 8 tracks and 100 trials, we should see almost all tracks selected as starting track
	if len(firstTrackCounts) < 6 {
		t.Fatalf("shuffle is not random, only %d distinct starting tracks seen in %d trials", len(firstTrackCounts), trials)
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

	q.Shuffle("3") // active track "3" should be preserved
	curr, ok := q.Current()
	if !ok || curr.ID != "3" {
		t.Fatalf("expected current track ID '3' after shuffle, got %v", curr.ID)
	}

	q.Unshuffle("3")
	curr, ok = q.Current()
	if !ok || curr.ID != "3" {
		t.Fatalf("expected current track ID '3' after unshuffle, got %v", curr.ID)
	}
}
