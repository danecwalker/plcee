package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func isAuthenticatedRequest(r *http.Request) bool {
	cookie, err := r.Cookie("session")
	return err == nil && cookie.Value == "authenticated"
}

func listDeviceRecordingFiles() ([]string, error) {
	entries, err := os.ReadDir(deviceLogDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "recording_") && strings.HasSuffix(name, ".csv") {
			files = append(files, name)
		}
	}

	sort.Strings(files)
	return files, nil
}

func buildDeviceLogsArchive(files []string) (*os.File, int, error) {
	tmp, err := os.CreateTemp("", "delphi-device-logs-*.zip")
	if err != nil {
		return nil, 0, err
	}

	zipWriter := zip.NewWriter(tmp)
	written := 0

	for _, name := range files {
		fullPath := filepath.Join(deviceLogDir, name)
		file, err := os.Open(fullPath)
		if err != nil {
			log.Printf("failed to open device log %s: %v", name, err)
			continue
		}

		info, err := file.Stat()
		if err != nil {
			log.Printf("failed to stat device log %s: %v", name, err)
			_ = file.Close()
			continue
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			log.Printf("failed to create zip header for %s: %v", name, err)
			_ = file.Close()
			continue
		}
		header.Name = name
		header.Method = zip.Deflate

		entryWriter, err := zipWriter.CreateHeader(header)
		if err != nil {
			log.Printf("failed to create zip entry for %s: %v", name, err)
			_ = file.Close()
			continue
		}

		if _, err := io.Copy(entryWriter, file); err != nil {
			log.Printf("failed to copy device log %s into zip: %v", name, err)
			_ = file.Close()
			continue
		}
		_ = file.Close()
		written++
	}

	if err := zipWriter.Close(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return nil, written, err
	}

	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return nil, written, err
	}

	return tmp, written, nil
}

// deviceLogsDownloadHandler returns a zip containing all locally stored CSV logs.
func deviceLogsDownloadHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if !isAuthenticatedRequest(r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		files, err := listDeviceRecordingFiles()
		if err != nil {
			log.Printf("failed to list device logs: %v", err)
			http.Error(w, "Failed to list device logs", http.StatusInternalServerError)
			return
		}
		if len(files) == 0 {
			http.Error(w, "No device logs available", http.StatusNotFound)
			return
		}

		archiveFile, written, err := buildDeviceLogsArchive(files)
		if err != nil {
			log.Printf("failed to build device log archive: %v", err)
			http.Error(w, "Failed to archive device logs", http.StatusInternalServerError)
			return
		}
		defer func() {
			name := archiveFile.Name()
			_ = archiveFile.Close()
			_ = os.Remove(name)
		}()

		if written == 0 {
			http.Error(w, "No readable device logs available", http.StatusInternalServerError)
			return
		}

		archiveName := fmt.Sprintf("delphi-device-logs-%s.zip", time.Now().Format("20060102-150405"))
		if info, err := archiveFile.Stat(); err == nil {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", archiveName))

		if _, err := io.Copy(w, archiveFile); err != nil {
			log.Printf("failed to stream device log archive: %v", err)
		}
	}
}
