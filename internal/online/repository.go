package online

import (
	"context"
)

// OnlineRepository provides an abstraction for online music discovery, streaming, and caching.
type OnlineRepository interface {
	Search(ctx context.Context, query string, limit int) ([]OnlineTrack, error)
	Suggestions(query string) ([]string, error)
	RelatedTracks(ctx context.Context, track OnlineTrack, limit int) ([]OnlineTrack, error)
	RecommendForLibrary(ctx context.Context, profile TasteProfile, limit int) ([]OnlineTrack, error)
	ResolveStream(ctx context.Context, track OnlineTrack, onStatus func(string)) (StreamResult, error)
	FetchArtwork(artworkURL string) ([]byte, error)
	GetCachedTrack(track OnlineTrack) (string, []byte, bool)
	Prebuffer(ctx context.Context, track OnlineTrack)
	Download(ctx context.Context, track OnlineTrack, targetDir string) (string, error)
}

// OnlineRepo is the default implementation of OnlineRepository coordinating caching, HTTP client, and streaming.
type OnlineRepo struct {
	cache  *MemoryCache
	scorer RecommendationScorer
}

// NewRepository creates a new OnlineRepo with the default cache and scorer.
func NewRepository() *OnlineRepo {
	return &OnlineRepo{
		cache:  DefaultCache(),
		scorer: &DefaultRecommendationScorer{},
	}
}

// Search searches for tracks online (YouTube + iTunes fallback) with caching.
func (r *OnlineRepo) Search(ctx context.Context, query string, limit int) ([]OnlineTrack, error) {
	return SearchOnline(ctx, query, limit)
}

// Suggestions retrieves instant search suggestions with caching.
func (r *OnlineRepo) Suggestions(query string) ([]string, error) {
	return FetchSuggestions(query)
}

// RelatedTracks queries radio tracks related to a song for continuous playback.
func (r *OnlineRepo) RelatedTracks(ctx context.Context, track OnlineTrack, limit int) ([]OnlineTrack, error) {
	return FetchRelatedTracks(ctx, track, limit)
}

// RecommendForLibrary derives recommendations based on the user's taste profile.
func (r *OnlineRepo) RecommendForLibrary(ctx context.Context, profile TasteProfile, limit int) ([]OnlineTrack, error) {
	return RecommendForLibrary(ctx, profile, limit, r.scorer)
}

// ResolveStream resolves a stream with progressive playback as soon as initial buffer is ready.
func (r *OnlineRepo) ResolveStream(ctx context.Context, track OnlineTrack, onStatus func(string)) (StreamResult, error) {
	return ResolveStream(ctx, track, onStatus)
}

// FetchArtwork downloads and caches cover art.
func (r *OnlineRepo) FetchArtwork(artworkURL string) ([]byte, error) {
	return FetchArtwork(artworkURL)
}

// GetCachedTrack checks if a track is cached locally.
func (r *OnlineRepo) GetCachedTrack(track OnlineTrack) (string, []byte, bool) {
	return GetCachedTrack(track)
}

// Prebuffer starts background resolution and caching for a track.
func (r *OnlineRepo) Prebuffer(ctx context.Context, track OnlineTrack) {
	PrebufferTrack(ctx, track)
}

// Download downloads an online track permanently into the destination folder.
func (r *OnlineRepo) Download(ctx context.Context, track OnlineTrack, targetDir string) (string, error) {
	return DownloadToLibrary(ctx, track, targetDir)
}
