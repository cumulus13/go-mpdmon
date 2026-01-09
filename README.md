# MPD Monitor with GNTP Notifications (Go)

Monitor MPD (Music Player Daemon) which sends notifications to Growl/GNTP every time there is a change in music playback status using **github.com/cumulus13/go-gntp** - the most advanced GNTP library for Go with full callback support and cross-platform compatibility!

## Features

✅ **Monitor MPD status changes:**
- Song change detection
- Detection of state changes (play, pause, stop)
- Skip database updates to avoid race conditions

✅ **GNTP/Growl notification with github.com/cumulus13/go-gntp:**
- Full GNTP 1.0 protocol implementation
- Multiple icon delivery modes (Binary, DataURL, FileURL, HttpURL)
- Cover art/album artwork as icon (embedded or external)
- Windows Growl compatibility with automatic workarounds
- Android Growl compatibility (tested!)
- Complete information: position/total/track, title, artist, album, bitrate, filepath
- Different colors for each type of information

✅ **Display runtime on console:**
- Neat current playing display with emojis
- Realtime information in an easy-to-read format

✅ **Flexible configuration:**
- Environment variables (MPD_HOST, MPD_PORT, MPD_TIMEOUT)
- Command line arguments
- File konfigurasi TOML
- Priority: CLI args > Env vars > Config file > Defaults

## Color Information

- **#00FFFF (Cyan)**: Position/Total/Track & Title, Time
- **#FFFF00 (Yellow)**: Artist
- **#FFAA7F (Orange)**: Album
- **#AAAAFF (Blue)**: Bitrate/Sample Rate
- **#00AA00 (Green)**: Filepath

## Icon Delivery Modes

The `go-gntp` library supports 4 icon delivery modes:

### 1. Binary Mode (Default - Recommended!)
```toml
icon_mode = "binary"
```
- Format: `x-growl-resource://UUID` + binary data
- **✅ Tested dan working di Windows Growl**
- Most reliable for Windows, macOS, Linux
- Spec-compliant GNTP 1.0

### 2. DataURL Mode
```toml
icon_mode = "dataurl"
```
- Format: `data:image/png;base64,iVBORw0KGgo...`
- Great for Android Growl
- ⚠️ There may be issues with large icons in Windows

### 3. FileURL Mode
```toml
icon_mode = "fileurl"
```
- Format: `file:///C:/full/path/to/icon.png`
- Untuk shared icons di disk
- ⚠️ Need absolute path, file must be accessible by Growl

### 4. Http URL Mode
```toml
icon_mode = "httpurl"
```
- Format: `http://example.com/icon.png`
- For-hosted icons, remote servers

## Installation

### Prerequisites

1. **Go 1.21+**
2. **MPD** (Music Player Daemon) which is already running
3. **Growl for Windows** or **GNTP-compatible client**

### Build

```bash
# Clone or download source code
# Then:

go mod download
go build -o mpd-monitor main.go
```

## Usage

### 1. Using Defaults

```bash
./mpd-monitor
```

Default values:
- MPD: localhost:6600
- GNTP: localhost:23053
- Timeout: 10 seconds
- Icon Mode: binary (recommended!)

### 2. Using Environment Variables

```bash
export MPD_HOST=192.168.1.100
export MPD_PORT=6600
export MPD_TIMEOUT=15

./mpd-monitor
```

### 3. Using Command Line Arguments

```bash
./mpd-monitor \
  -mpd-host 192.168.1.100 \
  -mpd-port 6600 \
  -mpd-timeout 15 \
  -gntp-host localhost \
  -gntp-port 23053 \
  -gntp-password mypassword \
  -icon-mode binary
```

### 4. Using Config Files

Create file `config.toml`:

```toml
[mpd]
host = "192.168.1.100"
port = "6600"
timeout = 15

[gntp]
host = "localhost"
port = 23053
password = ""
icon_mode = "binary"  # binary, dataurl, fileurl, httpurl
```

Run with:

```bash
./mpd-monitor -config config.toml
```

### 5. Remote Android Growl

```bash
./mpd-monitor \
  -gntp-host 192.168.1.50 \
  -icon-mode dataurl
```

Or in config.toml:
```toml
[gntp]
host="192.168.1.50"
port = 23053
icon_mode = "dataurl" # good dataurl or binary for Android
```

### 6. (Priority: CLI > ENV > Config > Default)

```bash
export MPD_HOST=192.168.1.100

./mpd-monitor \
  -config config.toml \
  -mpd-port 6601 \
  -icon-mode binary
```

## Command Line Flags

