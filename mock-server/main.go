package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// JSONData stores the current data and provides thread-safe access
type JSONData struct {
	mu   sync.RWMutex
	data interface{}
}

// Update safely updates the JSON data
func (j *JSONData) Update(newData interface{}) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.data = newData
}

// Get safely retrieves the JSON data
func (j *JSONData) Get() interface{} {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.data
}

var (
	jsonData = &JSONData{}
	filename string
)

// loadJSON reads and parses the JSON file
func loadJSON() error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}

	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return fmt.Errorf("error parsing JSON: %v", err)
	}

	jsonData.Update(data)
	log.Println("JSON data reloaded successfully")
	return nil
}

// setupFileWatcher initializes the file watcher for hot reloading
func setupFileWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("error creating watcher: %v", err)
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("File modified. Reloading...")
					if err := loadJSON(); err != nil {
						log.Printf("Error reloading JSON: %v", err)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Watcher error: %v", err)
			}
		}
	}()

	err = watcher.Add(filename)
	if err != nil {
		return fmt.Errorf("error watching file: %v", err)
	}

	return nil
}

// jsonHandler serves the current JSON data
func jsonHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	data := jsonData.Get()
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		return
	}
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: go run main.go <json-file>")
	}
	filename = os.Args[1]

	// Initial load of JSON data
	if err := loadJSON(); err != nil {
		log.Fatal(err)
	}

	// Setup file watcher for hot reloading
	if err := setupFileWatcher(); err != nil {
		log.Fatal(err)
	}

	// Setup HTTP server
	http.HandleFunc("/", jsonHandler)

	port := "8080"
	log.Printf("Server starting on port %s...", port)
	log.Printf("Serving JSON from file: %s", filename)
	log.Printf("Access data at: http://localhost:%s/", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
