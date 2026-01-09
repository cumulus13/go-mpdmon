package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/cumulus13/go-gntp"
	"github.com/fhs/gompd/v2/mpd"
	"golang.org/x/term"
)

const (
	// ANSI color codes for terminal
	colorReset  = "\033[0m"
	colorCyan   = "\033[96m"  // track/title
	colorYellow = "\033[93m"  // artist
	colorOrange = "\033[38;5;216m" // album
	colorBlue   = "\033[94m"  // bitrate
	colorGreen  = "\033[92m"  // filepath
)

type Config struct {
	MPD struct {
		Host    string `toml:"host"`
		Port    string `toml:"port"`
		Timeout int    `toml:"timeout"`
	} `toml:"mpd"`

	GNTP struct {
		Host     string `toml:"host"`
		Port     int    `toml:"port"`
		Password string `toml:"password"`
		IconMode string `toml:"icon_mode"` // binary, dataurl, fileurl, httpurl
	} `toml:"gntp"`
}

type AppState struct {
	lastSongFile string
	lastState    string
	conn         *mpd.Client
	gntp         *gntp.Client
	config       Config
	debug        bool
	gntpEnabled  bool
}

func loadConfig(configPath string) (Config, error) {
	var cfg Config

	// Default values
	cfg.MPD.Host = "localhost"
	cfg.MPD.Port = "6600"
	cfg.MPD.Timeout = 10
	cfg.GNTP.Host = "localhost"
	cfg.GNTP.Port = 23053
	cfg.GNTP.Password = ""
	cfg.GNTP.IconMode = "binary" // binary mode recommended for Windows

	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
				return cfg, fmt.Errorf("failed to parse config: %v", err)
			}
		}
	}

	return cfg, nil
}

func getEnvOrDefault(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func getTerminalWidth() int {
	// Try to get terminal size
	fd := int(os.Stdout.Fd())
	width, _, err := term.GetSize(fd)
	if err != nil || width <= 0 {
		return 80 // Default fallback
	}
	return width
}

func printSeparator() {
	width := getTerminalWidth()
	fmt.Println(strings.Repeat("─", width))
}

func connectMPD(host, port string, timeout int) (*mpd.Client, error) {
	addr := fmt.Sprintf("%s:%s", host, port)

	client, err := mpd.DialAuthenticated("tcp", addr, "")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MPD at %s: %v", addr, err)
	}

	return client, nil
}

func setupGNTP(cfg Config, debug bool) (*gntp.Client, bool) {
	client := gntp.NewClient("MPD Monitor").
		WithHost(cfg.GNTP.Host).
		WithPort(cfg.GNTP.Port).
		WithTimeout(10 * time.Second)

	// Set icon mode based on config
	switch strings.ToLower(cfg.GNTP.IconMode) {
	case "dataurl":
		client.WithIconMode(gntp.IconModeDataURL)
	case "fileurl":
		client.WithIconMode(gntp.IconModeFileURL)
	case "httpurl":
		client.WithIconMode(gntp.IconModeHttpURL)
	default:
		// Binary mode is default and recommended for Windows
		client.WithIconMode(gntp.IconModeBinary)
	}

	// Define notification types
	songChange := gntp.NewNotificationType("song_change").
		WithDisplayName("Song Changed")

	playerState := gntp.NewNotificationType("player_state").
		WithDisplayName("Player State")

	// Register notifications
	if err := client.Register([]*gntp.NotificationType{songChange, playerState}); err != nil {
		if debug {
			log.Printf("⚠️  Failed to register with GNTP: %v", err)
		}
		log.Println("⚠️  GNTP/Growl not available - notifications disabled")
		return nil, false
	}

	return client, true
}

func getAlbumArt(conn *mpd.Client, uri string) *gntp.Resource {
	// Try ReadPicture first (embedded artwork)
	artwork, err := conn.ReadPicture(uri)
	if err == nil && len(artwork) > 0 {
		// Detect content type
		contentType := "image/jpeg"
		if len(artwork) > 8 {
			if artwork[0] == 0x89 && artwork[1] == 0x50 && artwork[2] == 0x4E && artwork[3] == 0x47 {
				contentType = "image/png"
			}
		}
		return gntp.LoadResourceFromBytes(artwork, contentType)
	}

	// Try AlbumArt (external artwork)
	artwork, err = conn.AlbumArt(uri)
	if err == nil && len(artwork) > 0 {
		contentType := "image/jpeg"
		if len(artwork) > 8 {
			if artwork[0] == 0x89 && artwork[1] == 0x50 && artwork[2] == 0x4E && artwork[3] == 0x47 {
				contentType = "image/png"
			}
		}
		return gntp.LoadResourceFromBytes(artwork, contentType)
	}

	return nil
}

