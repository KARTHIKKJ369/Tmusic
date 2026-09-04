package playlist

import (
	"crypto/rand"
	"math/big"
	"time"

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

// PeekNext returns the next track without advancing the queue position.
func (q *Queue) PeekNext() (audio.Track, bool) {
	if q.pos+1 < len(q.order) {
		return q.original[q.order[q.pos+1]], true
	}
	return audio.Track{}, false
}

// HasPrev returns true if there is a track before the current one.
func (q *Queue) HasPrev() bool {
	return q.pos > 0
}

// Len returns the queue length.
func (q *Queue) Len() int { return len(q.order) }

// Pos returns the current 0-based position.
func (q *Queue) Pos() int { return q.pos }

// Append adds a track to the end of the queue.
func (q *Queue) Append(t audio.Track) {
	idx := len(q.original)
	q.original = append(q.original, t)
	q.order = append(q.order, idx)
}

// Tracks returns all tracks currently in the queue in their playback order.
func (q *Queue) Tracks() []audio.Track {
	res := make([]audio.Track, len(q.order))
	for i, idx := range q.order {
		res[i] = q.original[idx]
	}
	return res
}

// Shuffle re-orders the queue using a Spotify-style de-clustered shuffle.
// If activeTrackID is supplied, it preserves the currently playing track position.
// If activeTrackID is omitted/empty, it starts from position 0 with a 100% uniformly random track.
func (q *Queue) Shuffle(activeTrackID ...string) {
	if len(q.original) <= 1 {
		return
	}

	var targetID string
	if len(activeTrackID) > 0 {
		targetID = activeTrackID[0]
	}

	n := len(q.original)
	perm := make([]int, n)
	for i := range perm {
		perm[i] = i
	}

	// Fisher-Yates with crypto/rand
	for i := n - 1; i > 0; i-- {
		j := cryptoRandInt(i + 1)
		perm[i], perm[j] = perm[j], perm[i]
	}

	// Spread pass: separate consecutive same-artist tracks
	perm = spreadByArtist(perm, q.original)

	q.order = perm
	q.pos = 0

	// If targetID is supplied, locate it and stay on it
	if targetID != "" {
		for i, idx := range q.order {
			if q.original[idx].ID == targetID {
				q.pos = i
				break
			}
		}
	}
}

// Unshuffle restores the original order.
func (q *Queue) Unshuffle(activeTrackID ...string) {
	var targetID string
	if len(activeTrackID) > 0 {
		targetID = activeTrackID[0]
	}
	for i := range q.order {
		q.order[i] = i
	}
	q.pos = 0
	if targetID != "" {
		for i, idx := range q.order {
			if q.original[idx].ID == targetID {
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

// spreadByArtist rearranges consecutive same-artist tracks using lookahead swaps.
// Preserves the randomized order of perm[0] while eliminating clusters.
func spreadByArtist(perm []int, tracks []audio.Track) []int {
	if len(perm) <= 2 {
		return perm
	}

	result := make([]int, len(perm))
	copy(result, perm)

	for i := 1; i < len(result)-1; i++ {
		currArtist := tracks[result[i]].Artist
		prevArtist := tracks[result[i-1]].Artist

		if currArtist != "" && currArtist == prevArtist {
			// Find a subsequent track with a different artist
			for j := i + 1; j < len(result); j++ {
				candArtist := tracks[result[j]].Artist
				if candArtist != prevArtist {
					result[i], result[j] = result[j], result[i]
					break
				}
			}
		}
	}

	return result
}

func cryptoRandInt(max int) int {
	if max <= 0 {
		return 0
	}
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err == nil {
		return int(nBig.Int64())
	}
	// Fallback to nanotime
	return int(time.Now().UnixNano() % int64(max))
}
