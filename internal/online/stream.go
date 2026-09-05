package online

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// StreamResult contains the resolved stream or file ready for instant playback.
type StreamResult struct {
	Track     OnlineTrack
	FilePath  string            // Set if the file is completely cached on disk
	Stream    io.ReadSeekCloser // Set if progressive streaming is active
	Artwork   []byte            // Album artwork bytes if available
	FromCache bool              // True if loaded immediately from local cache
}

// CheckDependencies verifies whether yt-dlp and ffmpeg are available on the system.
func CheckDependencies() error {
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		switch runtime.GOOS {
		case "darwin":
			return fmt.Errorf("`yt-dlp` is required for online music streaming.\nInstall it via Homebrew: brew install yt-dlp")
		case "windows":
			return fmt.Errorf("`yt-dlp` is required for online music streaming.\nInstall it via winget: winget install yt-dlp")
		default: // linux, bsd
			return fmt.Errorf("`yt-dlp` is required for online music streaming.\nInstall via package manager or python: pip install yt-dlp")
		}
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		switch runtime.GOOS {
		case "darwin":
			return fmt.Errorf("`ffmpeg` is required to process audio streams.\nInstall it via Homebrew: brew install ffmpeg")
		case "windows":
			return fmt.Errorf("`ffmpeg` is required to process audio streams.\nInstall it via winget: winget install ffmpeg")
		default:
			return fmt.Errorf("`ffmpeg` is required to process audio streams.\nInstall via package manager: sudo apt install ffmpeg")
		}
	}
	return nil
}

// CacheDir returns the directory used to cache streamed online tracks.
func CacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "muse", "online")
	return dir, os.MkdirAll(dir, 0o755)
}

type streamJob struct {
	partFile   string
	cacheFile  string
	done       chan struct{}
	buffered   chan struct{}
	bufferOnce sync.Once
	err        error
	cancel     context.CancelFunc
}

var (
	streamJobsMu sync.Mutex
	streamJobs   = make(map[string]*streamJob)
)

func getOrCreateStreamJob(cacheKey, safeName, cacheDir string, track OnlineTrack) (*streamJob, bool) {
	streamJobsMu.Lock()
	defer streamJobsMu.Unlock()

	if job, exists := streamJobs[cacheKey]; exists {
		return job, false
	}

	partFile := filepath.Join(cacheDir, fmt.Sprintf("part_%s_%s.mp3", cacheKey, safeName))
	cacheFile := filepath.Join(cacheDir, fmt.Sprintf("%s_%s.mp3", cacheKey, safeName))

	ctx, cancel := context.WithCancel(context.Background())
	job := &streamJob{
		partFile:  partFile,
		cacheFile: cacheFile,
		done:      make(chan struct{}),
		buffered:  make(chan struct{}),
		cancel:    cancel,
	}
	streamJobs[cacheKey] = job

	go runStreamJob(ctx, job, track, cacheKey)

	return job, true
}

