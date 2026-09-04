package playlist

import (
	"math/rand"

	"github.com/KARTHIKKJ369/Tmusic/internal/audio"
)

// Queue is the active playback queue with shuffle support.
type Queue struct {
	original []audio.Track // original order
	order    []int         // indices into original
	pos      int           // current position in order
	history  []string      // ring buffer of recent track IDs (last 20)
}

// NewQueue creates a queue from a track slice.
func NewQueue(tracks []audio.Track) *Queue {
	order := make([]int, len(tracks))
	for i := range order {
		order[i] = i
	}
	return &Queue{original: tracks, order: order, pos: 0}
}

// Current returns the track at the current position.
func (q *Queue) Current() (audio.Track, bool) {
	if q.pos < 0 || q.pos >= len(q.order) {
		return audio.Track{}, false
	}
	return q.original[q.order[q.pos]], true
}

// Next advances to the next track and returns it.
func (q *Queue) Next() (audio.Track, bool) {
	q.recordHistory()
	q.pos++
	if q.pos >= len(q.order) {
		return audio.Track{}, false
	}
	return q.Current()
}

// Prev moves to the previous track.
func (q *Queue) Prev() (audio.Track, bool) {
	q.pos--
	if q.pos < 0 {
		q.pos = 0
	}
	return q.Current()
}

// JumpTo sets the queue to the track with the given ID.
func (q *Queue) JumpTo(id string) bool {
	for i, idx := range q.order {
		if q.original[idx].ID == id {
			q.pos = i
			return true
		}
	}
	return false
}

// HasNext returns true if there is a track after the current one.
func (q *Queue) HasNext() bool {
	return q.pos+1 < len(q.order)
}

// HasPrev returns true if there is a track before the current one.
func (q *Queue) HasPrev() bool {
	return q.pos > 0
}

// Len returns the queue length.
func (q *Queue) Len() int { return len(q.order) }

// Pos returns the current 0-based position.
func (q *Queue) Pos() int { return q.pos }

// Shuffle re-orders the queue using a Spotify-style de-clustered shuffle.
// It runs Fisher-Yates then spreads tracks so no two consecutive tracks
// share the same artist. After shuffling it repositions to the currently
// playing track.
func (q *Queue) Shuffle() {
	currentID := ""
	if t, ok := q.Current(); ok {
		currentID = t.ID
	}

	// Fisher-Yates
	r := rand.New(rand.NewSource(rand.Int63()))
	perm := r.Perm(len(q.original))

	// Spread pass: separate same-artist tracks
	perm = spreadByArtist(perm, q.original)

	q.order = perm
	q.pos = 0
	if currentID != "" {
		for i, idx := range q.order {
			if q.original[idx].ID == currentID {
				q.pos = i
				break
			}
		}
	}
}

// Unshuffle restores the original order.
func (q *Queue) Unshuffle() {
	currentID := ""
	if t, ok := q.Current(); ok {
		currentID = t.ID
	}
	for i := range q.order {
		q.order[i] = i
	}
	q.pos = 0
	if currentID != "" {
		for i, idx := range q.order {
			if q.original[idx].ID == currentID {
				q.pos = i
				break
			}
		}
	}
}

func (q *Queue) recordHistory() {
	if t, ok := q.Current(); ok {
		q.history = append(q.history, t.ID)
		if len(q.history) > 20 {
			q.history = q.history[1:]
		}
	}
}

// spreadByArtist rearranges perm so consecutive entries don't share an artist.
// Uses a greedy algorithm: always pick the candidate with the most-different artist
// from the previous pick.
func spreadByArtist(perm []int, tracks []audio.Track) []int {
	if len(perm) <= 1 {
		return perm
	}

	// Build a frequency map: artist → remaining count.
	freq := make(map[string]int)
	available := make([]int, len(perm))
	copy(available, perm)

	for _, idx := range perm {
		freq[tracks[idx].Artist]++
	}

	result := make([]int, 0, len(perm))
	lastArtist := ""

	for len(available) > 0 {
		// Try to pick a track with a different artist.
		chosen := -1
		chosenArtist := ""
		maxFreq := -1

		for i, idx := range available {
			artist := tracks[idx].Artist
			if artist == lastArtist {
				continue
			}
			if freq[artist] > maxFreq {
				maxFreq = freq[artist]
				chosen = i
				chosenArtist = artist
			}
		}

		// Fallback: no different artist available.
		if chosen == -1 {
			chosen = 0
			chosenArtist = tracks[available[0]].Artist
		}

		result = append(result, available[chosen])
		freq[chosenArtist]--
		lastArtist = chosenArtist
		available = append(available[:chosen], available[chosen+1:]...)
	}

	return result
}
