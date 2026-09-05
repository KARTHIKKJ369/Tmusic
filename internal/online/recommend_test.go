package online

import (
	"context"
	"testing"
)

func TestRecommendationScorer(t *testing.T) {
	scorer := &DefaultRecommendationScorer{}
	profile := TasteProfile{
		TopArtists:  []string{"Daft Punk", "Queen"},
		TopGenres:   []string{"Electronic", "Rock"},
		TotalTracks: 20,
	}

	matchArtist := OnlineTrack{Title: "Get Lucky", Artist: "Daft Punk", Genre: "Electronic"}
	matchGenreOnly := OnlineTrack{Title: "Strobe", Artist: "Deadmau5", Genre: "Electronic"}
	unrelated := OnlineTrack{Title: "Symphony No. 5", Artist: "Beethoven", Genre: "Classical"}

	score1 := scorer.Score(matchArtist, profile)
	score2 := scorer.Score(matchGenreOnly, profile)
	score3 := scorer.Score(unrelated, profile)

	if score1 <= score2 {
		t.Fatalf("expected artist match score (%f) > genre only score (%f)", score1, score2)
	}
	if score2 <= score3 {
		t.Fatalf("expected genre match score (%f) > unrelated score (%f)", score2, score3)
	}
}

func TestRecommendForLibraryEmptyProfile(t *testing.T) {
	ctx := context.Background()
	profile := TasteProfile{}
	// Empty profile should fallback gracefully
	tracks, err := RecommendForLibrary(ctx, profile, 2, nil)
	if err != nil {
		t.Skipf("Network unavailable for YouTube test: %v", err)
	}
	if len(tracks) == 0 {
		t.Fatal("expected at least 1 track in fallback recommendations")
	}
}
