package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"io"
	"os"
	"net/http"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// ExportContainerListHandler writes the list of running containers as JSON to a downloadable file
func ExportContainerListHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("Docker client error: %v", err)
		http.Error(w, "Docker client error", http.StatusInternalServerError)
		return
	}

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		log.Printf("Error listing containers: %v", err)
		http.Error(w, "Failed to list Docker containers", http.StatusInternalServerError)
		return
	}

	var result []ContainerInfo
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0]
			if name[0] == '/' {
				name = name[1:]
			}
		}
		result = append(result, ContainerInfo{
			ID:    c.ID[:12],
			Name:  name,
			Image: c.Image,
		})
	}

	w.Header().Set("Content-Disposition", "attachment; filename=containers.json")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ExportContainerLogsHandler writes the logs of a container to a downloadable .txt file
func ExportContainerLogsHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	name := r.URL.Query().Get("name")
	if id == "" || name == "" {
		http.Error(w, "Missing container id or name", http.StatusBadRequest)
		return
	}

	logFilePath := fmt.Sprintf("/tmp/%s.log", id) // or wherever you save logs
	file, err := os.Open(logFilePath)
	if err != nil {
		http.Error(w, "Log file not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	timestamp := time.Now().Format("1504") // HHMM format
	filename := fmt.Sprintf("%s_%s.txt", name, timestamp)

	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", "text/plain")

	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, "Failed to send file", http.StatusInternalServerError)
		return
	}
}
