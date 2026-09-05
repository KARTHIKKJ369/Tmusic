package online

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

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

// ResolveAndCache resolves a track via yt-dlp, saves it to the local cache,
// and concurrently downloads the album artwork in the background.
func ResolveAndCache(ctx context.Context, track OnlineTrack, onStatus func(string)) (string, []byte, error) {
	cacheDir, err := CacheDir()
	if err != nil {
		return "", nil, fmt.Errorf("get cache directory: %w", err)
	}

	// Deterministic filename based on track ID or title/artist hash
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

	// Non-blocking artwork fetch in background
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
		return cacheFile, artData, nil
	}

	// Check dependencies
	if err := CheckDependencies(); err != nil {
		return "", artData, err
	}

	if onStatus != nil {
		onStatus("Buffering audio stream...")
	}

	// Fast path: If track has a direct YouTube video ID, extract stream URL and pipe directly through ffmpeg
	if track.Source == "youtube" && len(track.ID) >= 8 && !strings.HasPrefix(track.ID, "itunes_") {
		videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", track.ID)
		if _, ffErr := exec.LookPath("ffmpeg"); ffErr == nil {
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
						cacheFile,
					)
					if err := cmdFF.Run(); err == nil {
						if fi, err := os.Stat(cacheFile); err == nil && fi.Size() > 50000 {
							return cacheFile, artData, nil
						}
					}
				}
			}
		}
	}

	// Fallback path: full yt-dlp extraction
	tempTarget := filepath.Join(cacheDir, fmt.Sprintf("tmp_%s.%%(ext)s", cacheKey))
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
	output, err := cmd.CombinedOutput()
	if err != nil {
		if track.Source != "youtube" {
			fallbackQuery := fmt.Sprintf("%s %s", track.Artist, track.Title)
			args[len(args)-1] = fallbackQuery
			cmdRetry := exec.CommandContext(ctx, "yt-dlp", args...)
			output, err = cmdRetry.CombinedOutput()
		}
		if err != nil {
			return "", artData, fmt.Errorf("audio resolution failed: %w (%s)", err, strings.TrimSpace(string(output)))
		}
	}

	// Locate the generated mp3 file
	expectedTempMp3 := filepath.Join(cacheDir, fmt.Sprintf("tmp_%s.mp3", cacheKey))
	if _, err := os.Stat(expectedTempMp3); err == nil {
		if err := os.Rename(expectedTempMp3, cacheFile); err == nil {
			return cacheFile, artData, nil
		}
	}

	files, _ := filepath.Glob(filepath.Join(cacheDir, fmt.Sprintf("tmp_%s.*", cacheKey)))
	for _, f := range files {
		if strings.HasSuffix(f, ".mp3") {
			_ = os.Rename(f, cacheFile)
			return cacheFile, artData, nil
		}
	}

	if _, err := os.Stat(cacheFile); err == nil {
		return cacheFile, artData, nil
	}

	return "", artData, fmt.Errorf("stream downloaded but output file not found in cache")
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
