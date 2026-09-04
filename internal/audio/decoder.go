package audio

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/flac"
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

// decode opens path and returns a seekable decoded stream.
func decode(path string) (decoded, error) {
	f, err := openFile(path)
	if err != nil {
		return decoded{}, err
	}

	ext := strings.ToLower(path[strings.LastIndex(path, ".")+1:])
	switch ext {
	case "mp3":
		s, fmt_, err := mp3.Decode(f)
		if err != nil {
			f.Close()
			return decoded{}, err
		}
		return decoded{s, fmt_}, nil
	case "flac":
		s, fmt_, err := flac.Decode(f)
		if err != nil {
			f.Close()
			return decoded{}, err
		}
		return decoded{s, fmt_}, nil
	case "wav":
		s, fmt_, err := wav.Decode(f)
		if err != nil {
			f.Close()
			return decoded{}, err
		}
		return decoded{s, fmt_}, nil
	case "ogg":
		s, fmt_, err := vorbis.Decode(f)
		if err != nil {
			f.Close()
			return decoded{}, err
		}
		return decoded{s, fmt_}, nil
	default:
		f.Close()
		return decoded{}, fmt.Errorf("unsupported format: %s", ext)
	}
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
