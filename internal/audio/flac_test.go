package audio

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestFLACOffsetReader(t *testing.T) {
	tmpDir := t.TempDir()

	// Test 1: Standard file starting directly with fLaC
	plainPath := filepath.Join(tmpDir, "plain.flac")
	plainData := append([]byte("fLaC\x00\x00\x00\x22"), bytes.Repeat([]byte{0x00}, 100)...)
	if err := os.WriteFile(plainPath, plainData, 0o644); err != nil {
		t.Fatalf("write plain: %v", err)
	}

	f1, err := os.Open(plainPath)
	if err != nil {
		t.Fatalf("open plain: %v", err)
	}
	defer f1.Close()

	r1, err := newFLACOffsetReader(f1)
	if err != nil {
		t.Fatalf("newFLACOffsetReader plain: %v", err)
	}
	if r1.offset != 0 {
		t.Errorf("expected offset 0, got %d", r1.offset)
	}

	// Test 2: File with ID3v2 tag prepended (e.g. 10 byte header + 128 bytes ID3 payload)
	id3PayloadLen := 128
	// ID3v2 header: "ID3", ver 3.0, flags 0, 4-byte syncsafe size (128 = 0x00 0x00 0x01 0x00)
	id3Header := []byte{
		'I', 'D', '3',
		0x03, 0x00, 0x00,
		0x00, 0x00, 0x01, 0x00, // syncsafe integer: 128
	}
	id3Payload := bytes.Repeat([]byte("TAGDATA"), 20)[:id3PayloadLen]
	flacData := append([]byte("fLaC\x00\x00\x00\x22"), bytes.Repeat([]byte{0x01}, 100)...)

	taggedData := append(id3Header, id3Payload...)
	taggedData = append(taggedData, flacData...)

	taggedPath := filepath.Join(tmpDir, "tagged.flac")
	if err := os.WriteFile(taggedPath, taggedData, 0o644); err != nil {
		t.Fatalf("write tagged: %v", err)
	}

	f2, err := os.Open(taggedPath)
	if err != nil {
		t.Fatalf("open tagged: %v", err)
	}
	defer f2.Close()

	r2, err := newFLACOffsetReader(f2)
	if err != nil {
		t.Fatalf("newFLACOffsetReader tagged: %v", err)
	}
	expectedOffset := int64(len(id3Header) + id3PayloadLen)
	if r2.offset != expectedOffset {
		t.Errorf("expected offset %d, got %d", expectedOffset, r2.offset)
	}

	// Verify reading from r2 yields "fLaC" at beginning
	magic := make([]byte, 4)
	if _, err := io.ReadFull(r2, magic); err != nil {
		t.Fatalf("read from r2: %v", err)
	}
	if string(magic) != "fLaC" {
		t.Errorf("expected magic 'fLaC', got '%s'", string(magic))
	}
}
