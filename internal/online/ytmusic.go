// Package online handles metadata discovery, suggestions, and streaming via YouTube and iTunes.
package online

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// OnlineTrack represents a unified music track from YouTube or iTunes.
type OnlineTrack struct {
	ID         string        `json:"id"`
	Title      string        `json:"title"`
	Artist     string        `json:"artist"`
	Album      string        `json:"album"`
	Duration   time.Duration `json:"duration"`
	ArtworkURL string        `json:"artwork_url"`
	Source     string        `json:"source"` // "youtube" or "itunes"
	Year       int           `json:"year"`
	Genre      string        `json:"genre"`
}

// FetchSuggestions retrieves instant search suggestions from YouTube in ~15ms with zero credentials.
func FetchSuggestions(query string) ([]string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	if sugs, ok := DefaultCache().GetSuggestions(query); ok {
		return sugs, nil
	}

	endpoint := fmt.Sprintf("https://suggestqueries.google.com/complete/search?client=firefox&ds=yt&q=%s", url.QueryEscape(query))
	resp, err := HTTPClient().Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data []any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if len(data) < 2 {
		return nil, nil
	}

	rawList, ok := data[1].([]any)
	if !ok {
		return nil, nil
	}

	var suggestions []string
	for _, item := range rawList {
		if s, ok := item.(string); ok && s != "" {
			suggestions = append(suggestions, s)
		}
	}
	if len(suggestions) > 8 {
		suggestions = suggestions[:8]
	}
	DefaultCache().SetSuggestions(query, suggestions)
	return suggestions, nil
}

// SearchYouTube queries YouTube using yt-dlp flat playlist extraction.
func SearchYouTube(ctx context.Context, query string, limit int) ([]OnlineTrack, error) {
	if limit <= 0 {
		limit = 15
	}

	cacheKey := fmt.Sprintf("yt:%d:%s", limit, query)
	if tracks, ok := DefaultCache().GetSearch(cacheKey); ok {
		return tracks, nil
	}

	// Use yt-dlp with --flat-playlist for sub-second search
	searchQuery := fmt.Sprintf("ytsearch%d:%s", limit, query)
	cmd := exec.CommandContext(ctx, "yt-dlp",
		"--flat-playlist",
		"--no-warnings",
		"--print", "%(id)s\t%(title)s\t%(uploader)s\t%(duration)s",
		searchQuery,
	)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("youtube search: %w", err)
	}

	tracks := parseYouTubeTracks(string(out))
	DefaultCache().SetSearch(cacheKey, tracks)
	return tracks, nil
}

func parseYouTubeTracks(out string) []OnlineTrack {
	lines := strings.Split(out, "\n")
	var tracks []OnlineTrack

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}

		id := parts[0]
		rawTitle := parts[1]
		uploader := "YouTube Music"
		if len(parts) >= 3 && parts[2] != "" && parts[2] != "NA" {
			uploader = parts[2]
		}

		var dur time.Duration
		if len(parts) >= 4 && parts[3] != "" && parts[3] != "NA" {
			if secs, err := strconv.ParseFloat(parts[3], 64); err == nil {
				dur = time.Duration(secs * float64(time.Second))
			}
		}

		title, artist := CleanYouTubeMetadata(rawTitle, uploader)
		artworkURL := fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", id)

		tracks = append(tracks, OnlineTrack{
			ID:         id,
			Title:      title,
			Artist:     artist,
			Album:      "YouTube Music",
			Duration:   dur,
			ArtworkURL: artworkURL,
			Source:     "youtube",
		})
	}

	return tracks
}

// CleanYouTubeMetadata strips noisy tags like "Video Song", "Official Audio", etc.
func CleanYouTubeMetadata(rawTitle, uploader string) (string, string) {
	title := rawTitle
	artist := uploader

	noise := []string{
		"Official Music Video", "Official Audio", "Official Video",
		"Lyric Video", "Full Video", "Video Song", "Audio Track",
		"HD Video", "4K Video", "Full Song", "Lyrical Video",
		"(Official Music Video)", "[Official Music Video]",
		"(Official Audio)", "[Official Audio]",
		"(Official Video)", "[Official Video]",
		"(Video Song)", "[Video Song]",
		"(Audio)", "[Audio]",
		"(Lyric Video)", "[Lyric Video]",
		"(Full Song)", "[Full Song]",
	}

	for _, n := range noise {
		title = strings.ReplaceAll(title, n, "")
		title = strings.ReplaceAll(title, strings.ToLower(n), "")
	}

	// If title contains " | ", take first part as title
	if idx := strings.Index(title, " | "); idx != -1 {
		title = strings.TrimSpace(title[:idx])
	} else if idx := strings.Index(title, "|"); idx != -1 {
		title = strings.TrimSpace(title[:idx])
	}

	// If title has " - ", split artist and title
	if parts := strings.Split(title, " - "); len(parts) == 2 {
		artist = strings.TrimSpace(parts[0])
		title = strings.TrimSpace(parts[1])
	}

	artist = strings.TrimSuffix(artist, " - Topic")
	artist = strings.TrimSuffix(artist, "VEVO")
	artist = strings.TrimSpace(artist)

	return strings.TrimSpace(title), artist
}

