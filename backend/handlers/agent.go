package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"fmt"
	"time"
)

// ReceiveAgentMetricsHandler stores metrics data received from agents
func ReceiveAgentMetricsHandler(w http.ResponseWriter, r *http.Request) {
	var payload AgentPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	PrintAgentMetricsLog(r, payload.HostID, len(payload.Containers))

	// ✅ Keep for alerts to work
	agentMetrics[payload.HostID] = payload.Containers

	for _, c := range payload.Containers {
		err := WriteMetricToInflux(
			payload.HostID,
			c.ID,
			c.Name,
			c.Image,
			c.CPUPercent,
			c.MemoryMB,
			c.RestartCount,
			time.Now(),
		)
		if err != nil {
			log.Printf("❌ Failed to write to InfluxDB for container %s: %v", c.ID, err)
			fmt.Fprintln(w, "Log handler not yet implemented")
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

