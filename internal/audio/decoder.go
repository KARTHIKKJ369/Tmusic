package audio

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/vorbis"
	"github.com/gopxl/beep/wav"
)

// openFile is a helper used by metadata and decoder.
func openFile(path string) (*os.File, error) {
	return os.Open(path)
}

// decoded holds the decoded stream and format for a track.
type decoded struct {
	stream beep.StreamSeekCloser
	format beep.Format
}

// decodeSeekCloser decodes an io.ReadSeekCloser based on format extension.
func decodeSeekCloser(r io.ReadSeekCloser, ext string) (decoded, error) {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	switch ext {
	case "mp3":
		s, fmt_, err := mp3.Decode(r)
		if err != nil {
			r.Close()
			return decoded{}, err
		}
		return decoded{s, fmt_}, nil
	case "flac":
		if f, ok := r.(*os.File); ok {
			s, fmt_, err := robustFLACDecode(f)
			if err != nil {
				f.Close()
				return decoded{}, err
			}
			return decoded{s, fmt_}, nil
		}
		r.Close()
		return decoded{}, fmt.Errorf("flac streaming requires *os.File")
	case "wav":
		s, fmt_, err := wav.Decode(r)
		if err != nil {
			r.Close()
			return decoded{}, err
		}
		return decoded{s, fmt_}, nil
	case "ogg":
		s, fmt_, err := vorbis.Decode(r)
		if err != nil {
			r.Close()
			return decoded{}, err
		}
		return decoded{s, fmt_}, nil
	default:
		r.Close()
		return decoded{}, fmt.Errorf("unsupported format: %s", ext)
	}
}

// decode opens path and returns a seekable decoded stream.
func decode(path string) (decoded, error) {
	f, err := openFile(path)
	if err != nil {
		return decoded{}, err
	}

	ext := path[strings.LastIndex(path, ".")+1:]
	return decodeSeekCloser(f, ext)
}

// Duration probes the track's duration without keeping the stream open.
func Duration(path string) (time.Duration, error) {
	d, err := decode(path)
	if err != nil {
		return 0, err
	}
	defer d.stream.Close()
	samples := d.stream.Len()
	if samples <= 0 {
		return 0, nil
	}
	return d.format.SampleRate.D(samples), nil
}
