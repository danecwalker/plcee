package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// LogEntry represents a single log entry to be written
type LogEntry struct {
	Timestamp  time.Time
	Load       float64
	ProxValue  float64
	DumpValve  bool
	Speed      bool
	AlarmError bool
	AlarmWarn  bool
	MaxTension bool
}

// LogCommand represents a logging control command
type LogCommand struct {
	Action     string // "log", "start", "stop"
	FootSwitch bool
	Entry      LogEntry
}

var (
	logQueue     chan LogCommand
	usbMountPath = "/mnt/usb/data" // Common USB mount point, adjust as needed
)

// checkUsbConnected checks if USB drive is mounted
func checkUsbConnected() bool {
	info, err := os.Stat(usbMountPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// startLogger starts the background logging worker
func startLogger(state *State, data *Data) {
	logQueue = make(chan LogCommand, 100)

	go func() {
		var currentFile *os.File
		var csvWriter *csv.Writer
		var lastStopTime time.Time
		var isLogging bool
		var hasEverStopped bool

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case cmd := <-logQueue:
				switch cmd.Action {
				case "start":
					// FootSwitch turned on
					now := time.Now()
					timeSinceStop := now.Sub(lastStopTime)

					// Read current delay setting
					currentDelayMs := data.LogSettings.LogDelayMs

					// Check if we should start a new recording or continue existing one
					// Create new file if: never stopped before OR time since stop exceeds delay
					if !hasEverStopped || timeSinceStop > time.Duration(currentDelayMs)*time.Millisecond {
						// Close previous file if open
						if currentFile != nil {
							csvWriter.Flush()
							currentFile.Close()
							currentFile = nil
							csvWriter = nil
						}

						// Start new recording if USB is connected
						if checkUsbConnected() {
							// Use timestamp for filename: recording_2025-11-05_143022.csv
							timestamp := now.Format("2006-01-02_150405")
							filename := filepath.Join(usbMountPath, fmt.Sprintf("recording_%s.csv", timestamp))
							file, err := os.Create(filename)
							if err != nil {
								log.Printf("error creating log file: %v", err)
								mu.Lock()
								state.UsbConnected = false
								mu.Unlock()
								break
							}

							currentFile = file
							csvWriter = csv.NewWriter(file)

							// Write header
							header := []string{"Timestamp", "Load", "ProxValue", "DumpValve", "Speed", "AlarmError", "AlarmWarn", "MaxTension"}
							if err := csvWriter.Write(header); err != nil {
								log.Printf("error writing CSV header: %v", err)
							}
							csvWriter.Flush()

							log.Printf("started new recording: %s (delay was %dms, time since stop: %v)", filename, currentDelayMs, timeSinceStop)
							mu.Lock()
							state.UsbConnected = true
							mu.Unlock()
						} else {
							log.Println("USB not connected, cannot start logging")
							mu.Lock()
							state.UsbConnected = false
							mu.Unlock()
						}
					} else {
						log.Printf("continuing existing recording (time since stop: %v, delay: %dms)", timeSinceStop, currentDelayMs)
					}
					isLogging = true

				case "stop":
					// FootSwitch turned off
					isLogging = false
					lastStopTime = time.Now()
					hasEverStopped = true

				case "log":
					// Write log entry
					if isLogging && currentFile != nil && csvWriter != nil {
						entry := cmd.Entry
						record := []string{
							entry.Timestamp.Format(time.RFC3339Nano),
							fmt.Sprintf("%.3f", entry.Load),
							fmt.Sprintf("%.3f", entry.ProxValue),
							fmt.Sprintf("%t", entry.DumpValve),
							fmt.Sprintf("%t", entry.Speed),
							fmt.Sprintf("%t", entry.AlarmError),
							fmt.Sprintf("%t", entry.AlarmWarn),
							fmt.Sprintf("%t", entry.MaxTension),
						}
						if err := csvWriter.Write(record); err != nil {
							log.Printf("error writing CSV record: %v", err)
						}
						csvWriter.Flush()
					}
				}

			case <-ticker.C:
				// Periodic check for USB connection status
				connected := checkUsbConnected()
				mu.Lock()
				state.UsbConnected = connected
				mu.Unlock()

				if !connected && currentFile != nil {
					// USB was disconnected, close file
					csvWriter.Flush()
					currentFile.Close()
					currentFile = nil
					csvWriter = nil
					isLogging = false
					log.Println("USB disconnected, stopped logging")
				}
			}
		}
	}()

	log.Println("logging worker started")
}

// queueLogEntry sends a log entry to the logging queue (call from main loop)
func queueLogEntry(state *State) {
	// Non-blocking send to avoid allocations/delays in main loop
	entry := LogEntry{
		Timestamp:  time.Now(),
		Load:       state.Load,
		ProxValue:  state.ProxValue,
		DumpValve:  state.DumpValve,
		Speed:      state.Speed,
		AlarmError: state.AlarmError,
		AlarmWarn:  state.AlarmWarn,
		MaxTension: state.MaxTension,
	}

	select {
	case logQueue <- LogCommand{Action: "log", Entry: entry}:
	default:
		// Queue full, drop this entry
	}
}

// queueLogStart signals that logging should start/continue
func queueLogStart() {
	select {
	case logQueue <- LogCommand{Action: "start"}:
	default:
	}
}

// queueLogStop signals that logging should pause
func queueLogStop() {
	select {
	case logQueue <- LogCommand{Action: "stop"}:
	default:
	}
}
