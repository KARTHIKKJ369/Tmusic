package online

import (
	"testing"
	"time"
)

func TestHighResArtworkURL(t *testing.T) {
	track := ITunesTrack{
		ArtworkUrl100: "https://is1-ssl.mzstatic.com/image/thumb/Music115/v4/e8/43/5f/e8435ffa-b6b9-b171-40ab-4ff3959ab661/886443919266.jpg/100x100bb.jpg",
	}
	expected := "https://is1-ssl.mzstatic.com/image/thumb/Music115/v4/e8/43/5f/e8435ffa-b6b9-b171-40ab-4ff3959ab661/886443919266.jpg/600x600bb.jpg"
	if track.HighResArtworkURL() != expected {
		t.Errorf("expected %s, got %s", expected, track.HighResArtworkURL())
	}
}

func TestSearchITunes(t *testing.T) {
	tracks, err := SearchITunes("Daft Punk Get Lucky", 5)
	if err != nil {
		t.Skipf("skipping network test: %v", err)
	}
	if len(tracks) == 0 {
		t.Fatalf("expected search results for Daft Punk Get Lucky, got 0")
	}
	first := tracks[0]
	if first.TrackName == "" || first.ArtistName == "" {
		t.Errorf("empty track metadata: %+v", first)
	}
	if first.Duration() <= 0 {
		t.Errorf("invalid track duration: %v", first.Duration())
	}
}

func TestCheckDependencies(t *testing.T) {
	if err := CheckDependencies(); err != nil {
		t.Logf("dependencies not installed in this environment: %v", err)
	}
}

func TestCacheDir(t *testing.T) {
	dir, err := CacheDir()
	if err != nil {
		t.Fatalf("CacheDir error: %v", err)
	}
	if dir == "" {
		t.Fatalf("empty CacheDir")
	}
}

func TestFetchSuggestions(t *testing.T) {
	sugs, err := FetchSuggestions("ente ellam")
	if err != nil {
		t.Skipf("network error fetching suggestions: %v", err)
	}
	if len(sugs) == 0 {
		t.Errorf("expected suggestions for 'ente ellam', got 0")
	}
	t.Logf("Suggestions: %v", sugs)
}

func TestCleanYouTubeMetadata(t *testing.T) {
	raw := "Ente Ellam Ellam Alle Video Song | Meesamadhavan | Dileep | Kavya Madhavan | Vidyasagar"
	uploader := "Sony Music Malayalam"
	title, artist := CleanYouTubeMetadata(raw, uploader)
	if title != "Ente Ellam Ellam Alle" {
		t.Errorf("expected 'Ente Ellam Ellam Alle', got %q", title)
	}
	if artist != "Sony Music Malayalam" {
		t.Errorf("expected 'Sony Music Malayalam', got %q", artist)
	}
}

func TestSearchYouTube(t *testing.T) {
	tracks, err := SearchYouTube(t.Context(), "ente ellam ellam alle", 3)
	if err != nil {
		t.Skipf("youtube search error: %v", err)
	}
	if len(tracks) == 0 {
		t.Fatalf("expected tracks for 'ente ellam ellam alle', got 0")
	}
	t.Logf("Found %d tracks: %+v", len(tracks), tracks[0])
	if tracks[0].Title == "" || tracks[0].ID == "" {
		t.Errorf("expected non-empty Title and ID: %+v", tracks[0])
	}
}

func TestResolveAndCache(t *testing.T) {
	track := OnlineTrack{
		ID:     "sZvDWK8bjt8",
		Title:  "Ente Ellam Ellam Alle",
		Artist: "Sony Music Malayalam",
		Source: "youtube",
	}
	start := time.Now()
	path, art, err := ResolveAndCache(t.Context(), track, nil)
	if err != nil {
		t.Skipf("resolve error: %v", err)
	}
	elapsed := time.Since(start)
	t.Logf("Resolved and cached in %v -> path: %s (artwork: %d bytes)", elapsed, path, len(art))
	if path == "" {
		t.Errorf("expected non-empty audio path")
	}

	// Verify second call is 0ms instant cache hit
	start2 := time.Now()
	cachedPath, _, found := GetCachedTrack(track)
	elapsed2 := time.Since(start2)
	if !found || cachedPath != path {
		t.Errorf("expected instant cache hit, got found=%v, path=%s", found, cachedPath)
	}
	t.Logf("Cache hit verified in %v", elapsed2)
}
