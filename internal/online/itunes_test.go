package online

import (
	"testing"
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
