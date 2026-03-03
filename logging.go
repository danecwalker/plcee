package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LogEntry represents a single log entry to be written.
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

// LogCommand represents a logging control command.
type LogCommand struct {
	Action string // "log", "start", "stop"
	Entry  LogEntry
}

var (
	logQueue chan LogCommand
)

const (
	usbMountPoint = "/mnt/usb"
	usbLogSubdir  = "data"
	deviceLogDir  = "/var/log/delphi"
)

var csvHeader = []string{"Timestamp", "Load", "ProxValue", "DumpValve", "Speed", "AlarmError", "AlarmWarn", "MaxTension"}

// checkUsbConnected checks if USB drive is mounted.
func checkUsbConnected() bool {
	return isMountPoint(usbMountPoint)
}

func resolveUSBLogDir() (string, error) {
	if !checkUsbConnected() {
		return "", fmt.Errorf("USB not mounted")
	}

	logDir := filepath.Join(usbMountPoint, usbLogSubdir)
	if err := ensureWritableDir(logDir, "USB data directory"); err != nil {
		return "", err
	}

	return logDir, nil
}

func resolveDeviceLogDir() (string, error) {
	if err := ensureWritableDir(deviceLogDir, "device log directory"); err != nil {
		return "", err
	}
	return deviceLogDir, nil
}

func ensureWritableDir(dirPath string, label string) error {
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return fmt.Errorf("failed to prepare %s: %w", label, err)
	}

	probePath := filepath.Join(dirPath, ".delphi-write-probe")
	probeFile, err := os.OpenFile(probePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("%s is not writable: %w", label, err)
	}
	if err := probeFile.Close(); err != nil {
		return fmt.Errorf("failed to close probe file for %s: %w", label, err)
	}
	_ = os.Remove(probePath)

	return nil
}

func isMountPoint(target string) bool {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 5 {
			continue
		}

		mountPoint := strings.NewReplacer(
			"\\040", " ",
			"\\011", "\t",
			"\\012", "\n",
			"\\134", "\\",
		).Replace(fields[4])

		if mountPoint == target {
			return true
		}
	}

	return scanner.Err() == nil
}

func openRecordingFile(logDir string, sessionID string) (*os.File, *csv.Writer, string, error) {
	filename := filepath.Join(logDir, fmt.Sprintf("recording_%s.csv", sessionID))
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, "", err
	}

	writer := csv.NewWriter(file)

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, "", err
	}
	if info.Size() == 0 {
		if err := writer.Write(csvHeader); err != nil {
			file.Close()
			return nil, nil, "", err
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			file.Close()
			return nil, nil, "", err
		}
	}

	return file, writer, filename, nil
}