func formatBitrate(attrs mpd.Attrs) string {
	if bitrate, ok := attrs["audio"]; ok {
		// audio format: "samplerate:bits:channels"
		parts := strings.Split(bitrate, ":")
		if len(parts) >= 1 {
			sampleRate := parts[0]
			if sr, err := strconv.Atoi(sampleRate); err == nil {
				kbps := sr / 1000
				return fmt.Sprintf("%d kHz", kbps)
			}
		}
	}

	// Fallback to bitrate field if available
	if bitrate, ok := attrs["bitrate"]; ok {
		return fmt.Sprintf("%s kbps", bitrate)
	}

	return "N/A"
}

func formatDuration(seconds string) string {
	if seconds == "" {
		return "0:00"
	}

	sec, err := strconv.ParseFloat(seconds, 64)
	if err != nil {
		return "0:00"
	}

	mins := int(sec) / 60
	secs := int(sec) % 60

	return fmt.Sprintf("%d:%02d", mins, secs)
}

func formatCurrentPlaying(song mpd.Attrs, status mpd.Attrs) string {
	pos := status["song"]
	total := status["playlistlength"]
	elapsed := formatDuration(status["elapsed"])
	duration := formatDuration(song["duration"])
	track := song["Track"]
	title := song["Title"]
	artist := song["Artist"]
	album := song["Album"]
	bitrate := formatBitrate(status)
	filepath := song["file"]

	if title == "" {
		title = filepath
	}

	if track == "" {
		track = "?"
	}

	var sb strings.Builder

	// Position/Total/Track. Title with time
	sb.WriteString(fmt.Sprintf("%s/%s/%s. %s\n", pos, total, track, title))
	sb.WriteString(fmt.Sprintf("%s / %s\n", elapsed, duration))

	// Artist
	if artist != "" {
		sb.WriteString(fmt.Sprintf("🎤 %s\n", artist))
	}

	// Album
	if album != "" {
		sb.WriteString(fmt.Sprintf("💿 %s\n", album))
	}

	// Bitrate
	sb.WriteString(fmt.Sprintf("🎵 %s\n", bitrate))

	// Filepath
	sb.WriteString(fmt.Sprintf("📁 %s", filepath))

	return sb.String()
}

