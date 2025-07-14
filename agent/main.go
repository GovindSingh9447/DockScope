package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const (
	centralServerURL = "http://13.203.202.82:9448/metrics"
	authToken        = "your-secret-token"
)

type ContainerMetrics struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Image  string  `json:"image"`
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
}

type AgentPayload struct {
	HostID     string             `json:"host_id"`
	Containers []ContainerMetrics `json:"containers"`
	Timestamp  string             `json:"timestamp"`
}

func main() {
	hostID := os.Getenv("HOST_ID")
	if hostID == "" {
		hostID = getLocalIP() // fallback to public IP detection
	}
	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		serverURL = centralServerURL
	}

	go startLogServer()

	for {
		payload := collectMetrics(hostID)
		log.Printf("Collected %d containers from Host: %s", len(payload.Containers), hostID)
		pushToServer(serverURL, payload)
		time.Sleep(10 * time.Second)
	}
}

func collectMetrics(hostID string) AgentPayload {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: false})
	if err != nil {
		log.Printf("Failed to list containers: %v", err)
		return AgentPayload{HostID: hostID}
	}

	var metrics []ContainerMetrics
	for _, container := range containers {
		stats, err := cli.ContainerStats(ctx, container.ID, false)
		if err != nil {
			log.Printf("Failed to get stats for container %s: %v", container.ID, err)
			continue
		}

		var containerStats types.StatsJSON
		if err := json.NewDecoder(stats.Body).Decode(&containerStats); err != nil {
			log.Printf("Failed to decode stats JSON for container %s: %v", container.ID, err)
			stats.Body.Close()
			continue
		}
		stats.Body.Close()

		cpu := calculateCPUPercent(containerStats)
		mem := float64(containerStats.MemoryStats.Usage) / (1024 * 1024)
		name := strings.TrimPrefix(container.Names[0], "/")

		log.Printf("Container: %s | CPU: %.2f%% | MEM: %.2fMB | IMAGE: %s", name, cpu, mem, container.Image)

		metrics = append(metrics, ContainerMetrics{
			ID:     container.ID,
			Name:   name,
			Image:  container.Image,
			CPU:    cpu,
			Memory: mem,
		})
	}

	return AgentPayload{
		HostID:     hostID,
		Containers: metrics,
		Timestamp:  time.Now().Format(time.RFC3339),
	}
}

func calculateCPUPercent(stats types.StatsJSON) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	if systemDelta > 0 && cpuDelta > 0 {
		return (cpuDelta / systemDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return 0.0
}

func pushToServer(url string, payload AgentPayload) {
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal payload: %v", err)
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Failed to send request: %v", err)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("Sent metrics to master. Host: %s | Response: %s", payload.HostID, string(respBody))
}

// ============ LOG SERVER ============

func startLogServer() {
	http.HandleFunc("/logs", logHandler)
	log.Println("Agent log API running on :8880")
	log.Fatal(http.ListenAndServe(":8880", nil))
}

func logHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+authToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	containerID := r.URL.Query().Get("id")
	if containerID == "" {
		http.Error(w, "Missing container ID", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		http.Error(w, "Docker client error", http.StatusInternalServerError)
		return
	}
	defer cli.Close()

	reader, err := cli.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "100",
	})
	if err != nil {
		http.Error(w, "Failed to fetch logs: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "text/plain")
	io.Copy(w, reader)
}

// ============ UTILS ============

func getLocalIP() string {
	resp, err := http.Get("http://checkip.amazonaws.com")
	if err != nil {
		log.Println("Could not fetch public IP, using fallback.")
		return "default-host"
	}
	defer resp.Body.Close()

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "default-host"
	}
	return strings.TrimSpace(string(ip))
}