// startLogger starts the background logging worker.
func startLogger(state *State, data *Data) {
	logQueue = make(chan LogCommand, 100)

	go func() {
		var usbFile *os.File
		var usbWriter *csv.Writer
		var deviceFile *os.File
		var deviceWriter *csv.Writer
		var currentSessionID string
		var lastStopTime time.Time
		var isLogging bool
		var hasEverStopped bool

		closeSink := func(file **os.File, writer **csv.Writer) {
			if *writer != nil {
				(*writer).Flush()
			}
			if *file != nil {
				_ = (*file).Close()
			}
			*file = nil
			*writer = nil
		}

		closeAllSinks := func() {
			closeSink(&usbFile, &usbWriter)
			closeSink(&deviceFile, &deviceWriter)
		}

		setUSBStatus := func(connected bool, usbErr string) {
			mu.Lock()
			state.UsbConnected = connected
			state.UsbError = usbErr
			mu.Unlock()
		}

		setDeviceLogError := func(deviceErr string) {
			mu.Lock()
			state.DeviceLogError = deviceErr
			mu.Unlock()
		}

		openDeviceSink := func() {
			if deviceWriter != nil || currentSessionID == "" {
				return
			}

			logDir, err := resolveDeviceLogDir()
			if err != nil {
				setDeviceLogError(err.Error())
				log.Printf("device logging unavailable: %v", err)
				return
			}

			file, writer, filename, err := openRecordingFile(logDir, currentSessionID)
			if err != nil {
				setDeviceLogError("device log write failed")
				log.Printf("error opening device log file: %v", err)
				return
			}

			deviceFile = file
			deviceWriter = writer
			setDeviceLogError("")
			log.Printf("device recording active: %s", filename)
		}

		openUSBSink := func() {
			if usbWriter != nil || currentSessionID == "" {
				return
			}

			logDir, err := resolveUSBLogDir()
			if err != nil {
				setUSBStatus(false, err.Error())
				log.Printf("USB unavailable, continuing with device-only logging: %v", err)
				return
			}

			file, writer, filename, err := openRecordingFile(logDir, currentSessionID)
			if err != nil {
				setUSBStatus(false, "USB write failed")
				log.Printf("error opening USB log file: %v", err)
				return
			}

			usbFile = file
			usbWriter = writer
			setUSBStatus(true, "")
			log.Printf("USB recording active: %s", filename)
		}

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case cmd := <-logQueue:
				switch cmd.Action {
				case "start":
					now := time.Now()
					timeSinceStop := now.Sub(lastStopTime)

					mu.RLock()
					currentDelayMs := data.LogSettings.LogDelayMs
					mu.RUnlock()

					if currentSessionID == "" || !hasEverStopped || timeSinceStop > time.Duration(currentDelayMs)*time.Millisecond {
						closeAllSinks()
						currentSessionID = now.Format("2006-01-02_150405")
					} else {
						log.Printf("continuing existing recording (time since stop: %v, delay: %dms)", timeSinceStop, currentDelayMs)
					}

					openDeviceSink()
					openUSBSink()
					isLogging = deviceWriter != nil || usbWriter != nil

				case "stop":
					isLogging = false
					lastStopTime = time.Now()
					hasEverStopped = true

				case "log":
					if !isLogging {
						continue
					}

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

					if deviceWriter != nil {
						if err := deviceWriter.Write(record); err != nil {
							log.Printf("error writing device CSV record: %v", err)
							setDeviceLogError("device log write failed")
							closeSink(&deviceFile, &deviceWriter)
						} else {
							deviceWriter.Flush()
							if err := deviceWriter.Error(); err != nil {
								log.Printf("error flushing device CSV record: %v", err)
								setDeviceLogError("device log write failed")
								closeSink(&deviceFile, &deviceWriter)
							}
						}
					}

					if usbWriter != nil {
						if err := usbWriter.Write(record); err != nil {
							log.Printf("error writing USB CSV record: %v", err)
							setUSBStatus(false, "USB write failed")
							closeSink(&usbFile, &usbWriter)
						} else {
							usbWriter.Flush()
							if err := usbWriter.Error(); err != nil {
								log.Printf("error flushing USB CSV record: %v", err)
								setUSBStatus(false, "USB write failed")
								closeSink(&usbFile, &usbWriter)
							}
						}
					}

					isLogging = deviceWriter != nil || usbWriter != nil
				}

			case <-ticker.C:
				connected := checkUsbConnected()
				if !connected {
					setUSBStatus(false, "USB not connected")
				} else {
					mu.Lock()
					state.UsbConnected = true
					if state.UsbError == "USB not connected" {
						state.UsbError = ""
					}
					mu.Unlock()
				}

				if !connected && usbFile != nil {
					closeSink(&usbFile, &usbWriter)
					log.Println("USB disconnected, USB logging stopped")
				}

				if isLogging && currentSessionID != "" {
					if deviceWriter == nil {
						openDeviceSink()
					}
					if connected && usbWriter == nil {
						openUSBSink()
					}
					isLogging = deviceWriter != nil || usbWriter != nil
				}
			}
		}
	}()

	log.Println("logging worker started")
}

// queueLogEntry sends a log entry to the logging queue (call from main loop).
func queueLogEntry(state *State) {
	mu.RLock()
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
	mu.RUnlock()

	select {
	case logQueue <- LogCommand{Action: "log", Entry: entry}:
	default:
		// Queue full; drop this entry to keep control loop real-time.
	}
}

// queueLogStart signals that logging should start/continue.
func queueLogStart() {
	select {
	case logQueue <- LogCommand{Action: "start"}:
	default:
	}
}

// queueLogStop signals that logging should pause.
func queueLogStop() {
	select {
	case logQueue <- LogCommand{Action: "stop"}:
	default:
	}
}
