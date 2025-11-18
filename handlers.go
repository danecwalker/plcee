package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"plcee/web"
	"slices"
	"strconv"
	"strings"
	"time"
)

func snapshotStreamHandler(pins *Pins, state *State) http.HandlerFunc {
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
				data, err := json.Marshal(state)
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

func snapshotHandler(pins *Pins, state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state.read(pins)
		// Serialize state to JSON and write to response
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(state); err != nil {
			log.Printf("error encoding snapshot response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

func dataHandler(data *Data) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read data without lock - commands write, this only reads
		// Go's memory model allows concurrent reads
		dataCopy := Data{
			TensionSettings: data.TensionSettings,
			CalTable: CalTableConfig{
				CalPoints: make(map[string]string, len(data.CalTable.CalPoints)),
			},
			LogSettings: data.LogSettings,
		}
		// Copy the map
		for k, v := range data.CalTable.CalPoints {
			dataCopy.CalTable.CalPoints[k] = v
		}

		// Serialize state to JSON and write to response
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(dataCopy); err != nil {
			log.Printf("error encoding data response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

func commandHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var cmd Command
		if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
			log.Printf("error decoding command: %v", err)
			http.Error(w, "Invalid command", http.StatusBadRequest)
			return
		}
		log.Printf("received command: %s", cmd.Name)

		// Try to send to queue
		select {
		case commandQueue <- cmd:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		default:
			// Queue is full - drop oldest command and add new one
			select {
			case oldCmd := <-commandQueue:
				log.Printf("warning: command queue full, dropping old command: %s", oldCmd.Name)
			default:
			}

			// Now add the new command
			select {
			case commandQueue <- cmd:
				log.Printf("added new command after dropping old one: %s", cmd.Name)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"ok"}`))
			default:
				// Still couldn't add (shouldn't happen)
				log.Printf("error: still couldn't add command after drop: %s", cmd.Name)
				http.Error(w, "Command queue error", http.StatusServiceUnavailable)
			}
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

		// Simple password check (in a real application, use secure methods)
		if data.AdminPassword == creds.Password {
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
		// Check if there's a hardware error and redirect to error page
		// unless we're already on the error page or API endpoint
		// if hardwareError != nil && r.URL.Path != "/error" && r.URL.Path != "/api/hardware-error" {
		// 	http.Redirect(w, r, "/error", http.StatusSeeOther)
		// 	return
		// }

		if slices.Contains(data.ProtectedRoutes, r.URL.Path) {
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
