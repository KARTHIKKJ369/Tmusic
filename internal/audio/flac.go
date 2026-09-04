package audio

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gopxl/beep"
	"github.com/mewkiz/flac"
)

// flacOffsetReader wraps an os.File so that Read and Seek are relative
// to the actual 'fLaC' signature offset (skipping any prepended ID3v2 headers).
type flacOffsetReader struct {
	file   *os.File
	offset int64
	size   int64
}

func newFLACOffsetReader(file *os.File) (*flacOffsetReader, error) {
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	size := stat.Size()

	// Find the 'fLaC' magic bytes (check offset 0 first, then ID3 header, then scan first 64KB)
	var flacOffset int64 = 0
	buf := make([]byte, 4)
	if _, err := file.ReadAt(buf, 0); err == nil {
		if string(buf) == "fLaC" {
			flacOffset = 0
		} else if string(buf[:3]) == "ID3" {
			// ID3v2 header: 10 bytes (magic + version + flags + 4-byte syncsafe size)
			hdr := make([]byte, 10)
			if _, err := file.ReadAt(hdr, 0); err == nil {
				id3Size := int64(hdr[6])<<21 | int64(hdr[7])<<14 | int64(hdr[8])<<7 | int64(hdr[9])
				candidate := 10 + id3Size
				chk := make([]byte, 4)
				if _, err := file.ReadAt(chk, candidate); err == nil && string(chk) == "fLaC" {
					flacOffset = candidate
				}
			}
		}
	}

	if flacOffset == 0 && string(buf) != "fLaC" {
		// Scan first 64KB for 'fLaC' magic bytes
		scanBuf := make([]byte, 65536)
		n, _ := file.ReadAt(scanBuf, 0)
		idx := bytes.Index(scanBuf[:n], []byte("fLaC"))
		if idx >= 0 {
			flacOffset = int64(idx)
		}
	}

	_, _ = file.Seek(flacOffset, io.SeekStart)
	return &flacOffsetReader{
		file:   file,
		offset: flacOffset,
		size:   size - flacOffset,
	}, nil
}

func (r *flacOffsetReader) Read(p []byte) (int, error) {
	return r.file.Read(p)
}

func (r *flacOffsetReader) Seek(offset int64, whence int) (int64, error) {
	var target int64
	switch whence {
	case io.SeekStart:
		target = r.offset + offset
	case io.SeekCurrent:
		curr, err := r.file.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, err
		}
		target = curr + offset
	case io.SeekEnd:
		target = (r.offset + r.size) + offset
	default:
		return 0, errors.New("invalid whence")
	}

	if target < r.offset {
		target = r.offset
	}

	n, err := r.file.Seek(target, io.SeekStart)
	if err != nil {
		return 0, err
	}
	return n - r.offset, nil
}

func (r *flacOffsetReader) Close() error {
	return r.file.Close()
}

// robustFLACDecode decodes FLAC streams with tolerance for ID3 headers and trailing EOF truncation.
func robustFLACDecode(file *os.File) (beep.StreamSeekCloser, beep.Format, error) {
	r, err := newFLACOffsetReader(file)
	if err != nil {
		file.Close()
		return nil, beep.Format{}, err
	}

	stream, err := flac.NewSeek(r)
	seekEnabled := true
	if err != nil {
		// Fallback to sequential flac.New if seek table parsing failed
		_, _ = r.file.Seek(r.offset, io.SeekStart)
		stream, err = flac.New(r)
		seekEnabled = false
		if err != nil {
			r.Close()
			return nil, beep.Format{}, fmt.Errorf("flac: %w", err)
		}
	}

	format := beep.Format{
		SampleRate:  beep.SampleRate(stream.Info.SampleRate),
		NumChannels: int(stream.Info.NChannels),
		Precision:   int(stream.Info.BitsPerSample / 8),
	}

	d := &flacDecoder{
		r:           r,
		stream:      stream,
		seekEnabled: seekEnabled,
	}

	return d, format, nil
}

type flacDecoder struct {
	r           *flacOffsetReader
	stream      *flac.Stream
	buf         [][2]float64
	pos         int
	err         error
	seekEnabled bool
}

func (d *flacDecoder) Stream(samples [][2]float64) (n int, ok bool) {
	if d.err != nil {
		return 0, false
	}
	for i := range samples {
		if len(d.buf) == 0 {
			if err := d.refill(); err != nil {
				d.err = err
				return i, i > 0
			}
		}
		samples[i] = d.buf[0]
		d.buf = d.buf[1:]
		d.pos++
	}
	return len(samples), true
}

func (d *flacDecoder) refill() error {
	d.buf = d.buf[:0]
	frame, err := d.stream.ParseNext()
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || strings.Contains(err.Error(), "unexpected EOF") {
			return io.EOF
		}
		return err
	}
	if len(frame.Subframes) == 0 || len(frame.Subframes[0].Samples) == 0 {
		return io.EOF
	}

	n := len(frame.Subframes[0].Samples)
	if cap(d.buf) < n {
		d.buf = make([][2]float64, n)
	} else {
		d.buf = d.buf[:n]
	}

	bps := d.stream.Info.BitsPerSample
	nchannels := d.stream.Info.NChannels
	s := 1 << (bps - 1)
	q := 1 / float64(s)

	switch {
	case bps <= 16 && nchannels == 1:
		for i := 0; i < n; i++ {
			v := float64(int16(frame.Subframes[0].Samples[i])) * q
			d.buf[i][0] = v
			d.buf[i][1] = v
		}
	case bps > 16 && nchannels == 1:
		for i := 0; i < n; i++ {
			v := float64(int32(frame.Subframes[0].Samples[i])) * q
			d.buf[i][0] = v
			d.buf[i][1] = v
		}
	case bps <= 16 && nchannels >= 2:
		for i := 0; i < n; i++ {
			d.buf[i][0] = float64(int16(frame.Subframes[0].Samples[i])) * q
			d.buf[i][1] = float64(int16(frame.Subframes[1].Samples[i])) * q
		}
	case bps > 16 && nchannels >= 2:
		for i := 0; i < n; i++ {
			d.buf[i][0] = float64(frame.Subframes[0].Samples[i]) * q
			d.buf[i][1] = float64(frame.Subframes[1].Samples[i]) * q
		}
	default:
		for i := 0; i < n; i++ {
			d.buf[i][0] = float64(frame.Subframes[0].Samples[i]) * q
			d.buf[i][1] = float64(frame.Subframes[0].Samples[i]) * q
		}
	}

	return nil
}

func (d *flacDecoder) Err() error {
	if d.err == nil || errors.Is(d.err, io.EOF) || errors.Is(d.err, io.ErrUnexpectedEOF) || strings.Contains(d.err.Error(), "unexpected EOF") {
		return nil
	}
	return d.err
}

func (d *flacDecoder) Len() int {
	return int(d.stream.Info.NSamples)
}

func (d *flacDecoder) Position() int {
	return d.pos
}

func (d *flacDecoder) Seek(p int) error {
	if !d.seekEnabled {
		return errors.New("flac: seek not enabled")
	}
	d.buf = d.buf[:0]
	d.err = nil
	pos, err := d.stream.Seek(uint64(p))
	d.pos = int(pos)
	return err
}

func (d *flacDecoder) Close() error {
	if d.r != nil {
		return d.r.Close()
	}
	return nil
}
