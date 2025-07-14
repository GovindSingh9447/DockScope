package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gorilla/websocket"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorCyan   = "\033[36m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
)

func PrintAgentMetricsLog(r *http.Request, hostID string, count int) {
	now := time.Now()
	fmt.Printf(ColorCyan + "\n🧠 [Agent Metrics]\n" + ColorReset)
	fmt.Printf(ColorGreen+" ├─ Host     : %s\n"+ColorReset, hostID)
	fmt.Printf(ColorYellow+" ├─ IP:Port  : %s\n"+ColorReset, r.RemoteAddr)
	fmt.Printf(" ├─ Time     : %s\n", now.Format("15:04:05"))
	fmt.Printf(" ├─ Containers: %d\n", count)
	fmt.Printf(" └─ Endpoint : %s\n", r.URL.Path)
}

func colorize(label, value, color string) string {
	return fmt.Sprintf("%s%s%s%s", color, label, ColorReset, value)
}

// Struct to return current container metrics
/*
type ContainerMetrics struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Uptime        string    `json:"uptime"`
	Restart       string    `json:"restart"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryMB      float64   `json:"memory_mb"`
	CPUHistory    []float64 `json:"cpu_history"`
	MemoryHistory []float64 `json:"memory_history"`
	HostID        string    `json:"host_id"`
} */

