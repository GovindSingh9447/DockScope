package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
	"bufio"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gorilla/websocket"

)

// GetContainerLogsHandler returns recent logs for a container (last 500 lines).
func GetContainerLogsHandler(w http.ResponseWriter, r *http.Request) {
	containerID := r.URL.Query().Get("id")
	if containerID == "" {
		http.Error(w, "Container ID is required", http.StatusBadRequest)
		return
	}

	search := r.URL.Query().Get("search")
	sinceStr := r.URL.Query().Get("since")
	untilStr := r.URL.Query().Get("until")

	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page := 1
	limit := 100

	if pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
		if page < 1 {
			page = 1
		}
	}
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
		if limit < 1 {
			limit = 100
		}
	}

	sinceTime := time.Time{}
	untilTime := time.Now()

	if sinceStr != "" {
		t, err := time.Parse(time.RFC3339, sinceStr)
		if err == nil {
			sinceTime = t
		}
	}

	if untilStr != "" {
		t, err := time.Parse(time.RFC3339, untilStr)
		if err == nil {
			untilTime = t
		}
	}

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		http.Error(w, "Could not connect to Docker daemon", http.StatusInternalServerError)
		return
	}

	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Follow:     false,
		Tail:       "1000", // Fetch extra logs to filter/paginate later
	}

	reader, err := cli.ContainerLogs(ctx, containerID, options)
	if err != nil {
		http.Error(w, "Could not get container logs", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "text/plain")

	scanner := bufio.NewScanner(reader)
	var filtered []string

	for scanner.Scan() {
		raw := scanner.Text()

		// Separate timestamp and message
		parts := strings.SplitN(raw, " ", 2)
		if len(parts) < 2 {
			continue
		}

		timestampStr := parts[0]
		message := parts[1]

		t, err := time.Parse(time.RFC3339Nano, timestampStr)
		if err != nil {
			continue
		}

		if t.Before(sinceTime) || t.After(untilTime) {
			continue
		}

		if search != "" && !strings.Contains(strings.ToLower(message), strings.ToLower(search)) {
			continue
		}

		filtered = append(filtered, raw)
	}

	// Pagination
	start := (page - 1) * limit
	end := start + limit
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}

	paged := filtered[start:end]

	for _, line := range paged {
		fmt.Fprintln(w, line)
	}
}


// Upgrader for WebSocket connections
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WSLogsHandler streams logs to the frontend over WebSocket.
func WSLogsHandler(w http.ResponseWriter, r *http.Request) {
	containerID := r.URL.Query().Get("id")
	filter := r.URL.Query().Get("filter")

	if containerID == "" {
		http.Error(w, "Container ID is required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Docker client error"))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logOptions := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: false,
		Tail:       "100",
	}

	reader, err := cli.ContainerLogs(ctx, containerID, logOptions)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Log stream error"))
		return
	}
	defer reader.Close()

	for {
		// Docker log multiplexed header is 8 bytes
		header := make([]byte, 8)
		_, err := io.ReadFull(reader, header)
		if err != nil {
			break // done reading
		}

		frameSize := int(header[4])<<24 | int(header[5])<<16 | int(header[6])<<8 | int(header[7])
		if frameSize <= 0 {
			continue
		}

		frame := make([]byte, frameSize)
		_, err = io.ReadFull(reader, frame)
		if err != nil {
			break
		}

		line := string(frame)

		if filter != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(filter)) {
			continue
		}

		if err := conn.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
			break
		}
	}
}

/*
func SearchLogsHandler(w http.ResponseWriter, r *http.Request) {
	containerID := r.URL.Query().Get("id")
	since := r.URL.Query().Get("since")
	until := r.URL.Query().Get("until")

	if containerID == "" {
		http.Error(w, "Container ID is required", http.StatusBadRequest)
		return
	}

	var sinceTime, untilTime time.Time
	var err error

	if since != "" {
		sinceTime, err = time.Parse(time.RFC3339, since)
		if err != nil {
			http.Error(w, "Invalid 'since' time format", http.StatusBadRequest)
			return
		}
	} else {
		sinceTime = time.Now().Add(-1 * time.Hour)
	}

	if until != "" {
		untilTime, err = time.Parse(time.RFC3339, until)
		if err != nil {
			http.Error(w, "Invalid 'until' time format", http.StatusBadRequest)
			return
		}
	} else {
		untilTime = time.Now()
	}

	logs, err := logstore.QueryLogs(containerID, sinceTime, untilTime)
	if err != nil {
		http.Error(w, "Failed to query logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
	*/

func sanitizeLog(raw string) string {
	if len(raw) >= 8 && (raw[0] < 32) {
		return raw[8:]
	}
	return raw
}