func runStreamJob(ctx context.Context, job *streamJob, track OnlineTrack, cacheKey string) {
	defer func() {
		job.bufferOnce.Do(func() { close(job.buffered) })
		close(job.done)
		streamJobsMu.Lock()
		delete(streamJobs, cacheKey)
		streamJobsMu.Unlock()
	}()

	// Monitor file size in background to notify when initial 64KB threshold is met
	stopMonitor := make(chan struct{})
	go func() {
		ticker := time.NewTicker(25 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stopMonitor:
				return
			case <-ticker.C:
				if fi, err := os.Stat(job.partFile); err == nil && fi.Size() >= 64*1024 {
					job.bufferOnce.Do(func() { close(job.buffered) })
					return
				}
			}
		}
	}()
	defer close(stopMonitor)

	_ = os.Remove(job.partFile)

	// 1. Fast path: Direct YouTube stream URL extraction and ffmpeg pipe
	var transcodeErr error
	if track.Source == "youtube" && len(track.ID) >= 8 && !strings.HasPrefix(track.ID, "itunes_") {
		videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", track.ID)
		cmdURL := exec.CommandContext(ctx, "yt-dlp",
			"--extractor-args", "youtube:player_client=android",
			"-f", "18/ba/b",
			"-g",
			videoURL,
		)
		outURL, err := cmdURL.Output()
		if err == nil {
			streamURL := strings.TrimSpace(strings.Split(string(outURL), "\n")[0])
			if streamURL != "" && strings.HasPrefix(streamURL, "http") {
				cmdFF := exec.CommandContext(ctx, "ffmpeg",
					"-y",
					"-i", streamURL,
					"-vn",
					"-c:a", "libmp3lame",
					"-q:a", "5",
					job.partFile,
				)
				if err := cmdFF.Run(); err == nil {
					if fi, err := os.Stat(job.partFile); err == nil && fi.Size() > 20000 {
						_ = os.Rename(job.partFile, job.cacheFile)
						return
					}
				} else {
					transcodeErr = err
				}
			}
		}
	}

	// 2. Fallback path: Full yt-dlp extraction
	tempTarget := filepath.Join(filepath.Dir(job.partFile), fmt.Sprintf("tmp_%s.%%(ext)s", cacheKey))
	var args []string
	if track.Source == "youtube" && len(track.ID) >= 8 && !strings.HasPrefix(track.ID, "itunes_") {
		videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", track.ID)
		args = []string{
			"--extractor-args", "youtube:player_client=android",
			"-f", "140/ba/18/best",
			"-x",
			"--audio-format", "mp3",
			"--audio-quality", "5",
			"-o", tempTarget,
			"--no-playlist",
			videoURL,
		}
	} else {
		query := fmt.Sprintf("%s %s Official Audio", track.Artist, track.Title)
		args = []string{
			"--extractor-args", "youtube:player_client=android",
			"-f", "140/ba/18/best",
			"-x",
			"--audio-format", "mp3",
			"--audio-quality", "5",
			"-o", tempTarget,
			"--no-playlist",
			"--default-search", "ytsearch1:",
			query,
		}
	}

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	_, err := cmd.CombinedOutput()
	if err != nil && track.Source != "youtube" {
		fallbackQuery := fmt.Sprintf("%s %s", track.Artist, track.Title)
		args[len(args)-1] = fallbackQuery
		cmdRetry := exec.CommandContext(ctx, "yt-dlp", args...)
		_, err = cmdRetry.CombinedOutput()
	}

	if err != nil {
		job.err = fmt.Errorf("unable to stream \"%s - %s\": %w", track.Artist, track.Title, err)
		return
	}

	// Locate generated mp3 file and rename to cacheFile
	expectedTempMp3 := filepath.Join(filepath.Dir(job.partFile), fmt.Sprintf("tmp_%s.mp3", cacheKey))
	if _, err := os.Stat(expectedTempMp3); err == nil {
		_ = os.Rename(expectedTempMp3, job.cacheFile)
		return
	}

	files, _ := filepath.Glob(filepath.Join(filepath.Dir(job.partFile), fmt.Sprintf("tmp_%s.*", cacheKey)))
	for _, f := range files {
		if strings.HasSuffix(f, ".mp3") {
			_ = os.Rename(f, job.cacheFile)
			return
		}
	}

	if fi, err := os.Stat(job.partFile); err == nil && fi.Size() > 20000 {
		_ = os.Rename(job.partFile, job.cacheFile)
		return
	}

	job.err = fmt.Errorf("stream failed: %v", transcodeErr)
}

// ResolveStream resolves a track for playback, returning as soon as initial buffer is ready
// without waiting for the full track download.
func ResolveStream(ctx context.Context, track OnlineTrack, onStatus func(string)) (StreamResult, error) {
	cacheDir, err := CacheDir()
	if err != nil {
		return StreamResult{}, fmt.Errorf("get cache directory: %w", err)
	}

	var cacheKey string
	if track.ID != "" {
		cacheKey = sanitize(track.ID)
	} else {
		h := sha256.Sum256([]byte(track.Artist + " - " + track.Title))
		cacheKey = fmt.Sprintf("%x", h[:8])
	}
	safeName := sanitize(track.Artist + " - " + track.Title)
	if len(safeName) > 40 {
		safeName = safeName[:40]
	}
	cacheFile := filepath.Join(cacheDir, fmt.Sprintf("%s_%s.mp3", cacheKey, safeName))
	artCacheFile := filepath.Join(cacheDir, fmt.Sprintf("%s_%s.jpg", cacheKey, safeName))

	// Parallel non-blocking artwork fetch in background
	go func() {
		if _, err := os.Stat(artCacheFile); err == nil {
			return
		}
		if track.ArtworkURL != "" {
			if data, err := FetchArtwork(track.ArtworkURL); err == nil && len(data) > 0 {
				_ = os.WriteFile(artCacheFile, data, 0o644)
			}
		}
	}()

	artData, _ := os.ReadFile(artCacheFile)

	// If audio is already in cache and valid, return immediately (0ms playback)
	if fi, err := os.Stat(cacheFile); err == nil && fi.Size() > 50000 {
		if onStatus != nil {
			onStatus("Loaded from local cache")
		}
		return StreamResult{
			Track:     track,
			FilePath:  cacheFile,
			Artwork:   artData,
			FromCache: true,
		}, nil
	}

	if err := CheckDependencies(); err != nil {
		return StreamResult{}, err
	}

	if onStatus != nil {
		onStatus("Buffering audio stream...")
	}

	job, _ := getOrCreateStreamJob(cacheKey, safeName, cacheDir, track)

	// Wait until buffer threshold is reached or job completes
	select {
	case <-job.buffered:
	case <-job.done:
	case <-ctx.Done():
		return StreamResult{}, ctx.Err()
	}

	if job.err != nil {
		return StreamResult{}, job.err
	}

	// If full cacheFile already exists, play directly
	if fi, err := os.Stat(cacheFile); err == nil && fi.Size() > 20000 {
		artData, _ = os.ReadFile(artCacheFile)
		return StreamResult{
			Track:    track,
			FilePath: cacheFile,
			Artwork:  artData,
		}, nil
	}

	// Progressive stream from active partial file
	readF, err := os.Open(job.partFile)
	if err == nil {
		pfs := NewProgressiveFileStream(ctx, nil, readF, job.done, &job.err, nil)
		artData, _ = os.ReadFile(artCacheFile)
		return StreamResult{
			Track:   track,
			Stream:  pfs,
			Artwork: artData,
		}, nil
	}

	// Wait for job to finish if opening partial file failed
	select {
	case <-job.done:
		if job.err != nil {
			return StreamResult{}, job.err
		}
		if fi, err := os.Stat(cacheFile); err == nil && fi.Size() > 20000 {
			artData, _ = os.ReadFile(artCacheFile)
			return StreamResult{
				Track:    track,
				FilePath: cacheFile,
				Artwork:  artData,
			}, nil
		}
	case <-ctx.Done():
		return StreamResult{}, ctx.Err()
	}

	return StreamResult{}, fmt.Errorf("stream buffer failed")
}

