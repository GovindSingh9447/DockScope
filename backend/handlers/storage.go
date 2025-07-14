package handlers

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

const alertEventsFile = "data/alert_events.json"

// SaveAlertEventsToFile saves alert events to disk
func SaveAlertEventsToFile() {
	data, err := json.MarshalIndent(alertEvents, "", "  ")
	if err != nil {
		log.Println("[ERROR] Failed to marshal alert events:", err)
		return
	}

	// Ensure data/ directory exists
	if err := os.MkdirAll(filepath.Dir(alertEventsFile), 0755); err != nil {
		log.Println("[ERROR] Failed to create data directory:", err)
		return
	}

	if err := ioutil.WriteFile(alertEventsFile, data, 0644); err != nil {
		log.Println("[ERROR] Failed to write alert events:", err)
	}
}

// LoadAlertEventsFromFile loads alert events from disk
func LoadAlertEventsFromFile() {
	file, err := os.Open(alertEventsFile)
	if err != nil {
		if os.IsNotExist(err) {
			alertEvents = []AlertEvent{}
			return
		}
		log.Println("[ERROR] Failed to open alert events file:", err)
		return
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println("[ERROR] Failed to read alert events file:", err)
		return
	}

	if err := json.Unmarshal(data, &alertEvents); err != nil {
		log.Println("[ERROR] Failed to unmarshal alert events:", err)
	}
}