| Flag | Description | Default |
|------|-----------|---------|
| `-config` | Path to file config TOML | - |
| `-mpd-host` | MPD server host | localhost |
| `-mpd-port` | MPD server port | 6600 |
| `-mpd-timeout` | Connection timeout (detik) | 10 |
| `-gntp-host` | GNTP/Growl server host | localhost |
| `-gntp-port` | GNTP/Growl server port | 23053 |
| `-gntp-password` | GNTP/Growl password | - |
| `-icon-mode` | Icon mode: binary/dataurl/fileurl/httpurl | binary |

## Environment Variables

- `MPD_HOST`: MPD server host
- `MPD_PORT`: MPD server port
- `MPD_TIMEOUT`: Connection timeout dalam detik

## Output Console

```
🎵 MPD Monitor started
📡 Monitoring: localhost:6600
📢 GNTP Server: localhost:23053
✅ GNTP registered (icon mode: binary)
============================================================

▶ 5/12/3. Song Title
  ⏱  2:15 / 3:42
  🎤 Artist Name
  💿 Album Name
  🎵 44 kHz
  📁 music/artist/album/03-song.mp3
────────────────────────────────────────────────────────────
```

## Notifikasi GNTP
Every song or state change will send a notification with:
- **Title**: Song title or status
- **Message**: Complete information with HTML format and color
- **Icon**: Cover art of the album (if available) using the selected mode

## Platform Compatibility

| Platform | Binary | DataURL | FileURL | Recommended |
|----------|--------|---------|---------|-------------|
| **Windows (Growl for Windows)** | ✅ Works! | ⚠️ Issues | ⚠️ Issues | **Binary** |
| **macOS (Growl)** | ✅ Works | ✅ Works | ✅ Works | Binary |
| **Linux (Growl-compatible)** | ✅ Works | ✅ Works | ✅ Works | Binary |
| **Android (Growl for Android)** | ✅ Works | ✅ Works | ⚠️ Issues | **Binary/DataURL** |

**Testing Results from go-gntp:**
- ✅ **Binary Mode**: Confirmed working on Windows Growl
- ⚠️ **DataURL Mode**: May fail with large icons (base64 size limit)
- ⚠️ **FileURL Mode**: Requires absolute path, may have permission issues

## Troubleshooting

### MPD Connection Failed

```bash
# Check MPD is running
systemctl status mpd

# Check MPD network settings in /etc/mpd.conf
bind_to_address "0.0.0.0"  #or specific IP
port "6600"
```

### GNTP Registration Failed

```bash
# Check Growl/GNTP client is running
# Check the firewall is not blocking port 23053
# Make sure the password is correct if any
```

### Icon Doesn't Appear

Try different icon modes:

```bash
# Try binary mode (most reliable)
./mpd-monitor -icon-mode binary

# Or dataurl mode (for Android)
./mpd-monitor -icon-mode dataurl
```

or edit config.toml:
```toml
[gntp]
icon_mode = "binary"  # or "dataurl"
```

### No Album Art

MPD needs to be configured to save/access cover art:
- Embedded artwork in music files (MP3 ID3 tags, FLAC, etc)
- External artwork (cover.jpg, folder.jpg in album folder)

### Android Connection Issues

```bash
# Use dataurl or binary mode for Android
./mpd-monitor \
  -gntp-host 192.168.1.50 \
  -icon-mode dataurl \
  -mpd-timeout 15
```

or config.toml:
```toml
[mpd]
timeout = 15  # Longer timeout for mobile

[gntp]
host = "192.168.1.50"
icon_mode = "dataurl"  # or "binary"
```

## Advantages github.com/cumulus13/go-gntp

The library used has the following advantages:

✨ **Full GNTP 1.0 protocol implementation**
✨ **Callback support** (click, close, timeout events) - ready for development
✨ **Multiple icon delivery modes** with auto-detection
✨ **Windows Growl compatibility** with automatic workarounds
✨ **Android Growl tested and working**
✨ **Cross-platform** (Windows, macOS, Linux, Android)
✨ **Resource deduplication** to prevent errors
✨ **Zero external dependencies** (except uuid)

## Dependencies

- [github.com/cumulus13/go-gntp](https://github.com/cumulus13/go-gntp) - Advanced GNTP client dengan full features
- [github.com/fhs/gompd/v2](https://github.com/fhs/gompd) - MPD client library
- [github.com/BurntSushi/toml](https://github.com/BurntSushi/toml) - TOML parser

## Notes

- The program will skip notifications for `database` and `update` events to avoid race conditions
- Cover art is sent using the selected mode (binary/dataurl/fileurl/httpurl)
- Binary mode is the default and most reliable for all platforms
- Bitrate is displayed from the audio format or bitrate field MPD
- Support image formats: JPEG, PNG (auto-detected from magic bytes)

## License

MIT License

---

**Powered by [github.com/cumulus13/go-gntp](https://github.com/cumulus13/go-gntp)** - The most advanced GNTP library for Go! 🚀