package web

import (
	"embed"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

//go:embed all:build
var files embed.FS

// SvelteKitHandler serves a prerendered SvelteKit static build with 200.html fallback
func SvelteKitHandler(mount string) http.Handler {
	fsys, err := fs.Sub(files, "build")
	if err != nil {
		log.Fatalf("failed to access embedded filesystem: %v", err)
	}
	log.Println("SvelteKit handler initialized with embedded filesystem")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originalPath := r.URL.Path
		reqPath := strings.TrimPrefix(r.URL.Path, mount)
		log.Printf("Request: %s %s", r.Method, originalPath)

		// Root request
		if reqPath == "" || reqPath == "/" {
			reqPath = "index.html"
			log.Printf("  → serving root: %s", reqPath)
		} else {
			name := path.Base(reqPath)

			// Check if it looks like a file (has extension)
			if strings.Contains(name, ".") {
				// Looks like an asset - serve it directly or 404
				log.Printf("  → serving asset: %s", reqPath)
				// reqPath stays as-is, will 404 if not found below
			} else {
				// No extension - might be a prerendered page route
				// Try: /path.html, /path/index.html, then fallback to 200.html

				testPath := strings.TrimPrefix(reqPath, "/")
				if _, err := fsys.Open(testPath + ".html"); err == nil {
					// Try with .html extension
					reqPath = testPath + ".html"
					log.Printf("  → serving as .html: %s", reqPath)
				} else if _, err := fsys.Open(testPath + "/index.html"); err == nil {
					// Try /path/index.html
					reqPath = testPath + "/index.html"
					log.Printf("  → serving /index.html: %s", reqPath)
				} else {
					// Not found, use SPA fallback
					log.Printf("  → route not found: %s, fallback to 200.html", reqPath)
					reqPath = "200.html"
				}
			}
		}

		// Remove leading slash for fsys.Open
		reqPath = strings.TrimPrefix(reqPath, "/")

		// Open and serve the file
		file, err := fsys.Open(reqPath)
		if err != nil {
			log.Printf("  → error opening file %s: %v", reqPath, err)
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		defer file.Close()

		// Read file content
		content, err := io.ReadAll(file)
		if err != nil {
			log.Printf("  → error reading file: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Detect content type from file extension
		contentType := mime.TypeByExtension(filepath.Ext(reqPath))
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		w.Header().Set("Content-Type", contentType)
		log.Printf("  → successfully serving: %s (%s, %d bytes)", reqPath, contentType, len(content))
		w.Write(content)
	})
}
