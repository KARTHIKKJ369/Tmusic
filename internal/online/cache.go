package online

import (
	"sync"
	"time"
)

type cacheItem struct {
	value     any
	expiresAt time.Time
}

func (i cacheItem) isExpired() bool {
	return !i.expiresAt.IsZero() && time.Now().After(i.expiresAt)
}

// MemoryCache is a thread-safe in-memory cache with time-to-live expiration.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]cacheItem
}

var defaultCache = NewMemoryCache()

// DefaultCache returns the package-level shared memory cache.
func DefaultCache() *MemoryCache {
	return defaultCache
}

// NewMemoryCache creates an initialized MemoryCache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		items: make(map[string]cacheItem),
	}
}

// Get retrieves a value by key if not expired.
func (c *MemoryCache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found || item.isExpired() {
		return nil, false
	}
	return item.value, true
}

// Set stores a value with a given time-to-live.
func (c *MemoryCache) Set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	c.items[key] = cacheItem{
		value:     value,
		expiresAt: expiresAt,
	}
}

// Delete removes a key from the cache.
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// PurgeExpired removes all expired items from the cache.
func (c *MemoryCache) PurgeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for k, item := range c.items {
		if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
			delete(c.items, k)
		}
	}
}

// Typed convenience methods on MemoryCache

// GetSearch retrieves cached search results.
func (c *MemoryCache) GetSearch(query string) ([]OnlineTrack, bool) {
	val, ok := c.Get("search:" + query)
	if !ok {
		return nil, false
	}
	tracks, ok := val.([]OnlineTrack)
	return tracks, ok
}

// SetSearch stores search results with a 1-hour TTL.
func (c *MemoryCache) SetSearch(query string, tracks []OnlineTrack) {
	c.Set("search:"+query, tracks, 1*time.Hour)
}

// GetSuggestions retrieves cached search suggestions.
func (c *MemoryCache) GetSuggestions(query string) ([]string, bool) {
	val, ok := c.Get("sug:" + query)
	if !ok {
		return nil, false
	}
	sugs, ok := val.([]string)
	return sugs, ok
}

// SetSuggestions stores search suggestions with a 30-minute TTL.
func (c *MemoryCache) SetSuggestions(query string, suggestions []string) {
	c.Set("sug:"+query, suggestions, 30*time.Minute)
}

// GetRadio retrieves cached radio/related tracks.
func (c *MemoryCache) GetRadio(trackID string) ([]OnlineTrack, bool) {
	val, ok := c.Get("radio:" + trackID)
	if !ok {
		return nil, false
	}
	tracks, ok := val.([]OnlineTrack)
	return tracks, ok
}

// SetRadio stores radio/related tracks with a 24-hour TTL.
func (c *MemoryCache) SetRadio(trackID string, tracks []OnlineTrack) {
	c.Set("radio:"+trackID, tracks, 24*time.Hour)
}

// GetArtwork retrieves cached artwork bytes.
func (c *MemoryCache) GetArtwork(url string) ([]byte, bool) {
	val, ok := c.Get("art:" + url)
	if !ok {
		return nil, false
	}
	data, ok := val.([]byte)
	return data, ok
}

// SetArtwork stores artwork bytes in memory with a 6-hour TTL.
func (c *MemoryCache) SetArtwork(url string, data []byte) {
	c.Set("art:"+url, data, 6*time.Hour)
}
