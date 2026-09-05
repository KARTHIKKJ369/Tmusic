# 🎵 muse

A lightweight, blazing fast, and highly efficient terminal music player built in pure Go.

Supports **FLAC**, **MP3**, **WAV**, and **OGG** with zero external runtime dependencies on **Windows**, **macOS**, and **Linux**.

---

## ⚡ 1-Line Universal Installation

### 🪟 Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/KARTHIKKJ369/Tmusic/main/install.ps1 | iex
```

### 🍏 macOS (Apple Silicon & Intel) & 🐧 Linux
```bash
curl -fsSL https://raw.githubusercontent.com/KARTHIKKJ369/Tmusic/main/install.sh | bash
```

### 🔄 In-Place Self-Updating
```bash
muse -u
# or
muse update
```

### 🛠️ Via Go (All Platforms)
```bash
go install github.com/KARTHIKKJ369/Tmusic/cmd/muse@latest
```

---

## ✨ Features

- 🚀 **Run From Anywhere**: Installed to PATH — simply type `muse` anywhere in your terminal.
- 🌐 **Keyless Online Streaming**: Stream millions of tracks instantly via YouTube Music & iTunes with zero API keys required (`muse --online [query]`).
- 📻 **Smart Contextual Radio Automix**: Automatically queues genuine related recommendations (by artist, era, mood, and genre) in the background while you listen.
- 💾 **1-Key Download to Library (`d`)**: Press **`d`** in Now Playing or Online Search to permanently save any online song to your local music directory.
- 🎼 **Native Codecs**: Pure Go decoders for `.flac`, `.mp3`, `.wav`, `.ogg`.
- 📂 **One-Time Library Setup**: Configure your music folder once (`muse dir <path>`).
- 🔀 **Smart De-clustered Shuffle**: Press **`s`** to shuffle and de-cluster artist tracks without interrupting playback.
- ⏭ **Direct Time Jumps & Percent Skips**:
  - Press **`0` - `9`** in Now Playing to jump to 0%, 10%, 20%, ..., 90% of the song.
  - Press **`g`** or **`:`** to jump to an exact timestamp (e.g. `1:30`, `90s`, or `50%`).
  - Press **`→` / `←`** (±5s) or **`Shift+→` / `Shift+←`** (±30s).
- 🖼 **HD ANSI TrueColor Album Art**: High-definition **24-bit TrueColor Half-Blocks (`▀`)** centered above song metadata, with animated spectrum visualizer and instant **`o`** photo viewer.
- ❤️ **Favourites & Playlists**: Toggle favourites with persistent hearts (`♥`), manage custom playlists with dual-pane browsing.
- 🔍 **Live Search & Autocomplete**: Real-time autocomplete suggestions with full word deletion (`Option+Backspace`, `Ctrl+W`, `Ctrl+U`).

---

## 💻 CLI Commands

```bash
muse                       # Launch interactive player (resumes last session)
muse --online [query]      # Launch directly into online search & streaming mode
muse play [query]          # Launch and immediately play matching/random track
muse dir <path>            # Set music directory and index tracks
muse rescan                # Force rescan and refresh library cache
muse info                  # Show configuration and library statistics
muse update, -u, --update  # Check and install latest release
muse install               # Re-install 'muse' binary to user bin directory
muse help, --help, -h      # Show CLI help guide
muse version               # Print version
```

---

## ⌨️ Complete In-App Keyboard Reference

### Navigation & Views
| Key | Action |
|---|---|
| `1` | Jump to **Library** |
| `2` | Jump to **Playlists** |
| `3` | Jump to **Favourites** |
| `4` | Jump to **Online Search & Streaming** |
| `5` | Jump to **Now Playing** |
| `Tab` / `Shift+Tab` | Cycle through tabs / toggle playlist panes |
| `↑` / `↓` (or `j` / `k`) | Navigate song & playlist lists |
| `←` / `→` (or `h` / `l`) | Switch between Playlists (left) and Songs (right) in Playlists tab |

### Playback & Time Jumps
| Key | Action |
|---|---|
| `Space` | Play / Pause |
| `Enter` | Play selected track or playlist (or stream online track) |
| `d` | **Download online track to library** (works in Now Playing and Online view) |
| `s` | Shuffle queue / random track (in Online: play random search result) |
| `S` or `z` | Force pick a new random track from library |
| `n` / `p` | Next / Previous track in queue |
| `0` - `9` | **Jump to 0% - 90%** of track in Now Playing |
| `g` or `:` | **Jump to exact time** (prompt: `1:30`, `90s`, `50%`) |
| `→` / `←` | Seek ±5 seconds |
| `Shift+→` / `Shift+←` | Seek ±30 seconds (or `H` / `L`) |
| `o` | Open high-resolution album cover art in system photo viewer |
| `+` / `-` | Volume up / down (5% increments) |
| `m` | Mute / Unmute |
| `r` | Cycle Repeat mode (`Off` → `Track` → `Queue`) |

### Search, Playlists & Input Editing
| Key | Action |
|---|---|
| `/` | Focus search bar (Online search in Online view, fuzzy library filter in local views) |
| `Option+Backspace` / `Ctrl+W` | Delete word backwards in all search and modal input fields |
| `Ctrl+U` | Clear entire input line |
| `f` | Toggle favourite heart (♥) on selected / currently playing track |
| `a` | Open playlist picker to add selected track to any playlist |
| `c` | Create a new playlist (opens naming modal) |
| `d` / `x` | Delete playlist or remove track from playlist (in Playlists tab) |
| `?` | Toggle keyboard help guide modal |
| `q` / `Ctrl+C` | Save state and quit |

