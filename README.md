# 🎵 muse

A lightweight, blazing fast, and highly efficient terminal music player built in pure Go.

Supports **FLAC**, **MP3**, **WAV**, and **OGG** with zero runtime dependencies (no `mpv`, `ffmpeg`, or `portaudio` installs required).

---

## ✨ Features

- 🚀 **Run From Anywhere**: Installed to `~/.local/bin/muse` — simply type `muse` in your terminal.
- 🎼 **Multiple Formats**: Native decoding for `.flac`, `.mp3`, `.wav`, `.ogg`.
- 📂 **One-Time Library Setup**: Configure your music folder once (`muse dir <path>`).
- 🔀 **Smart De-clustered Shuffle**: Press **`s`** to instantly shuffle and jump directly to Now Playing.
- ⏭ **Direct Time Jumps & Percent Skips**:
  - Press **`0` - `9`** in Now Playing to jump to 0%, 10%, 20%, ..., 90% of the song.
  - Press **`g`** or **`:`** to jump to an exact timestamp (e.g. `1:30`, `90s`, or `50%`).
  - Press **`→` / `←`** (±5s) or **`Shift+→` / `Shift+←`** (±30s).
- 🖼 **Embedded Album Art & Vinyl Disc**: Extracts embedded JPEG/PNG cover art and renders high-density **ANSI 24-bit TrueColor Half-Blocks (`▀`)** in the center of Now Playing, with procedural animated Vinyl Disc artwork fallback.
- ❤️ **Favourites & Playlists**: Toggle favourites with persistent hearts (`♥`), manage custom playlists with dual-pane browsing.
- 🔍 **Instant Fuzzy Search**: Real-time live filtering across titles, artists, and albums (`/`).

---

## 💻 CLI Commands

```bash
muse                      # Launch interactive player
muse dir <path>           # Set music directory and index tracks
muse rescan               # Force rescan and refresh library cache
muse info                 # Show configuration and library statistics
muse play [query]         # Launch and immediately play matching/random track
muse install              # Re-install 'muse' command to ~/.local/bin
muse help, --help, -h     # Show CLI help guide
muse version              # Print version
```

---

## ⌨️ Complete In-App Keyboard Reference

### Navigation & Views
| Key | Action |
|---|---|
| `1` | Jump to **Library** |
| `2` | Jump to **Playlists** |
| `3` | Jump to **Favourites** |
| `4` | Jump to **Now Playing** |
| `Tab` / `Shift+Tab` | Cycle through tabs / toggle playlist panes |
| `↑` / `↓` (or `j` / `k`) | Navigate lists |
| `←` / `→` (or `h` / `l`) | Switch between Playlists (left) and Songs (right) in Playlists tab |

### Playback & Time Jumps
| Key | Action |
|---|---|
| `s` | Shuffle library and start playing immediately (moves to Now Playing) / toggle shuffle |
| `S` or `z` | Force pick a new random track and start playing in Now Playing |
| `Space` | Play / Pause |
| `Enter` | Play selected track / playlist |
| `n` / `p` | Next / Previous track in queue |
| `0` - `9` | **Jump to 0% - 90%** of track in Now Playing |
| `g` or `:` | **Jump to exact time** (prompt: `1:30`, `90s`, `50%`) |
| `→` / `←` | Seek ±5 seconds |
| `Shift+→` / `Shift+←` | Seek ±30 seconds (or `H` / `L`) |
| `+` / `-` | Volume up / down (5% increments) |
| `m` | Mute / Unmute |
| `r` | Cycle Repeat mode (`Off` → `Track` → `Queue`) |

### Playlists, Favourites & Search
| Key | Action |
|---|---|
| `f` | Toggle favourite heart (♥) on selected / currently playing track |
| `a` | Open playlist picker to add selected track to any playlist |
| `c` | Create a new playlist (opens naming modal) |
| `d` / `x` | Delete playlist or remove track from playlist |
| `/` | Open live fuzzy search filter |
| `?` | Toggle keyboard help guide modal |
| `q` / `Ctrl+C` | Save state and quit |
