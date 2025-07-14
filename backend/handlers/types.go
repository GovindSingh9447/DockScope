package handlers

import (
	"sync"
	"time"
)

// Core container metrics (used in dashboard, alerts, and agents)
type ContainerMetrics struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Image         string    `json:"image"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryMB      float64   `json:"memory_mb"`
	MemoryPercent float64   `json:"memory_percent"`
	Uptime        string    `json:"uptime"`
	Restart       string    `json:"restart"`
	RestartCount  int     `json:"restart_count"`
	CPUHistory    []float64 `json:"cpu_history"`
	MemoryHistory []float64 `json:"memory_history"`
	Logs          []string  `json:"logs,omitempty"`
	HostID        string    `json:"host_id"`
}

// Alert definition structure (defined by user)
type AlertDefinition struct {
	ID           string  `json:"id"`
	HostID       string  `json:"host_id"`
	Type         string  `json:"type"` // e.g., high_cpu, high_memory, log_pattern
	Threshold    float64 `json:"threshold"`
	Pattern      string  `json:"pattern"`
	ContainerID  string  `json:"container_id"`
	Enabled      bool    `json:"enabled"`
	SlackWebhook string  `json:"slack_webhook"`
	Email        string  `json:"email"`
	AutoRestart  bool    `json:"auto_restart"` // Not used anymore
	AutoStop     bool    `json:"auto_stop"`    // Not used anymore
}

// Triggered alert event (recorded in memory + file)
type AlertEvent struct {
	AlertID     string    `json:"alert_id"`
	HostID      string    `json:"host_id"`
	ContainerID string    `json:"container_id"`
	Type        string    `json:"type"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	Restarted   bool      `json:"restarted"` // Now always false (optional)
}

// Agent push payloads (metrics)
type AgentPayload struct {
	HostID     string             `json:"host_id"`
	Timestamp  string             `json:"timestamp"`
	Containers []ContainerMetrics `json:"containers"`
}

// Agent push payloads (logs)
type AgentLogPayload struct {
	HostID      string    `json:"host_id"`
	ContainerID string    `json:"container_id"`
	Logs        string    `json:"logs"`
	Timestamp   time.Time `json:"timestamp"`
}

// Simple alert rule format for local checks (used in monitor.go)
type AlertRule struct {
	ID           string
	HostID       string
	ContainerID  string
	Type         string  // "high_cpu", "high_memory", "log_pattern"
	Threshold    float64
	Pattern      string
	Email        string
	SlackWebhook string
	AutoRestart  bool // deprecated
	AutoStop     bool // deprecated
	Enabled      bool
}

// For raw listing or UI
type ContainerInfo struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	CPUPercent float64  `json:"cpu_percent"`
	MemoryMB   float64  `json:"memory_mb"`
	Logs       []string `json:"logs,omitempty"`
	Image      string   `json:"image"`
	HostID     string   `json:"host_id,omitempty"`
}

// 🔁 Shared in-memory variables
var (
	alertRules      []AlertRule
	alertsMutex     = &sync.RWMutex{}
	alertEvents     []AlertEvent
	alertDefinitions []AlertDefinition
	agentMetrics    = make(map[string][]ContainerMetrics)
)

const (
	HighCPU    = "high_cpu"
	HighMemory = "high_memory"
	LogPattern = "log_pattern"
)