// FetchRelatedTracks searches for songs related to the current track for continuous playback.
// It queries the official YouTube Automix Radio playlist (list=RD<video_id>) for genuine contextual
// recommendations (matching genre, artist, era, mood), falling back to artist radio search.
func FetchRelatedTracks(ctx context.Context, track OnlineTrack, limit int) ([]OnlineTrack, error) {
	if limit <= 0 {
		limit = 10
	}

	radioKey := fmt.Sprintf("%s:%d", track.ID, limit)
	if tracks, ok := DefaultCache().GetRadio(radioKey); ok {
		return tracks, nil
	}

	var rawTracks []OnlineTrack

	// 1. Primary & Best Approach: Official YouTube Automix Radio (list=RD<id>)
	if track.Source == "youtube" && len(track.ID) >= 8 && !strings.HasPrefix(track.ID, "itunes_") {
		radioURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s&list=RD%s", track.ID, track.ID)
		cmd := exec.CommandContext(ctx, "yt-dlp",
			"--flat-playlist",
			"--playlist-end", strconv.Itoa(limit+8),
			"--no-warnings",
			"--print", "%(id)s\t%(title)s\t%(uploader)s\t%(duration)s",
			radioURL,
		)
		out, err := cmd.Output()
		if err == nil && len(out) > 0 {
			rawTracks = parseYouTubeTracks(string(out))
		}
	}

	// 2. Fallback: Search based on playing track's artist and title
	if len(rawTracks) == 0 {
		var query string
		if track.Artist != "" && track.Artist != "YouTube Music" {
			query = fmt.Sprintf("%s %s Radio Mix", track.Artist, track.Title)
		} else {
			query = fmt.Sprintf("%s similar songs playlist", track.Title)
		}
		var err error
		rawTracks, err = SearchYouTube(ctx, query, limit+6)
		if err != nil {
			return nil, err
		}
	}

	normCurrent := strings.ToLower(strings.TrimSpace(track.Title))
	var filtered []OnlineTrack
	for _, t := range rawTracks {
		normTitle := strings.ToLower(strings.TrimSpace(t.Title))
		// Filter out identical track ID or songs whose title contains the current track's title (e.g. covers, karaoke, remixes)
		if t.ID == track.ID {
			continue
		}
		if normCurrent != "" && (strings.Contains(normTitle, normCurrent) || (len(normCurrent) > 5 && strings.Contains(normCurrent, normTitle))) {
			continue
		}
		filtered = append(filtered, t)
		if len(filtered) >= limit {
			break
		}
	}
	DefaultCache().SetRadio(radioKey, filtered)
	return filtered, nil
}

// SearchOnline queries YouTube first for comprehensive music discovery, with iTunes fallback.
func SearchOnline(ctx context.Context, query string, limit int) ([]OnlineTrack, error) {
	if limit <= 0 {
		limit = 15
	}

	cacheKey := fmt.Sprintf("online:%d:%s", limit, query)
	if tracks, ok := DefaultCache().GetSearch(cacheKey); ok {
		return tracks, nil
	}

	// 1. YouTube Search (100% catalog coverage across all languages and genres)
	ytTracks, err := SearchYouTube(ctx, query, limit)
	if err == nil && len(ytTracks) > 0 {
		DefaultCache().SetSearch(cacheKey, ytTracks)
		return ytTracks, nil
	}

	// 2. Fallback to iTunes Search
	itunesTracks, itErr := SearchITunes(query, limit)
	if itErr == nil && len(itunesTracks) > 0 {
		var results []OnlineTrack
		for _, t := range itunesTracks {
			results = append(results, t.ToOnlineTrack())
		}
		DefaultCache().SetSearch(cacheKey, results)
		return results, nil
	}

	if err != nil {
		return nil, err
	}
	return nil, itErr
}

// DownloadToLibrary saves an online track permanently into the user's local music directory.
func DownloadToLibrary(ctx context.Context, track OnlineTrack, targetDir string) (string, error) {
	if targetDir == "" {
		return "", fmt.Errorf("no music directory configured. Run: muse dir <path>")
	}
	_ = os.MkdirAll(targetDir, 0o755)

	// Ensure the track is resolved and cached
	cachedFile, _, err := ResolveAndCache(ctx, track, nil)
	if err != nil {
		return "", fmt.Errorf("download track: %w", err)
	}

	safeName := sanitize(fmt.Sprintf("%s - %s", track.Artist, track.Title))
	if safeName == "" || safeName == " - " {
		safeName = fmt.Sprintf("track_%s", track.ID)
	}
	targetFile := filepath.Join(targetDir, safeName+".mp3")

	data, err := os.ReadFile(cachedFile)
	if err != nil {
		return "", fmt.Errorf("read cached file: %w", err)
	}

	if err := os.WriteFile(targetFile, data, 0o644); err != nil {
		return "", fmt.Errorf("write to music folder: %w", err)
	}

	return targetFile, nil
}
