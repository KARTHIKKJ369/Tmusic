package audio

import (
	"io"
	"sync"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/speaker"
)

// EventType describes what happened in the player.
type EventType int

const (
	EventTrackEnd EventType = iota
	EventError
	EventTick // fired ~every second for progress updates
)

// Event is emitted on the Events channel.
type Event struct {
	Type EventType
	Err  error
}

// Player is a goroutine-safe audio playback engine.
type Player struct {
	mu      sync.Mutex
	current *decoded
	ctrl    *beep.Ctrl
	vol     *effects.Volume
	done    chan struct{}
	Events  chan Event

	sampleRate beep.SampleRate
	volume     float64 // 0.0–1.0
}

const targetSampleRate beep.SampleRate = 44100

// NewPlayer initialises the speaker and returns a ready Player.
func NewPlayer(volume float64) (*Player, error) {
	if err := speaker.Init(targetSampleRate, targetSampleRate.N(time.Second/10)); err != nil {
		return nil, err
	}
	p := &Player{
		sampleRate: targetSampleRate,
		volume:     clamp(volume, 0, 1),
		Events:     make(chan Event, 16),
	}
	return p, nil
}

// Load opens a track file ready to play (does not start playback).
func (p *Player) Load(path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopLocked()

	d, err := decode(path)
	if err != nil {
		return err
	}
	return p.loadDecodedLocked(d)
}

// LoadStream opens an io.ReadSeekCloser stream ready to play (does not start playback).
func (p *Player) LoadStream(stream io.ReadSeekCloser, format string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopLocked()

	d, err := decodeSeekCloser(stream, format)
	if err != nil {
		return err
	}
	return p.loadDecodedLocked(d)
}

func (p *Player) loadDecodedLocked(d decoded) error {
	// Resample if needed.
	var stream beep.StreamSeekCloser = d.stream
	if d.format.SampleRate != targetSampleRate {
		resampled := beep.Resample(4, d.format.SampleRate, targetSampleRate, d.stream)
		// Wrap back into a StreamSeekCloser via adapter.
		stream = &resampleAdapter{Streamer: resampled, inner: d.stream}
	}

	p.current = &decoded{stream: stream, format: d.format}
	p.done = make(chan struct{})

	ctrl := &beep.Ctrl{Streamer: stream, Paused: true}
	vol := &effects.Volume{
		Streamer: ctrl,
		Base:     2,
		Volume:   volumeToBeep(p.volume),
		Silent:   p.volume == 0,
	}
	p.ctrl = ctrl
	p.vol = vol
	return nil
}

// Play starts or resumes playback.
func (p *Player) Play() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ctrl == nil {
		return
	}
	done := p.done
	p.ctrl.Paused = false

	speaker.Play(beep.Seq(p.vol, beep.Callback(func() {
		select {
		case p.Events <- Event{Type: EventTrackEnd}:
		default:
		}
		close(done)
	})))

	// tick goroutine
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				select {
				case p.Events <- Event{Type: EventTick}:
				default:
				}
			}
		}
	}()
}

// Pause pauses playback without closing the stream.
func (p *Player) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ctrl != nil {
		speaker.Lock()
		p.ctrl.Paused = true
		speaker.Unlock()
	}
}

// TogglePause flips the pause state.
func (p *Player) TogglePause() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ctrl == nil {
		return false
	}
	speaker.Lock()
	p.ctrl.Paused = !p.ctrl.Paused
	paused := p.ctrl.Paused
	speaker.Unlock()
	return paused
}

// Paused returns whether playback is currently paused.
func (p *Player) Paused() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ctrl == nil {
		return true
	}
	speaker.Lock()
	defer speaker.Unlock()
	return p.ctrl.Paused
}

// Seek moves the playback position by delta (can be negative).
func (p *Player) Seek(delta time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.current == nil {
		return nil
	}
	speaker.Lock()
	defer speaker.Unlock()

	pos := p.current.stream.Position()
	target := pos + p.current.format.SampleRate.N(delta)
	total := p.current.stream.Len()
	target = clampInt(target, 0, total-1)
	return p.current.stream.Seek(target)
}

// SeekTo sets the playback position to an absolute duration.
func (p *Player) SeekTo(target time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.current == nil {
		return nil
	}
	speaker.Lock()
	defer speaker.Unlock()

	sampleTarget := p.current.format.SampleRate.N(target)
	total := p.current.stream.Len()
	sampleTarget = clampInt(sampleTarget, 0, total-1)
	return p.current.stream.Seek(sampleTarget)
}

// SeekPercent sets playback position to a percentage (0.0 to 1.0) of the total track.
func (p *Player) SeekPercent(pct float64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.current == nil {
		return nil
	}
	speaker.Lock()
	defer speaker.Unlock()

	total := p.current.stream.Len()
	sampleTarget := int(float64(total) * clamp(pct, 0, 1))
	sampleTarget = clampInt(sampleTarget, 0, total-1)
	return p.current.stream.Seek(sampleTarget)
}

// Position returns current playback position.
func (p *Player) Position() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.current == nil {
		return 0
	}
	speaker.Lock()
	defer speaker.Unlock()
	return p.current.format.SampleRate.D(p.current.stream.Position())
}

// SetVolume adjusts volume (0.0–1.0).
func (p *Player) SetVolume(v float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.volume = clamp(v, 0, 1)
	if p.vol == nil {
		return
	}
	speaker.Lock()
	p.vol.Volume = volumeToBeep(p.volume)
	p.vol.Silent = p.volume == 0
	speaker.Unlock()
}

// Volume returns current volume (0.0–1.0).
func (p *Player) Volume() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.volume
}

// Stop stops playback and releases the stream.
func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopLocked()
}

func (p *Player) stopLocked() {
	if p.current != nil {
		speaker.Clear()
		p.current.stream.Close()
		p.current = nil
		p.ctrl = nil
		p.vol = nil
	}
}

// volumeToBeep converts 0-1 linear to beep's logarithmic volume scale.
// beep.Volume uses base-2 log; -5 is silent, 0 is unity.
func volumeToBeep(v float64) float64 {
	if v <= 0 {
		return -10
	}
	// Map 0-1 to roughly -5 to 0
	return (v - 1) * 5
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// resampleAdapter wraps a resampled streamer while keeping the inner seekable stream accessible.
type resampleAdapter struct {
	beep.Streamer
	inner beep.StreamSeekCloser
}

func (r *resampleAdapter) Seek(n int) error  { return r.inner.Seek(n) }
func (r *resampleAdapter) Position() int      { return r.inner.Position() }
func (r *resampleAdapter) Len() int           { return r.inner.Len() }
func (r *resampleAdapter) Close() error       { return r.inner.Close() }
func (r *resampleAdapter) Err() error         { return nil }
