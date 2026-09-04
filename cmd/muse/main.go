package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/KARTHIKKJ369/Tmusic/internal/audio"
	"github.com/KARTHIKKJ369/Tmusic/internal/config"
	"github.com/KARTHIKKJ369/Tmusic/internal/library"
	"github.com/KARTHIKKJ369/Tmusic/internal/playlist"
	"github.com/KARTHIKKJ369/Tmusic/internal/tui"
)

const version = "0.1.0"

func printHelp() {
	helpText := `
  MUSE // High-Performance Terminal Music Player (v` + version + `)
  Pure Go · FLAC, MP3, WAV, OGG · Smart Shuffle · Album Art

  USAGE:
    muse                      Launch interactive player
    muse dir <path>           Set music directory and index tracks
    muse rescan               Force rescan and refresh library cache
    muse info                 Show configuration and library statistics
    muse play [query]         Launch and immediately play matching/random track
    muse install              Install 'muse' command to ~/.local/bin
    muse help, --help, -h     Show this help guide
    muse version, --version   Print version

  GETTING STARTED:
    1. Set your music folder (only needed once):
       $ muse dir ~/Music

    2. Start playing:
       $ muse

  KEYBOARD SHORTCUTS (IN-APP):
    1, 2, 3, 4      Switch views: Library, Playlists, Favourites, Now Playing
    Tab / Shift+Tab Cycle between sections & playlist panes
    s               Shuffle & play random track / toggle de-clustered shuffle
    Space           Play / Pause
    n / p           Next / Previous track in queue
    → / ←           Seek ±5s
    Shift+→ / ←     Seek ±30s
    0-9             Jump to 0% - 90% in Now Playing
    g or :          Jump to exact time (e.g. 1:30 or 90s or 50%)
    + / -           Volume up / down (5% steps)
    m               Mute / Unmute
    r               Cycle Repeat (Off → Track → Queue)
    f               Toggle favourite heart (♥)
    a               Add selected track to a playlist
    c               Create new playlist
    d / x           Delete playlist or remove track from playlist
    /               Live fuzzy search
    ?               In-app keyboard cheatsheet
    q / Ctrl+C      Save state and quit
`
	fmt.Println(helpText)
}

func main() {
	args := os.Args[1:]

	// Custom flag parsing or subcommand dispatcher
	if len(args) > 0 {
		switch args[0] {
		case "help", "--help", "-h":
			printHelp()
			return
		case "version", "--version", "-v":
			fmt.Println("muse", version)
			return
		case "install":
			handleInstall()
			return
		case "info", "status":
			handleInfo()
			return
		case "rescan", "--rescan":
			handleRescan()
			return
		case "dir", "set-dir", "--set-dir":
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "Error: missing directory path.")
				fmt.Fprintln(os.Stderr, "Usage: muse dir /path/to/music")
				os.Exit(1)
			}
			handleSetDir(args[1])
			return
		case "play":
			query := ""
			if len(args) > 1 {
				query = strings.Join(args[1:], " ")
			}
			launchPlayer(false, query)
			return
		}
	}

	// Flag fallback
	setDirFlag := flag.String("set-dir", "", "Set the music directory and exit")
	rescanFlag := flag.Bool("rescan", false, "Force rescan of the music library")
	verFlag := flag.Bool("version", false, "Print version and exit")
	helpFlag := flag.Bool("help", false, "Print help")
	flag.CommandLine.Init(os.Args[0], flag.ContinueOnError)
	_ = flag.CommandLine.Parse(args)

	if *helpFlag {
		printHelp()
		return
	}
	if *verFlag {
		fmt.Println("muse", version)
		return
	}
	if *setDirFlag != "" {
		handleSetDir(*setDirFlag)
		return
	}

	// Normal launch
	cfg, err := config.Load()
	if err != nil {
		fatalf("config load: %v", err)
	}

	// If no music directory is configured, display the help guide!
	if cfg.MusicDir == "" {
		printHelp()
		fmt.Println("  [!] No music directory configured yet.")
		fmt.Println("      Run: muse dir /path/to/your/music")
		fmt.Println()
		os.Exit(0)
	}

	launchPlayer(*rescanFlag, "")
}

