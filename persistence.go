package main

import (
	"log"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

var writeFileMutex sync.Mutex

// startDataWriter starts a background worker that debounces and batches writes to disk
// This avoids excessive I/O operations during rapid updates
func startDataWriter(data *Data) {
	const debounceDelay = 2 * time.Second
	var debounceTimer *time.Timer
	var timerActive bool

	log.Println("data writer started")

	for {
		select {
		case dataToWrite := <-dataWriteQueue:
			// Immediate write requested (bypass debounce)
			if err := writeDataFile(dataToWrite); err != nil {
				log.Printf("error writing data file (immediate): %v", err)
			} else {
				log.Println("data file written successfully (immediate)")
			}
			// Clear dirty flag since we just wrote
			dataDirtyMutex.Lock()
			dataDirty = false
			dataDirtyMutex.Unlock()

			// Cancel any pending debounced write
			if debounceTimer != nil {
				debounceTimer.Stop()
				timerActive = false
			}

		default:
			// Check if data is dirty and needs debounced write
			dataDirtyMutex.Lock()
			isDirty := dataDirty
			dataDirtyMutex.Unlock()

			if isDirty && !timerActive {
				// Start the debounce timer only if not already active
				log.Println("starting new debounce timer (2 seconds)")
				timerActive = true
				debounceTimer = time.AfterFunc(debounceDelay, func() {
					log.Println("debounce timer fired, checking if still dirty")
					dataDirtyMutex.Lock()
					if dataDirty {
						dataDirty = false
						dataDirtyMutex.Unlock()

						log.Println("writing data file (debounced)")
						// Write to file
						if err := writeDataFile(data); err != nil {
							log.Printf("error writing data file: %v", err)
						} else {
							log.Println("data file written successfully (debounced)")
						}
					} else {
						dataDirtyMutex.Unlock()
						log.Println("data no longer dirty, skipping write")
					}
					timerActive = false
				})
			}

			// Small sleep to avoid busy loop
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// writeDataFile writes the data to the YAML file
func writeDataFile(data *Data) error {
	// Prevent concurrent file writes
	writeFileMutex.Lock()
	defer writeFileMutex.Unlock()

	f, err := os.Create(dataFile)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := yaml.NewEncoder(f)
	if err := encoder.Encode(data); err != nil {
		return err
	}

	// Explicitly close the encoder to flush buffers
	if err := encoder.Close(); err != nil {
		return err
	}

	// Sync to ensure data is written to disk
	if err := f.Sync(); err != nil {
		return err
	}

	return nil
}

// markDataDirty marks that the data has been modified and needs to be written
func markDataDirty() {
	dataDirtyMutex.Lock()
	dataDirty = true
	dataDirtyMutex.Unlock()
	log.Println("data marked dirty - will write in 2 seconds")
}

// requestImmediateWrite requests an immediate write of the data to disk
func requestImmediateWrite(data *Data) {
	select {
	case dataWriteQueue <- data:
		log.Println("immediate data write requested")
	default:
		log.Println("write queue full, marking dirty instead")
		markDataDirty()
	}
}