// Handler for fetching single container metrics (for UI or API)
func GetContainerMetricsHandler(w http.ResponseWriter, r *http.Request) {
	containerID := r.URL.Query().Get("id")
	if containerID == "" {
		http.Error(w, "Container ID is required", http.StatusBadRequest)
		return
	}

	hostID := r.URL.Query().Get("host_id")
	if hostID == "" {
		hostID = "master"
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Println(ColorRed, "[ERROR] Docker client error:", err, ColorReset)
		http.Error(w, "Docker client error", http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats, err := cli.ContainerStatsOneShot(ctx, containerID)
	if err != nil {
		log.Println(ColorRed, "[ERROR] Error fetching stats:", err, ColorReset)
		http.Error(w, "Failed to get stats", http.StatusInternalServerError)
		return
	}
	defer stats.Body.Close()

	var data types.StatsJSON
	if err := json.NewDecoder(stats.Body).Decode(&data); err != nil {
		log.Println(ColorRed, "[ERROR] Stats decode error:", err, ColorReset)
		http.Error(w, "Failed to parse stats", http.StatusInternalServerError)
		return
	}

	cpuDelta := float64(data.CPUStats.CPUUsage.TotalUsage - data.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(data.CPUStats.SystemUsage - data.PreCPUStats.SystemUsage)
	cpuPercent := 0.0
	if systemDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(len(data.CPUStats.CPUUsage.PercpuUsage)) * 100
	}
	if cpuPercent < 0 || cpuPercent > 1000 {
		cpuPercent = 0
	}

	memUsed := data.MemoryStats.Usage - data.MemoryStats.Stats["cache"]
	memUsage := float64(memUsed) / (1024 * 1024)

	info, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		log.Println(ColorRed, "[ERROR] Container inspect error:", err, ColorReset)
		http.Error(w, "Failed to inspect container", http.StatusInternalServerError)
		return
	}

	startTime, _ := time.Parse(time.RFC3339Nano, info.State.StartedAt)
	uptime := time.Since(startTime).Round(time.Second).String()

	restartTime, _ := time.Parse(time.RFC3339Nano, info.State.FinishedAt)
	if restartTime.Before(startTime) {
		restartTime = startTime
	}

	log.Printf("%s[Metrics]%s Host: %s | Container: %s | CPU: %.2f%% | MEM: %.2fMB | Uptime: %s",
		ColorBlue, ColorReset, hostID, info.Name, cpuPercent, memUsage, uptime)

	// Push to InfluxDB
	go func() {
		err := WriteMetricToInflux(
			hostID,
			containerID,
			info.Name,
			info.Config.Image,
			cpuPercent,
			memUsage,
			info.RestartCount,
			time.Now().UTC(),
		)
		if err != nil {
			log.Println(ColorRed, "[Influx ERROR] Failed to write metrics:", err, ColorReset)
		}
	}()

	resp := ContainerMetrics{
		ID:            containerID,
		Name:          info.Name,
		Uptime:        uptime,
		Restart:       restartTime.Format(time.RFC3339),
		CPUPercent:    cpuPercent,
		MemoryMB:      memUsage,
		CPUHistory:    []float64{cpuPercent},
		MemoryHistory: []float64{memUsage},
		HostID:        hostID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// WebSocket payload format
type MetricsPayload struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsage   uint64  `json:"memory_usage"`
	MemoryLimit   uint64  `json:"memory_limit"`
	MemoryPercent float64 `json:"memory_percent"`
	NetRx         uint64  `json:"network_rx"`
	NetTx         uint64  `json:"network_tx"`
	DiskRead      uint64  `json:"disk_read"`
	DiskWrite     uint64  `json:"disk_write"`
}

var metricsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Handler for WebSocket live metrics + Influx logging
func WSContainerMetricsHandler(w http.ResponseWriter, r *http.Request) {
	containerID := r.URL.Query().Get("id")
	if containerID == "" {
		http.Error(w, "Container ID is required", http.StatusBadRequest)
		return
	}

	hostID := r.URL.Query().Get("host_id")
	if hostID == "" {
		hostID = "master"
	}

	conn, err := metricsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(ColorRed, "[ERROR] WebSocket upgrade failed:", err, ColorReset)
		return
	}
	defer conn.Close()

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Println(ColorRed, "[ERROR] Docker client error:", err, ColorReset)
		return
	}

	info, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		log.Println(ColorRed, "[ERROR] Container inspect error:", err, ColorReset)
		return
	}

	for {
		stats, err := cli.ContainerStatsOneShot(ctx, containerID)
		if err != nil {
			log.Println(ColorRed, "[ERROR] Stats fetch error:", err, ColorReset)
			break
		}

		var data types.StatsJSON
		if err := json.NewDecoder(stats.Body).Decode(&data); err != nil {
			log.Println(ColorRed, "[ERROR] Stats decode error:", err, ColorReset)
			break
		}
		stats.Body.Close()

		cpuDelta := float64(data.CPUStats.CPUUsage.TotalUsage - data.PreCPUStats.CPUUsage.TotalUsage)
		systemDelta := float64(data.CPUStats.SystemUsage - data.PreCPUStats.SystemUsage)
		cpuPercent := 0.0
		if systemDelta > 0 && cpuDelta > 0 {
			cpuPercent = (cpuDelta / systemDelta) * float64(len(data.CPUStats.CPUUsage.PercpuUsage)) * 100
		}
		if cpuPercent < 0 || cpuPercent > 1000 {
			cpuPercent = 0
		}

		memUsed := data.MemoryStats.Usage - data.MemoryStats.Stats["cache"]
		memPercent := float64(memUsed) / float64(data.MemoryStats.Limit) * 100

		netRx, netTx := uint64(0), uint64(0)
		for _, v := range data.Networks {
			netRx += v.RxBytes
			netTx += v.TxBytes
		}

		var diskRead, diskWrite uint64
		for _, block := range data.BlkioStats.IoServiceBytesRecursive {
			switch block.Op {
			case "Read":
				diskRead += block.Value
			case "Write":
				diskWrite += block.Value
			}
		}

		log.Printf("%s[WS Metrics]%s Container: %s | CPU: %.2f%% | MEM: %.2fMB | MEM%%: %.2f%% | RX: %d | TX: %d | RD: %d | WR: %d",
			ColorGreen, ColorReset,
			containerID[:12], cpuPercent, float64(memUsed)/(1024*1024), memPercent, netRx, netTx, diskRead, diskWrite)

		payload := MetricsPayload{
			CPUPercent:    cpuPercent,
			MemoryUsage:   memUsed,
			MemoryLimit:   data.MemoryStats.Limit,
			MemoryPercent: memPercent,
			NetRx:         netRx,
			NetTx:         netTx,
			DiskRead:      diskRead,
			DiskWrite:     diskWrite,
		}

		if err := conn.WriteJSON(payload); err != nil {
			log.Println(ColorRed, "[ERROR] WebSocket write failed:", err, ColorReset)
			break
		}

		// Send to InfluxDB
		go func() {
			err := WriteMetricToInflux(
				hostID,
				containerID,
				info.Name,
				info.Config.Image,
				cpuPercent,
				float64(memUsed)/(1024*1024),
				info.RestartCount,
				time.Now().UTC(),
			)
			if err != nil {
				log.Println(ColorRed, "[Influx ERROR] WS metrics write failed:", err, ColorReset)
			}
		}()

		time.Sleep(2 * time.Second)
	}
}