func handleSetDir(dir string) {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		fatalf("path: %v", err)
	}
	stat, err := os.Stat(absPath)
	if err != nil || !stat.IsDir() {
		fatalf("directory '%s' does not exist or is not a folder", absPath)
	}

	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}
	cfg.MusicDir = absPath
	if err := config.Save(cfg); err != nil {
		fatalf("save config: %v", err)
	}

	fmt.Printf("✓ Music directory set to: %s\n", absPath)
	fmt.Println("Scanning library...")

	idx := library.NewIndex()
	_ = idx.Scan(absPath, func(done, total int) {
		fmt.Printf("\r  Indexed %d / %d tracks...", done, total)
	})
	fmt.Println()
	_ = idx.Save()

	fmt.Printf("✓ Success! %d audio tracks indexed.\n", idx.Len())
	fmt.Println("You can now start listening by typing: muse")
}

func handleRescan() {
	cfg, err := config.Load()
	if err != nil || cfg.MusicDir == "" {
		fatalf("no music directory configured. Run: muse dir <path>")
	}
	fmt.Printf("Rescanning %s ...\n", cfg.MusicDir)
	idx := library.NewIndex()
	_ = idx.Scan(cfg.MusicDir, func(done, total int) {
		fmt.Printf("\r  Indexed %d / %d tracks...", done, total)
	})
	fmt.Println()
	_ = idx.Save()
	fmt.Printf("✓ Done! %d tracks indexed and cached.\n", idx.Len())
}

func handleInfo() {
	cfg, err := config.Load()
	if err != nil {
		fatalf("config: %v", err)
	}
	dir, _ := config.Dir()

	idx := library.NewIndex()
	_ = idx.Load()

	pm := playlist.NewManager()
	_ = pm.Load()

	fmt.Println("\nMUSE // CONFIGURATION & STATUS")
	fmt.Println("----------------------------------------")
	fmt.Printf("  Version:        %s\n", version)
	fmt.Printf("  Config Dir:     %s\n", dir)
	fmt.Printf("  Music Folder:   %s\n", cfg.MusicDir)
	fmt.Printf("  Indexed Tracks: %d\n", idx.Len())
	fmt.Printf("  Playlists:      %d\n", len(pm.Playlists))
	fmt.Printf("  Favourites:     %d\n", len(pm.Favourites))
	fmt.Printf("  Default Volume: %d%%\n", int(cfg.Volume*100))
	fmt.Printf("  Repeat Mode:    %s\n", cfg.Repeat)
	fmt.Printf("  Shuffle Mode:   %v\n", cfg.Shuffle)
	fmt.Println("----------------------------------------")
	fmt.Println()
}

func handleInstall() {
	execPath, err := os.Executable()
	if err != nil {
		fatalf("get executable: %v", err)
	}

	targetDir := filepath.Join(os.Getenv("HOME"), ".local", "bin")
	_ = os.MkdirAll(targetDir, 0o755)
	targetPath := filepath.Join(targetDir, "muse")

	data, err := os.ReadFile(execPath)
	if err != nil {
		fatalf("read binary: %v", err)
	}

	if err := os.WriteFile(targetPath, data, 0o755); err != nil {
		fatalf("write binary to %s: %v", targetPath, err)
	}

	fmt.Printf("✓ Installed 'muse' to %s\n", targetPath)
	fmt.Println("You can now run 'muse' directly from anywhere in your terminal!")
}

func launchPlayer(forceRescan bool, autoPlayQuery string) {
	cfg, err := config.Load()
	if err != nil {
		fatalf("config: %v", err)
	}

	idx := library.NewIndex()
	if !forceRescan {
		_ = idx.Load()
	}

	if idx.Len() == 0 || forceRescan {
		fmt.Printf("Scanning %s ...\n", cfg.MusicDir)
		_ = idx.Scan(cfg.MusicDir, func(done, total int) {
			fmt.Printf("\r  Indexed %d / %d tracks...", done, total)
		})
		fmt.Println()
		_ = idx.Save()
	}

	pm := playlist.NewManager()
	_ = pm.Load()

	player, err := audio.NewPlayer(cfg.Volume)
	if err != nil {
		fatalf("audio init: %v", err)
	}

	model := tui.New(cfg, idx, pm, player)

	// If autoPlayQuery is provided or user asked to play, trigger autoplay
	if autoPlayQuery != "" {
		// find matching track or first random track
		tracks := idx.All()
		var target audio.Track
		q := strings.ToLower(autoPlayQuery)
		for _, t := range tracks {
			if strings.Contains(strings.ToLower(t.DisplayTitle()), q) ||
				strings.Contains(strings.ToLower(t.Artist), q) {
				target = t
				break
			}
		}
		if target.ID == "" && len(tracks) > 0 {
			target = tracks[0]
		}
		if target.ID != "" {
			_ = player.Load(target.Path)
			player.Play()
		}
	}

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fatalf("tui: %v", err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "muse: "+format+"\n", args...)
	os.Exit(1)
}
