package audio

import (
	"bytes"
	"testing"
)

type dummyReadSeekCloser struct {
	*bytes.Reader
}

func (d *dummyReadSeekCloser) Close() error {
	return nil
}

func TestDecodeSeekCloserUnsupported(t *testing.T) {
	dummy := &dummyReadSeekCloser{bytes.NewReader([]byte("dummy data"))}
	_, err := decodeSeekCloser(dummy, "xyz")
	if err == nil {
		t.Fatal("expected error for unsupported format, got nil")
	}
}

func TestDecodeSeekCloserInvalidMP3(t *testing.T) {
	dummy := &dummyReadSeekCloser{bytes.NewReader([]byte("not an mp3 file content"))}
	_, err := decodeSeekCloser(dummy, "mp3")
	if err == nil {
		t.Fatal("expected error for invalid mp3 data, got nil")
	}
}

func TestPlayerLoadStreamInvalid(t *testing.T) {
	p := &Player{
		sampleRate: targetSampleRate,
		volume:     0.5,
		Events:     make(chan Event, 16),
	}
	dummy := &dummyReadSeekCloser{bytes.NewReader([]byte("invalid"))}
	err := p.LoadStream(dummy, "mp3")
	if err == nil {
		t.Fatal("expected load error for invalid stream, got nil")
	}
}
