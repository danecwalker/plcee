package main

import (
	"delphi/web"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

func snapshotState(state *State) State {
	mu.RLock()
	defer mu.RUnlock()
	return *state
}

func snapshotConfig(data *Data) Data {
	mu.RLock()
	defer mu.RUnlock()

	dataCopy := Data{
		TensionSettings: data.TensionSettings,
		CalTable: CalTableConfig{
			CalPoints: make(map[string]string, len(data.CalTable.CalPoints)),
		},
		LogSettings:      data.LogSettings,
		DistancePerPulse: data.DistancePerPulse,
	}

	for k, v := range data.CalTable.CalPoints {
		dataCopy.CalTable.CalPoints[k] = v
	}

	return dataCopy
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("error encoding JSON response: %v", err)
	}
}

func snapshotStreamHandler(_ *Pins, state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse rate parameter
		tStr := r.URL.Query().Get("t")
		interval := 1000 // default to 1s
		if tStr != "" {
			if v, err := strconv.Atoi(tStr); err == nil && v > 0 {
				interval = v
			}
		}

		// Prepare headers for SSE
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// Stream loop
		ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
		defer ticker.Stop()

		ctx := r.Context()

		for {
			select {
			case <-ctx.Done():
				log.Println("client disconnected from /snapshot/stream")
				return
			case <-ticker.C:
				snapshot := snapshotState(state)
				data, err := json.Marshal(snapshot)
				if err != nil {
					log.Printf("error marshaling snapshot: %v", err)
					continue
				}

				// Send SSE event
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}
	}
}

func snapshotHandler(_ *Pins, state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshot := snapshotState(state)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(snapshot); err != nil {
			log.Printf("error encoding snapshot response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

func dataHandler(data *Data) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dataCopy := snapshotConfig(data)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(dataCopy); err != nil {
			log.Printf("error encoding data response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

func commandHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var cmd Command
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&cmd); err != nil {
			log.Printf("error decoding command: %v", err)
			writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Invalid command payload"})
			return
		}
		if cmd.Name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Command name is required"})
			return
		}
		log.Printf("received command: %s", cmd.Name)

		cmd.Result = make(chan CommandResult, 1)

		select {
		case commandQueue <- cmd:
			// queued successfully
		default:
			log.Printf("warning: command queue full, rejecting command: %s", cmd.Name)
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Command queue full"})
			return
		}

		select {
		case result := <-cmd.Result:
			if !result.OK {
				writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": result.Error})
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		case <-time.After(2 * time.Second):
			log.Printf("command timed out waiting for execution: %s", cmd.Name)
			writeJSON(w, http.StatusGatewayTimeout, map[string]string{"status": "error", "error": "Command execution timed out"})
		}
	}
}

func authHandler(data *Data) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			http.SetCookie(w, &http.Cookie{
				Name:   "session",
				Value:  "",
				MaxAge: -1,
			})
			w.WriteHeader(http.StatusOK)
			return
		} else if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var creds struct {
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			log.Printf("error decoding auth credentials: %v", err)
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		mu.RLock()
		adminPassword := data.AdminPassword
		mu.RUnlock()

		// Simple password check (in a real application, use secure methods)
		if adminPassword == creds.Password {
			log.Println("authentication successful")
			// add session cookie
			http.SetCookie(w, &http.Cookie{
				Name:  "session",
				Value: "authenticated",
			})
			w.WriteHeader(http.StatusOK)
		} else {
			log.Printf("authentication failed for password attempt")
			// clear session cookie
			http.SetCookie(w, &http.Cookie{
				Name:   "session",
				Value:  "",
				MaxAge: -1,
			})
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	}
}

func rootHandler(data *Data) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		protectedRoutes := append([]string(nil), data.ProtectedRoutes...)
		mu.RUnlock()

		if slices.Contains(protectedRoutes, r.URL.Path) {
			cookie, err := r.Cookie("session")
			if err != nil || cookie.Value != "authenticated" {
				log.Printf("unauthorized access attempt to: %s", r.URL.Path)
				http.Redirect(w, r, "/login?redirect="+r.URL.Path, http.StatusSeeOther)
				return
			}
		} else {
			if len(strings.Split(r.URL.Path, ".")) <= 1 {
				http.SetCookie(w, &http.Cookie{
					Name:   "session",
					Value:  "",
					MaxAge: -1,
				})
			}
		}

		web.SvelteKitHandler("/").ServeHTTP(w, r)
	}
}

func loggingHealthHandler(data *Data, state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		mu.RLock()
		logSettings := data.LogSettings
		mu.RUnlock()

		currentState := State{}
		if state != nil {
			currentState = snapshotState(state)
		}
		if currentState.ControlLoopError == "" && hardwareError != nil {
			currentState.ControlLoopError = hardwareError.Error()
		}

		queueDepth := len(logQueue)
		queueCapacity := cap(logQueue)
		queueUtilization := 0.0
		if queueCapacity > 0 {
			queueUtilization = float64(queueDepth) / float64(queueCapacity)
		}

		usbMounted := checkUsbConnected()
		usbHealthy := currentState.UsbConnected && currentState.UsbError == ""
		deviceHealthy := currentState.DeviceLogError == ""

		status := "ok"
		issues := make([]string, 0, 4)

		if currentState.ControlLoopError != "" {
			status = "error"
			issues = append(issues, "control loop error: "+currentState.ControlLoopError)
		}

		if logSettings.Enabled {
			if currentState.DeviceLogError != "" {
				if status == "ok" {
					status = "degraded"
				}
				issues = append(issues, "device logging: "+currentState.DeviceLogError)
			}

			if currentState.UsbError != "" {
				if status == "ok" {
					status = "degraded"
				}
				issues = append(issues, "usb logging: "+currentState.UsbError)
			} else if !currentState.UsbConnected {
				if status == "ok" {
					status = "degraded"
				}
				issues = append(issues, "usb logging: USB not connected")
			}
		}

		if queueCapacity > 0 && queueUtilization >= 0.80 {
			if status == "ok" {
				status = "degraded"
			}
			issues = append(issues, fmt.Sprintf("logging queue utilization high: %.0f%%", queueUtilization*100))
		}

		httpStatus := http.StatusOK
		if status == "error" {
			httpStatus = http.StatusServiceUnavailable
		}

		writeJSON(w, httpStatus, map[string]any{
			"status":    status,
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
			"issues":    issues,
			"logging": map[string]any{
				"enabled":    logSettings.Enabled,
				"intervalMs": logSettings.IntervalMs,
				"delayMs":    logSettings.LogDelayMs,
			},
			"usb": map[string]any{
				"connected": currentState.UsbConnected,
				"mounted":   usbMounted,
				"healthy":   usbHealthy,
				"error":     currentState.UsbError,
			},
			"device": map[string]any{
				"logDir":  deviceLogDir,
				"healthy": deviceHealthy,
				"error":   currentState.DeviceLogError,
			},
			"queue": map[string]any{
				"depth":       queueDepth,
				"capacity":    queueCapacity,
				"utilization": queueUtilization,
			},
			"controlLoop": map[string]any{
				"error": currentState.ControlLoopError,
			},
		})
	}
}

func hardwareErrorAPIHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		errorMessage := "Unknown hardware error"
		if hardwareError != nil {
			errorMessage = hardwareError.Error()
		}

		response := map[string]string{
			"error": errorMessage,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("error encoding hardware error response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}