func formatConsolePlaying(song mpd.Attrs, status mpd.Attrs) string {
	pos := status["song"]
	total := status["playlistlength"]
	elapsed := formatDuration(status["elapsed"])
	duration := formatDuration(song["duration"])
	track := song["Track"]
	title := song["Title"]
	artist := song["Artist"]
	album := song["Album"]
	bitrate := formatBitrate(status)
	filepath := song["file"]

	if title == "" {
		title = filepath
	}

	if track == "" {
		track = "?"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s▶ %s/%s/%s. %s%s\n", colorCyan, pos, total, track, title, colorReset))
	sb.WriteString(fmt.Sprintf("%s  🕓 %s / %s%s\n", colorCyan, elapsed, duration, colorReset))

	if artist != "" {
		sb.WriteString(fmt.Sprintf("%s  🎤 %s%s\n", colorYellow, artist, colorReset))
	}

	if album != "" {
		sb.WriteString(fmt.Sprintf("%s  💿 %s%s\n", colorOrange, album, colorReset))
	}

	sb.WriteString(fmt.Sprintf("%s  🎵 %s%s\n", colorBlue, bitrate, colorReset))
	sb.WriteString(fmt.Sprintf("%s  📁 %s%s", colorGreen, filepath, colorReset))

	return sb.String()
}

func sendNotification(state *AppState, event, title, message string, icon *gntp.Resource) error {
	// Skip if GNTP not enabled
	if !state.gntpEnabled || state.gntp == nil {
		return nil
	}

	opts := gntp.NewNotifyOptions()

	if icon != nil {
		opts.WithIcon(icon)
	}

	err := state.gntp.NotifyWithOptions(event, title, message, opts)
	if err != nil && state.debug {
		return err
	}
	return nil
}

// func reconnectMPD(state *AppState) error {
// 	if state.conn != nil {
// 		state.conn.Close()
// 	}

// 	conn, err := connectMPD(state.config.MPD.Host, state.config.MPD.Port, state.config.MPD.Timeout)
// 	if err != nil {
// 		return err
// 	}

// 	state.conn = conn
// 	return nil
// }

func reconnectMPD(state *AppState) error {
    if state.conn != nil {
        state.conn.Close()
        state.conn = nil
    }
    
    maxRetries := 5
    for i := 0; i < maxRetries; i++ {
        conn, err := connectMPD(state.config.MPD.Host, state.config.MPD.Port, state.config.MPD.Timeout)
        if err != nil {
            if state.debug {
                log.Printf("🔄 Reconnect attempt %d/%d failed: %v", i+1, maxRetries, err)
            }
            if i < maxRetries-1 {
                time.Sleep(time.Duration(i+1) * time.Second) // Exponential backoff
            }
            continue
        }
        
        state.conn = conn
        
        // Test the connection
        if err := conn.Ping(); err != nil {
            conn.Close()
            state.conn = nil
            if state.debug {
                log.Printf("🔄 Reconnect ping failed: %v", err)
            }
            continue
        }
        
        if state.debug {
            log.Printf("✅ Successfully reconnected on attempt %d", i+1)
        }
        return nil
    }
    
    return fmt.Errorf("failed to reconnect after %d attempts", maxRetries)
}

// func monitor(state *AppState) error {
// 	w, err := mpd.NewWatcher("tcp",
// 		fmt.Sprintf("%s:%s", state.config.MPD.Host, state.config.MPD.Port),
// 		"", "player", "mixer")
// 	if err != nil {
// 		return fmt.Errorf("failed to create watcher: %v", err)
// 	}
// 	defer w.Close()

// 	log.Println("🎵 MPD Monitor started")
// 	log.Printf("📡 Monitoring: %s:%s", state.config.MPD.Host, state.config.MPD.Port)
// 	if state.gntpEnabled {
// 		log.Printf("📢 GNTP Server: %s:%d", state.config.GNTP.Host, state.config.GNTP.Port)
// 		log.Printf("✅ GNTP registered (icon mode: %s)", state.config.GNTP.IconMode)
// 	} else {
// 		log.Println("📢 GNTP/Growl notifications: disabled")
// 	}
// 	if state.debug {
// 		log.Println("🐛 Debug mode: enabled")
// 	}
// 	fmt.Println(strings.Repeat("=", getTerminalWidth()))

// 	// Initial status
// 	if err := checkStatus(state); err != nil {
// 		if state.debug {
// 			log.Printf("⚠️  Initial status check failed: %v", err)
// 		}
// 	}

// 	// Monitor for errors
// 	go func() {
// 		for err := range w.Error {
// 			if state.debug {
// 				log.Printf("❌ Watcher error: %v", err)
// 			}
// 		}
// 	}()

// 	// Monitor for events
// 	for subsystem := range w.Event {
// 		// Skip database updates to avoid race conditions and bugs
// 		if subsystem == "database" || subsystem == "update" {
// 			continue
// 		}

// 		if err := checkStatus(state); err != nil {
// 			if state.debug {
// 				log.Printf("⚠️  Status check failed: %v", err)
// 			}

// 			// Try to reconnect if connection lost
// 			if strings.Contains(err.Error(), "EOF") || 
// 			   strings.Contains(err.Error(), "connection") ||
// 			   strings.Contains(err.Error(), "broken pipe") {
				
// 				if state.debug {
// 					log.Println("🔄 Attempting to reconnect to MPD...")
// 				}
				
// 				time.Sleep(2 * time.Second)
				
// 				if err := reconnectMPD(state); err != nil {
// 					if state.debug {
// 						log.Printf("❌ Reconnect failed: %v", err)
// 					}
// 					time.Sleep(5 * time.Second)
// 					continue
// 				}
				
// 				if state.debug {
// 					log.Println("✅ Reconnected to MPD")
// 				}
				
// 				// Recreate watcher
// 				w.Close()
// 				newWatcher, err := mpd.NewWatcher("tcp",
// 					fmt.Sprintf("%s:%s", state.config.MPD.Host, state.config.MPD.Port),
// 					"", "player", "mixer")
// 				if err != nil {
// 					if state.debug {
// 						log.Printf("❌ Failed to recreate watcher: %v", err)
// 					}
// 					time.Sleep(5 * time.Second)
// 					continue
// 				}
// 				w = newWatcher
				
// 				// Restart error monitor
// 				go func() {
// 					for err := range w.Error {
// 						if state.debug {
// 							log.Printf("❌ Watcher error: %v", err)
// 						}
// 					}
// 				}()
// 			}
// 		}
// 	}

// 	return nil
// }

// func monitor(state *AppState) error {
//     w, err := mpd.NewWatcher("tcp",
//         fmt.Sprintf("%s:%s", state.config.MPD.Host, state.config.MPD.Port),
//         "", "player", "mixer")
//     if err != nil {
//         return fmt.Errorf("failed to create watcher: %v", err)
//     }
//     defer w.Close()

//     log.Println("🎵 MPD Monitor started")
//     log.Printf("📡 Monitoring: %s:%s", state.config.MPD.Host, state.config.MPD.Port)
//     if state.gntpEnabled {
//         log.Printf("📢 GNTP Server: %s:%d", state.config.GNTP.Host, state.config.GNTP.Port)
//         log.Printf("✅ GNTP registered (icon mode: %s)", state.config.GNTP.IconMode)
//     } else {
//         log.Println("📢 GNTP/Growl notifications: disabled")
//     }
//     if state.debug {
//         log.Println("🐛 Debug mode: enabled")
//     }
//     fmt.Println(strings.Repeat("=", getTerminalWidth()))

//     // Initial status
//     if err := checkStatus(state); err != nil {
//         if state.debug {
//             log.Printf("⚠️  Initial status check failed: %v", err)
//         }
//     }

//     // Create a channel to signal when the error monitoring goroutine should stop
//     stopErrorMonitor := make(chan struct{})
    
//     // Monitor for errors with proper cleanup
//     go func() {
//         defer func() {
//             if r := recover(); r != nil && state.debug {
//                 log.Printf("Recovered from panic in error monitor: %v", r)
//             }
//         }()
        
//         for {
//             select {
//             case err, ok := <-w.Error:
//                 if !ok {
//                     // Channel closed, exit goroutine
//                     return
//                 }
//                 if state.debug {
//                     log.Printf("❌ Watcher error: %v", err)
//                 }
//             case <-stopErrorMonitor:
//                 // Received stop signal
//                 return
//             }
//         }
//     }()

//     // Monitor for events
//     for subsystem := range w.Event {
//         // Skip database updates to avoid race conditions and bugs
//         if subsystem == "database" || subsystem == "update" {
//             continue
//         }

//         if err := checkStatus(state); err != nil {
//             if state.debug {
//                 log.Printf("⚠️  Status check failed: %v", err)
//             }

//             // Try to reconnect if connection lost
//             if strings.Contains(err.Error(), "EOF") ||
//                 strings.Contains(err.Error(), "connection") ||
//                 strings.Contains(err.Error(), "broken pipe") {

//                 if state.debug {
//                     log.Println("🔄 Attempting to reconnect to MPD...")
//                 }

//                 // Signal the error monitoring goroutine to stop
//                 close(stopErrorMonitor)
                
//                 time.Sleep(2 * time.Second)

//                 if err := reconnectMPD(state); err != nil {
//                     if state.debug {
//                         log.Printf("❌ Reconnect failed: %v", err)
//                     }
//                     time.Sleep(5 * time.Second)
                    
//                     // Recreate the stop channel for the next iteration
//                     stopErrorMonitor = make(chan struct{})
//                     continue
//                 }

//                 if state.debug {
//                     log.Println("✅ Reconnected to MPD")
//                 }

//                 // Recreate watcher
//                 w.Close()
//                 newWatcher, err := mpd.NewWatcher("tcp",
//                     fmt.Sprintf("%s:%s", state.config.MPD.Host, state.config.MPD.Port),
//                     "", "player", "mixer")
//                 if err != nil {
//                     if state.debug {
//                         log.Printf("❌ Failed to recreate watcher: %v", err)
//                     }
//                     time.Sleep(5 * time.Second)
                    
//                     // Recreate the stop channel for the next iteration
//                     stopErrorMonitor = make(chan struct{})
//                     continue
//                 }
//                 w = newWatcher

//                 // Restart error monitor with new watcher
//                 stopErrorMonitor = make(chan struct{})
//                 go func() {
//                     defer func() {
//                         if r := recover(); r != nil && state.debug {
//                             log.Printf("Recovered from panic in error monitor: %v", r)
//                         }
//                     }()
                    
//                     for {
//                         select {
//                         case err, ok := <-w.Error:
//                             if !ok {
//                                 return
//                             }
//                             if state.debug {
//                                 log.Printf("❌ Watcher error: %v", err)
//                             }
//                         case <-stopErrorMonitor:
//                             return
//                         }
//                     }
//                 }()
//             }
//         }
//     }

//     return nil
// }

func monitor(state *AppState) error {
    log.Println("🎵 MPD Monitor started")
    log.Printf("📡 Monitoring: %s:%s", state.config.MPD.Host, state.config.MPD.Port)
    if state.gntpEnabled {
        log.Printf("📢 GNTP Server: %s:%d", state.config.GNTP.Host, state.config.GNTP.Port)
        log.Printf("✅ GNTP registered (icon mode: %s)", state.config.GNTP.IconMode)
    } else {
        log.Println("📢 GNTP/Growl notifications: disabled")
    }
    if state.debug {
        log.Println("🐛 Debug mode: enabled")
    }
    fmt.Println(strings.Repeat("=", getTerminalWidth()))

    // Initial status
    if err := checkStatus(state); err != nil {
        if state.debug {
            log.Printf("⚠️  Initial status check failed: %v", err)
        }
    }

    // Main monitoring loop with reconnection
    for {
        err := monitorOnce(state)
        if err != nil {
            if state.debug {
                log.Printf("❌ Monitor error: %v", err)
            }
            
            // Check if it's a connection error that warrants reconnection
            if strings.Contains(err.Error(), "EOF") ||
                strings.Contains(err.Error(), "connection") ||
                strings.Contains(err.Error(), "broken pipe") ||
                strings.Contains(err.Error(), "watcher") {
                
                if state.debug {
                    log.Println("🔄 Attempting to reconnect to MPD...")
                }
                
                time.Sleep(2 * time.Second)
                
                // Try to reconnect MPD
                if err := reconnectMPD(state); err != nil {
                    if state.debug {
                        log.Printf("❌ Reconnect failed: %v", err)
                    }
                    time.Sleep(5 * time.Second)
                    continue
                }
                
                if state.debug {
                    log.Println("✅ Reconnected to MPD")
                }
                
                // Continue the loop to create new watcher
                continue
            }
            
            // If it's not a connection error, return it
            return err
        }
        
        // If monitorOnce returns without error, it means we should reconnect
        if state.debug {
            log.Println("📡 Connection lost, attempting to reconnect...")
        }
        time.Sleep(2 * time.Second)
    }
}

func monitorOnce(state *AppState) error {
    // Create a new watcher
    w, err := mpd.NewWatcher("tcp",
        fmt.Sprintf("%s:%s", state.config.MPD.Host, state.config.MPD.Port),
        "", "player", "mixer")
    if err != nil {
        return fmt.Errorf("failed to create watcher: %v", err)
    }
    
    // Create a done channel to signal when monitoring should stop
    done := make(chan struct{})
    defer close(done)
    
    // Error handling goroutine with panic recovery
    go func() {
        defer func() {
            if r := recover(); r != nil && state.debug {
                log.Printf("🛡️  Recovered from panic in error monitor: %v", r)
            }
        }()
        
        for {
            select {
            case err, ok := <-w.Error:
                if !ok {
                    // Channel closed, exit goroutine
                    return
                }
                if state.debug {
                    log.Printf("⚠️  Watcher error: %v", err)
                }
            case <-done:
                // Monitoring stopped, exit goroutine
                return
            }
        }
    }()
    
    // Main event monitoring loop
    for {
        select {
        case subsystem, ok := <-w.Event:
            if !ok {
                // Event channel closed, return to trigger reconnection
                w.Close()
                return fmt.Errorf("watcher event channel closed")
            }
            
            // Skip database updates to avoid race conditions and bugs
            if subsystem == "database" || subsystem == "update" {
                continue
            }
            
            if err := checkStatus(state); err != nil {
                if state.debug {
                    log.Printf("⚠️  Status check failed: %v", err)
                }
                
                // If checkStatus fails with a connection error, close watcher and return
                if strings.Contains(err.Error(), "EOF") ||
                    strings.Contains(err.Error(), "connection") ||
                    strings.Contains(err.Error(), "broken pipe") {
                    w.Close()
                    return err
                }
            }
            
        case <-done:
            // Monitoring stopped, close watcher and return
            w.Close()
            return nil
            
        case <-time.After(30 * time.Second):
            // Periodic status check to ensure we're still connected
            if err := state.conn.Ping(); err != nil {
                if state.debug {
                    log.Printf("⚠️  Ping failed: %v", err)
                }
                w.Close()
                return fmt.Errorf("ping failed: %v", err)
            }
        }
    }
}

// func checkStatus(state *AppState) error {
// 	status, err := state.conn.Status()
// 	if err != nil {
// 		return fmt.Errorf("failed to get status: %v", err)
// 	}

// 	currentState := status["state"]

// 	// Get current song
// 	song, err := state.conn.CurrentSong()
// 	if err != nil {
// 		return fmt.Errorf("failed to get current song: %v", err)
// 	}

// 	currentFile := song["file"]

// 	// Check if song changed or state changed
// 	songChanged := currentFile != state.lastSongFile && currentFile != ""
// 	stateChanged := currentState != state.lastState && state.lastState != "" // Only if we have previous state

// 	// Display current status
// 	if currentState == "play" && currentFile != "" {
// 		info := formatConsolePlaying(song, status)
// 		fmt.Println()
// 		fmt.Println(info)
// 		printSeparator()
// 	} else if stateChanged {
// 		fmt.Printf("⏸  State: %s\n", currentState)
// 		printSeparator()
// 	}

// 	// Send notification for song change
// 	if songChanged && currentState == "play" {
// 		artwork := getAlbumArt(state.conn, currentFile)

// 		title := song["Title"]
// 		if title == "" {
// 			title = currentFile
// 		}

// 		message := formatCurrentPlaying(song, status)

// 		if err := sendNotification(state, "song_change", title, message, artwork); err != nil {
// 			if state.debug {
// 				log.Printf("⚠️  Failed to send notification: %v", err)
// 			}
// 		} //else if state.gntpEnabled {
// 		// 	fmt.Println("📢 Notification sent")
// 		// }

// 		state.lastSongFile = currentFile
// 	}

// 	// Send notification for state change (play, stop, pause)
// 	if stateChanged {
// 		var stateMsg string
// 		switch currentState {
// 		case "play":
// 			stateMsg = "▶ Playing"
// 		case "pause":
// 			stateMsg = "⏸ Paused"
// 		case "stop":
// 			stateMsg = "⏹ Stopped"
// 		default:
// 			stateMsg = fmt.Sprintf("State: %s", currentState)
// 		}

// 		var artwork *gntp.Resource
// 		if currentFile != "" {
// 			artwork = getAlbumArt(state.conn, currentFile)
// 		}

// 		message := stateMsg
// 		if currentState == "play" && currentFile != "" {
// 			message = formatCurrentPlaying(song, status)
// 		}

// 		if err := sendNotification(state, "player_state", stateMsg, message, artwork); err != nil {
// 			if state.debug {
// 				log.Printf("⚠️  Failed to send notification: %v", err)
// 			}
// 		} else if state.gntpEnabled {
// 			fmt.Println("📢 State notification sent")
// 		}
// 	}

// 	state.lastState = currentState

// 	return nil
// }

func checkStatus(state *AppState) error {
    // First, ping to check connection
    if err := state.conn.Ping(); err != nil {
        return fmt.Errorf("connection lost: %v", err)
    }
    
    status, err := state.conn.Status()
    if err != nil {
        return fmt.Errorf("failed to get status: %v", err)
    }

    currentState := status["state"]

    // Get current song
    song, err := state.conn.CurrentSong()
    if err != nil {
        return fmt.Errorf("failed to get current song: %v", err)
    }

    currentFile := song["file"]

    // Check if song changed or state changed
    songChanged := currentFile != state.lastSongFile && currentFile != ""
    stateChanged := currentState != state.lastState && state.lastState != "" // Only if we have previous state

    // Display current status
    if currentState == "play" && currentFile != "" {
        info := formatConsolePlaying(song, status)
        fmt.Println()
        fmt.Println(info)
        printSeparator()
    } else if stateChanged {
        fmt.Printf("⏸  State: %s\n", currentState)
        printSeparator()
    }

    // Send notification for song change
    if songChanged && currentState == "play" {
        artwork := getAlbumArt(state.conn, currentFile)

        title := song["Title"]
        if title == "" {
            title = currentFile
        }

        message := formatCurrentPlaying(song, status)

        if err := sendNotification(state, "song_change", title, message, artwork); err != nil {
            if state.debug {
                log.Printf("⚠️  Failed to send notification: %v", err)
            }
        }

        state.lastSongFile = currentFile
    }

    // Send notification for state change (play, stop, pause)
    if stateChanged {
        var stateMsg string
        switch currentState {
        case "play":
            stateMsg = "▶ Playing"
        case "pause":
            stateMsg = "⏸ Paused"
        case "stop":
            stateMsg = "⏹ Stopped"
        default:
            stateMsg = fmt.Sprintf("State: %s", currentState)
        }

        var artwork *gntp.Resource
        if currentFile != "" {
            artwork = getAlbumArt(state.conn, currentFile)
        }

        message := stateMsg
        if currentState == "play" && currentFile != "" {
            message = formatCurrentPlaying(song, status)
        }

        if err := sendNotification(state, "player_state", stateMsg, message, artwork); err != nil {
            if state.debug {
                log.Printf("⚠️  Failed to send notification: %v", err)
            }
        } //else if state.gntpEnabled {
        //     fmt.Println("📢 State notification sent")
        // }
    }

    state.lastState = currentState

    return nil
}

func main() {
	var (
		configFile string
		mpdHost    string
		mpdPort    string
		mpdTimeout int
		gntpHost   string
		gntpPort   int
		gntpPass   string
		iconMode   string
	)

	flag.StringVar(&configFile, "config", "", "Path to TOML config file")
	flag.StringVar(&mpdHost, "mpd-host", "", "MPD host (default: localhost or MPD_HOST env)")
	flag.StringVar(&mpdPort, "mpd-port", "", "MPD port (default: 6600 or MPD_PORT env)")
	flag.IntVar(&mpdTimeout, "mpd-timeout", 0, "MPD timeout in seconds (default: 10 or MPD_TIMEOUT env)")
	flag.StringVar(&gntpHost, "gntp-host", "", "GNTP/Growl host (default: localhost)")
	flag.IntVar(&gntpPort, "gntp-port", 0, "GNTP/Growl port (default: 23053)")
	flag.StringVar(&gntpPass, "gntp-password", "", "GNTP/Growl password")
	flag.StringVar(&iconMode, "icon-mode", "", "Icon mode: binary, dataurl, fileurl, httpurl (default: binary)")

	flag.Parse()

	// Check DEBUG environment variable
	debug := os.Getenv("DEBUG") == "1"

	// Load config file
	config, err := loadConfig(configFile)
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	// Override with environment variables
	config.MPD.Host = getEnvOrDefault("MPD_HOST", config.MPD.Host)
	config.MPD.Port = getEnvOrDefault("MPD_PORT", config.MPD.Port)

	if timeoutStr := os.Getenv("MPD_TIMEOUT"); timeoutStr != "" {
		if t, err := strconv.Atoi(timeoutStr); err == nil {
			config.MPD.Timeout = t
		}
	}

	// Override with command line arguments
	if mpdHost != "" {
		config.MPD.Host = mpdHost
	}
	if mpdPort != "" {
		config.MPD.Port = mpdPort
	}
	if mpdTimeout > 0 {
		config.MPD.Timeout = mpdTimeout
	}
	if gntpHost != "" {
		config.GNTP.Host = gntpHost
	}
	if gntpPort > 0 {
		config.GNTP.Port = gntpPort
	}
	if gntpPass != "" {
		config.GNTP.Password = gntpPass
	}
	if iconMode != "" {
		config.GNTP.IconMode = iconMode
	}

	// Connect to MPD
	conn, err := connectMPD(config.MPD.Host, config.MPD.Port, config.MPD.Timeout)
	if err != nil {
		log.Fatalf("❌ %v", err)
	}
	defer conn.Close()

	// Setup GNTP (optional - don't fail if not available)
	gntpClient, gntpEnabled := setupGNTP(config, debug)

	state := &AppState{
		conn:        conn,
		gntp:        gntpClient,
		config:      config,
		debug:       debug,
		gntpEnabled: gntpEnabled,
	}

	// Start monitoring
	if err := monitor(state); err != nil {
		log.Fatalf("❌ Monitor error: %v", err)
	}
}