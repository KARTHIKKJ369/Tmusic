package online

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// TasteProfile encapsulates user music preferences derived from their listening habits/library.
type TasteProfile struct {
	TopArtists  []string
	TopGenres   []string
	TotalTracks int
}

// RecommendationScorer computes a relevance score for a candidate online track given user taste.
type RecommendationScorer interface {
	Score(candidate OnlineTrack, profile TasteProfile) float64
}

// DefaultRecommendationScorer scores tracks based on artist matching, genre alignment, and diversity.
type DefaultRecommendationScorer struct{}

// Score calculates a numerical relevance score for candidate relative to the profile.
func (s *DefaultRecommendationScorer) Score(candidate OnlineTrack, profile TasteProfile) float64 {
	score := 1.0

	candArtist := strings.ToLower(strings.TrimSpace(candidate.Artist))
	for i, artist := range profile.TopArtists {
		a := strings.ToLower(strings.TrimSpace(artist))
		if a != "" && (candArtist == a || strings.Contains(candArtist, a) || strings.Contains(a, candArtist)) {
			boost := 10.0 - float64(i)*1.5
			if boost < 2.0 {
				boost = 2.0
			}
			score += boost
			break
		}
	}

	candGenre := strings.ToLower(strings.TrimSpace(candidate.Genre))
	if candGenre != "" {
		for i, genre := range profile.TopGenres {
			g := strings.ToLower(strings.TrimSpace(genre))
			if g != "" && (candGenre == g || strings.Contains(candGenre, g)) {
				score += 5.0 - float64(i)
				break
			}
		}
	}

	return score
}

// RecommendForLibrary finds personalized music recommendations based on the user's library taste profile.
func RecommendForLibrary(ctx context.Context, profile TasteProfile, limit int, scorer RecommendationScorer) ([]OnlineTrack, error) {
	if limit <= 0 {
		limit = 15
	}
	if scorer == nil {
		scorer = &DefaultRecommendationScorer{}
	}

	cacheKey := fmt.Sprintf("rec:%d:%s:%s", limit, strings.Join(profile.TopArtists, ","), strings.Join(profile.TopGenres, ","))
	if val, ok := DefaultCache().Get(cacheKey); ok {
		if cached, ok := val.([]OnlineTrack); ok {
			return cached, nil
		}
	}

	// If library has no top artists, recommend general popular music
	if len(profile.TopArtists) == 0 {
		fallback, err := SearchOnline(ctx, "Top Hits Music", limit)
		if err == nil && len(fallback) > 0 {
			DefaultCache().Set(cacheKey, fallback, 1*time.Hour)
		}
		return fallback, err
	}

	// Concurrently query up to top 3 artists
	artistsToQuery := profile.TopArtists
	if len(artistsToQuery) > 3 {
		artistsToQuery = artistsToQuery[:3]
	}

	var (
		mu         sync.Mutex
		candidates []OnlineTrack
		seenIDs    = make(map[string]bool)
		seenTitles = make(map[string]bool)
		wg         sync.WaitGroup
	)

	addTrack := func(t OnlineTrack) {
		mu.Lock()
		defer mu.Unlock()
		normTitle := strings.ToLower(strings.TrimSpace(t.Title))
		if t.ID == "" || seenIDs[t.ID] || seenTitles[normTitle] {
			return
		}
		seenIDs[t.ID] = true
		seenTitles[normTitle] = true
		candidates = append(candidates, t)
	}

	for _, artist := range artistsToQuery {
		wg.Add(1)
		go func(a string) {
			defer wg.Done()
			query := fmt.Sprintf("%s top songs playlist", a)
			tracks, err := SearchYouTube(ctx, query, 8)
			if err == nil {
				for _, t := range tracks {
					addTrack(t)
				}
			}
		}(artist)
	}

	// Also query genre if available
	if len(profile.TopGenres) > 0 && profile.TopGenres[0] != "" {
		wg.Add(1)
		go func(g string) {
			defer wg.Done()
			query := fmt.Sprintf("Best of %s music playlist", g)
			tracks, err := SearchYouTube(ctx, query, 6)
			if err == nil {
				for _, t := range tracks {
					addTrack(t)
				}
			}
		}(profile.TopGenres[0])
	}

	wg.Wait()

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no recommendations found")
	}

	type scoredTrack struct {
		track OnlineTrack
		score float64
	}
	scored := make([]scoredTrack, len(candidates))
	for i, t := range candidates {
		scored[i] = scoredTrack{
			track: t,
			score: scorer.Score(t, profile),
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	if len(scored) > limit {
		scored = scored[:limit]
	}

	results := make([]OnlineTrack, len(scored))
	for i, st := range scored {
		results[i] = st.track
	}

	DefaultCache().Set(cacheKey, results, 1*time.Hour)
	return results, nil
}
