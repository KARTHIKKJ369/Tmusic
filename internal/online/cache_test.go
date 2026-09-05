package online

import (
	"testing"
	"time"
)

func TestMemoryCacheTTL(t *testing.T) {
	c := NewMemoryCache()

	c.Set("short", "val1", 50*time.Millisecond)
	c.Set("long", "val2", 1*time.Hour)

	if val, ok := c.Get("short"); !ok || val != "val1" {
		t.Fatalf("expected val1, got %v", val)
	}
	if val, ok := c.Get("long"); !ok || val != "val2" {
		t.Fatalf("expected val2, got %v", val)
	}

	time.Sleep(60 * time.Millisecond)

	if _, ok := c.Get("short"); ok {
		t.Fatal("expected short to be expired, but found")
	}
	if _, ok := c.Get("long"); !ok {
		t.Fatal("expected long to still be present")
	}

	c.PurgeExpired()
	c.Delete("long")
	if _, ok := c.Get("long"); ok {
		t.Fatal("expected long to be deleted")
	}
}

func TestMemoryCacheTyped(t *testing.T) {
	c := NewMemoryCache()

	tracks := []OnlineTrack{{ID: "1", Title: "Song 1"}}
	c.SetSearch("test", tracks)
	if res, ok := c.GetSearch("test"); !ok || len(res) != 1 || res[0].Title != "Song 1" {
		t.Fatalf("unexpected search cache result: %v", res)
	}

	sugs := []string{"sug1", "sug2"}
	c.SetSuggestions("test", sugs)
	if res, ok := c.GetSuggestions("test"); !ok || len(res) != 2 {
		t.Fatalf("unexpected suggestions cache result: %v", res)
	}

	art := []byte{0x01, 0x02, 0x03}
	c.SetArtwork("http://art.jpg", art)
	if res, ok := c.GetArtwork("http://art.jpg"); !ok || len(res) != 3 {
		t.Fatalf("unexpected artwork cache result: %v", res)
	}
}