// ResolveAndCache resolves a track via yt-dlp, saves it to the local cache,
// and concurrently downloads the album artwork in the background.
func ResolveAndCache(ctx context.Context, track OnlineTrack, onStatus func(string)) (string, []byte, error) {
	res, err := ResolveStream(ctx, track, onStatus)
	if err != nil {
		return "", nil, err
	}
	if res.FilePath != "" {
		return res.FilePath, res.Artwork, nil
	}

	// If streaming, wait until completely downloaded to cacheFile
	cacheDir, _ := CacheDir()
	var cacheKey string
	if track.ID != "" {
		cacheKey = sanitize(track.ID)
	} else {
		h := sha256.Sum256([]byte(track.Artist + " - " + track.Title))
		cacheKey = fmt.Sprintf("%x", h[:8])
	}
	safeName := sanitize(track.Artist + " - " + track.Title)
	if len(safeName) > 40 {
		safeName = safeName[:40]
	}
	cacheFile := filepath.Join(cacheDir, fmt.Sprintf("%s_%s.mp3", cacheKey, safeName))

	streamJobsMu.Lock()
	job := streamJobs[cacheKey]
	streamJobsMu.Unlock()

	if job != nil {
		select {
		case <-job.done:
			if job.err != nil {
				return "", res.Artwork, job.err
			}
			return cacheFile, res.Artwork, nil
		case <-ctx.Done():
			return "", res.Artwork, ctx.Err()
		}
	}

	if fi, err := os.Stat(cacheFile); err == nil && fi.Size() > 20000 {
		return cacheFile, res.Artwork, nil
	}

	return "", res.Artwork, fmt.Errorf("cache file not found")
}

// ResolveAndCacheITunes is a backward-compatibility wrapper for ITunesTrack.
func ResolveAndCacheITunes(ctx context.Context, track ITunesTrack, onStatus func(string)) (string, []byte, error) {
	return ResolveAndCache(ctx, track.ToOnlineTrack(), onStatus)
}

// PrebufferTrack resolves and caches a track in the background so it is ready for instant playback.
func PrebufferTrack(ctx context.Context, track OnlineTrack) {
	_, _, _ = ResolveAndCache(ctx, track, nil)
}

// GetCachedTrack checks whether a track is already cached locally and returns its path and artwork.
func GetCachedTrack(track OnlineTrack) (string, []byte, bool) {
	cacheDir, err := CacheDir()
	if err != nil {
		return "", nil, false
	}
	var cacheKey string
	if track.ID != "" {
		cacheKey = sanitize(track.ID)
	} else {
		h := sha256.Sum256([]byte(track.Artist + " - " + track.Title))
		cacheKey = fmt.Sprintf("%x", h[:8])
	}
	safeName := sanitize(track.Artist + " - " + track.Title)
	if len(safeName) > 40 {
		safeName = safeName[:40]
	}
	cacheFile := filepath.Join(cacheDir, fmt.Sprintf("%s_%s.mp3", cacheKey, safeName))
	artCacheFile := filepath.Join(cacheDir, fmt.Sprintf("%s_%s.jpg", cacheKey, safeName))

	if fi, err := os.Stat(cacheFile); err == nil && fi.Size() > 50000 {
		artData, _ := os.ReadFile(artCacheFile)
		return cacheFile, artData, true
	}
	return "", nil, false
}

func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, s)
}
