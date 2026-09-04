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
	"sync"
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
// and concurrently downloads the 600x600 album artwork.
func ResolveAndCache(ctx context.Context, track ITunesTrack, onStatus func(string)) (string, []byte, error) {
	cacheDir, err := CacheDir()
	if err != nil {
		return "", nil, fmt.Errorf("get cache directory: %w", err)
	}

	// Deterministic filename based on track ID or title/artist hash
	var cacheKey string
	if track.TrackID > 0 {
		cacheKey = fmt.Sprintf("%d", track.TrackID)
	} else {
		h := sha256.Sum256([]byte(track.ArtistName + " - " + track.TrackName))
		cacheKey = fmt.Sprintf("%x", h[:8])
	}
	safeName := sanitize(track.ArtistName + " - " + track.TrackName)
	if len(safeName) > 40 {
		safeName = safeName[:40]
	}
	cacheFile := filepath.Join(cacheDir, fmt.Sprintf("%s_%s.mp3", cacheKey, safeName))
	artCacheFile := filepath.Join(cacheDir, fmt.Sprintf("%s_%s.jpg", cacheKey, safeName))

	var (
		artData []byte
		wg      sync.WaitGroup
	)

	// Concurrently fetch artwork
	wg.Add(1)
	go func() {
		defer wg.Done()
		if data, err := os.ReadFile(artCacheFile); err == nil && len(data) > 0 {
			artData = data
			return
		}
		artURL := track.HighResArtworkURL()
		if artURL != "" {
			if data, err := FetchArtwork(artURL); err == nil && len(data) > 0 {
				artData = data
				_ = os.WriteFile(artCacheFile, data, 0o644)
			}
		}
	}()

	// If audio is already in cache and valid, return immediately
	if fi, err := os.Stat(cacheFile); err == nil && fi.Size() > 50000 {
		if onStatus != nil {
			onStatus("Loaded from local cache")
		}
		wg.Wait()
		return cacheFile, artData, nil
	}

	// Check yt-dlp dependency
	if err := CheckDependencies(); err != nil {
		wg.Wait()
		return "", artData, err
	}

	if onStatus != nil {
		onStatus("Resolving stream via yt-dlp...")
	}

	// Build search query: prioritize "Official Audio" to skip music video intros
	query := fmt.Sprintf("%s %s Official Audio", track.ArtistName, track.TrackName)

	// Execute yt-dlp to download and convert to MP3 directly
	tempTarget := filepath.Join(cacheDir, fmt.Sprintf("tmp_%s.%%(ext)s", cacheKey))
	cmd := exec.CommandContext(ctx, "yt-dlp",
		"--extractor-args", "youtube:player_client=android",
		"-f", "ba/140/18/best",
		"-x",
		"--audio-format", "mp3",
		"--audio-quality", "0",
		"-o", tempTarget,
		"--no-playlist",
		"--default-search", "ytsearch1:",
		query,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback: try search without "Official Audio"
		if onStatus != nil {
			onStatus("Retrying stream resolution...")
		}
		fallbackQuery := fmt.Sprintf("%s %s", track.ArtistName, track.TrackName)
		cmd2 := exec.CommandContext(ctx, "yt-dlp",
			"--extractor-args", "youtube:player_client=android",
			"-f", "ba/140/18/best",
			"-x",
			"--audio-format", "mp3",
			"--audio-quality", "0",
			"-o", tempTarget,
			"--no-playlist",
			"--default-search", "ytsearch1:",
			fallbackQuery,
		)
		output2, err2 := cmd2.CombinedOutput()
		if err2 != nil {
			wg.Wait()
			return "", artData, fmt.Errorf("yt-dlp resolution failed: %w (output: %s)", err2, strings.TrimSpace(string(output2)))
		}
	}

	_ = output

	// Locate the generated mp3 file
	expectedTempMp3 := filepath.Join(cacheDir, fmt.Sprintf("tmp_%s.mp3", cacheKey))
	if _, err := os.Stat(expectedTempMp3); err == nil {
		if err := os.Rename(expectedTempMp3, cacheFile); err == nil {
			wg.Wait()
			return cacheFile, artData, nil
		}
		expectedTempMp3 = cacheFile
	}

	// Fallback scan of cache dir for matching file
	files, _ := filepath.Glob(filepath.Join(cacheDir, fmt.Sprintf("tmp_%s.*", cacheKey)))
	for _, f := range files {
		if strings.HasSuffix(f, ".mp3") {
			_ = os.Rename(f, cacheFile)
			wg.Wait()
			return cacheFile, artData, nil
		}
	}

	wg.Wait()
	if _, err := os.Stat(cacheFile); err == nil {
		return cacheFile, artData, nil
	}

	return "", artData, fmt.Errorf("stream downloaded but output file not found in cache")
}

func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, s)
}
