// Package online handles metadata discovery via the public iTunes Search API
// and audio stream resolution/caching via yt-dlp.
package online

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ITunesTrack represents a song returned by Apple's public iTunes Search API.
type ITunesTrack struct {
	TrackID         int64  `json:"trackId"`
	TrackName       string `json:"trackName"`
	ArtistName      string `json:"artistName"`
	CollectionName  string `json:"collectionName"`
	ArtworkUrl100   string `json:"artworkUrl100"`
	PreviewURL      string `json:"previewUrl"`
	TrackTimeMillis int    `json:"trackTimeMillis"`
	ReleaseDate     string `json:"releaseDate"`
	PrimaryGenre    string `json:"primaryGenreName"`
}

// Duration returns the track's duration as a time.Duration.
func (t ITunesTrack) Duration() time.Duration {
	return time.Duration(t.TrackTimeMillis) * time.Millisecond
}

// Year returns the release year if available.
func (t ITunesTrack) Year() int {
	if len(t.ReleaseDate) >= 4 {
		var y int
		if _, err := fmt.Sscanf(t.ReleaseDate[:4], "%d", &y); err == nil {
			return y
		}
	}
	return 0
}

// HighResArtworkURL returns a 600x600 high-resolution artwork URL.
func (t ITunesTrack) HighResArtworkURL() string {
	if t.ArtworkUrl100 == "" {
		return ""
	}
	// iTunes URLs end in e.g. "100x100bb.jpg". Replace with "600x600bb.jpg" for Retina quality.
	return strings.Replace(t.ArtworkUrl100, "100x100bb.jpg", "600x600bb.jpg", 1)
}

// ToOnlineTrack converts an ITunesTrack to a unified OnlineTrack.
func (t ITunesTrack) ToOnlineTrack() OnlineTrack {
	return OnlineTrack{
		ID:         fmt.Sprintf("itunes_%d", t.TrackID),
		Title:      t.TrackName,
		Artist:     t.ArtistName,
		Album:      t.CollectionName,
		Duration:   t.Duration(),
		ArtworkURL: t.HighResArtworkURL(),
		Source:     "itunes",
		Year:       t.Year(),
		Genre:      t.PrimaryGenre,
	}
}

type iTunesResponse struct {
	ResultCount int           `json:"resultCount"`
	Results     []ITunesTrack `json:"results"`
}

// SearchITunes queries the public iTunes Search API for songs matching query.
// It requires 0 credentials and returns official metadata with high-res artwork.
func SearchITunes(query string, limit int) ([]ITunesTrack, error) {
	if limit <= 0 {
		limit = 25
	}
	endpoint := fmt.Sprintf(
		"https://itunes.apple.com/search?term=%s&entity=song&limit=%d",
		url.QueryEscape(query),
		limit,
	)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "muse/0.1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("itunes search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("itunes search API returned HTTP %d", resp.StatusCode)
	}

	var data iTunesResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode itunes response: %w", err)
	}

	return data.Results, nil
}

// FetchArtwork downloads cover art bytes from an image URL.
func FetchArtwork(artworkURL string) ([]byte, error) {
	if artworkURL == "" {
		return nil, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(artworkURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch artwork HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
