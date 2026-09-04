// Package updater handles automatic self-updates from GitHub Releases across macOS, Linux, and Windows.
package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	repoOwner = "KARTHIKKJ369"
	repoName  = "Tmusic"
	apiURL    = "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases/latest"
)

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Name    string        `json:"name"`
	Body    string        `json:"body"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// CheckAndUpdate checks for the latest version and updates the binary if available.
func CheckAndUpdate(currentVersion string, force bool) error {
	fmt.Println("==> Checking for updates...")

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "muse-updater/"+currentVersion)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connect to GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return fmt.Errorf("decode release metadata: %w", err)
	}

	latestVer := strings.TrimPrefix(rel.TagName, "v")
	currVer := strings.TrimPrefix(currentVersion, "v")

	if latestVer == currVer && !force {
		fmt.Printf("✓ muse is already up to date (%s)\n", currentVersion)
		return nil
	}

	fmt.Printf("==> Updating muse: %s → %s\n", currentVersion, rel.TagName)

	// Find matching asset
	assetURL, assetName, err := findMatchingAsset(rel.Assets)
	if err != nil {
		return fmt.Errorf("no compatible release binary found for %s/%s: %w", runtime.GOOS, runtime.GOARCH, err)
	}

	fmt.Printf("==> Downloading %s...\n", assetName)
	data, err := downloadAsset(client, assetURL)
	if err != nil {
		return fmt.Errorf("download binary: %w", err)
	}

	// Extract binary from archive
	binData, err := extractBinary(data, assetName)
	if err != nil {
		return fmt.Errorf("extract binary from %s: %w", assetName, err)
	}

	// Install binary
	targetPaths := determineTargetPaths()
	var installedPath string
	for _, target := range targetPaths {
		if err := installBinary(binData, target); err == nil {
			installedPath = target
			break
		}
	}

	if installedPath == "" {
		return fmt.Errorf("failed to write binary to target paths: %v", targetPaths)
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  ✓ Successfully updated muse to %s!\n", rel.TagName)
	fmt.Printf("    Installed to: %s\n", installedPath)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	return nil
}

func findMatchingAsset(assets []githubAsset) (url string, name string, err error) {
	osName := runtime.GOOS
	archName := runtime.GOARCH

	// Map Go runtime names to GoReleaser naming patterns
	var osPatterns []string
	switch osName {
	case "darwin":
		osPatterns = []string{"Darwin", "darwin"}
	case "linux":
		osPatterns = []string{"Linux", "linux"}
	case "windows":
		osPatterns = []string{"Windows", "windows"}
	default:
		osPatterns = []string{osName}
	}

	var archPatterns []string
	switch archName {
	case "amd64":
		archPatterns = []string{"x86_64", "amd64"}
	case "arm64":
		archPatterns = []string{"arm64", "aarch64"}
	case "386":
		archPatterns = []string{"i386", "386"}
	default:
		archPatterns = []string{archName}
	}

	for _, a := range assets {
		aLower := strings.ToLower(a.Name)
		for _, osp := range osPatterns {
			if strings.Contains(aLower, strings.ToLower(osp)) {
				for _, ap := range archPatterns {
					if strings.Contains(aLower, strings.ToLower(ap)) {
						return a.BrowserDownloadURL, a.Name, nil
					}
				}
			}
		}
	}

	return "", "", fmt.Errorf("no asset matching OS %v and Arch %v", osPatterns, archPatterns)
}

func downloadAsset(client *http.Client, url string) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func extractBinary(data []byte, filename string) ([]byte, error) {
	switch {
	case strings.HasSuffix(filename, ".tar.gz") || strings.HasSuffix(filename, ".tgz"):
		gzr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		defer gzr.Close()

		tr := tar.NewReader(gzr)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			name := filepath.Base(hdr.Name)
			if name == "muse" || name == "muse.exe" {
				return io.ReadAll(tr)
			}
		}
		return nil, fmt.Errorf("binary 'muse' not found in archive")

	case strings.HasSuffix(filename, ".zip"):
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, err
		}
		for _, f := range zr.File {
			name := filepath.Base(f.Name)
			if name == "muse" || name == "muse.exe" {
				rc, err := f.Open()
				if err != nil {
					return nil, err
				}
				defer rc.Close()
				return io.ReadAll(rc)
			}
		}
		return nil, fmt.Errorf("binary 'muse' not found in zip")

	default:
		// Direct binary
		return data, nil
	}
}

func determineTargetPaths() []string {
	var paths []string

	// Current executable path if writable
	if execPath, err := os.Executable(); err == nil {
		execPath, _ = filepath.EvalSymlinks(execPath)
		if execPath != "" {
			paths = append(paths, execPath)
		}
	}

	if runtime.GOOS == "windows" {
		localApp := os.Getenv("LOCALAPPDATA")
		if localApp != "" {
			paths = append(paths, filepath.Join(localApp, "Programs", "muse", "muse.exe"))
		}
		userProfile := os.Getenv("USERPROFILE")
		if userProfile != "" {
			paths = append(paths, filepath.Join(userProfile, "bin", "muse.exe"))
		}
	} else {
		home, _ := os.UserHomeDir()
		if home != "" {
			paths = append(paths, filepath.Join(home, ".local", "bin", "muse"))
		}
		paths = append(paths, "/usr/local/bin/muse")
	}

	return paths
}

func installBinary(data []byte, targetPath string) error {
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// On Windows, if the file is currently executing, rename it to .old first
	if runtime.GOOS == "windows" {
		oldFile := targetPath + ".old"
		_ = os.Remove(oldFile)
		if _, err := os.Stat(targetPath); err == nil {
			_ = os.Rename(targetPath, oldFile)
		}
	}

	// Write to temporary file in the same directory, then atomic rename
	tmpFile := targetPath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0o755); err != nil {
		return err
	}

	if err := os.Rename(tmpFile, targetPath); err != nil {
		_ = os.Remove(tmpFile)
		// Fallback to direct write if rename fails
		if writeErr := os.WriteFile(targetPath, data, 0o755); writeErr != nil {
			return err
		}
	}

	return os.Chmod(targetPath, 0o755)
}
