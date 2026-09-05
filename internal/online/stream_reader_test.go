package online

import (
	"context"
	"io"
	"os"
	"testing"
	"time"
)

func TestProgressiveFileStream(t *testing.T) {
	tmp, err := os.CreateTemp("", "prog_test_*.dat")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	done := make(chan struct{})
	var writerErr error

	// Write 10 bytes first
	_, _ = tmp.Write([]byte("0123456789"))
	_ = tmp.Sync()

	readF, err := os.Open(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}

	pfs := NewProgressiveFileStream(context.Background(), nil, readF, done, &writerErr, nil)
	defer pfs.Close()

	// Read first 10 bytes
	buf := make([]byte, 10)
	n, err := pfs.Read(buf)
	if err != nil || n != 10 || string(buf) != "0123456789" {
		t.Fatalf("first read failed: n=%d err=%v buf=%s", n, err, string(buf))
	}

	// Concurrently write second chunk after 50ms
	go func() {
		time.Sleep(50 * time.Millisecond)
		_, _ = tmp.Write([]byte("abcdefghij"))
		_ = tmp.Sync()
		time.Sleep(20 * time.Millisecond)
		close(done)
		_ = tmp.Close()
	}()

	// Progressive read should block until second chunk arrives
	buf2 := make([]byte, 10)
	n, err = pfs.Read(buf2)
	if err != nil || n != 10 || string(buf2) != "abcdefghij" {
		t.Fatalf("second progressive read failed: n=%d err=%v buf=%s", n, err, string(buf2))
	}

	// Next read should return EOF since writer is closed
	buf3 := make([]byte, 10)
	n, err = pfs.Read(buf3)
	if err != io.EOF {
		t.Fatalf("expected EOF after writer completed, got n=%d err=%v", n, err)
	}
}
