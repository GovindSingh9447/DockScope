package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types/container"
)

// StartMonitoring starts background monitoring
func StartMonitoring() {
	go monitorLoop()
}

func monitorLoop() {
	for {
		checkAllAlerts()
		time.Sleep(10 * time.Second) // Polling interval
	}
}

func checkAllAlerts() {
	alertsMutex.RLock()
	rulesCopy := make([]AlertRule, len(alertRules))
	copy(rulesCopy, alertRules)
	alertsMutex.RUnlock()

	for _, rule := range rulesCopy {
		if !rule.Enabled {
			continue
		}

		switch rule.Type {
		case HighCPU, HighMemory:
			checkContainerResource(rule)
		case LogPattern:
			checkLogPattern(rule)
		}
	}
}

func checkContainerResource(rule AlertRule) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Printf("Error creating Docker client: %v", err)
		return
	}
	defer cli.Close()

	stats, err := cli.ContainerStats(ctx, rule.ContainerID, false)
	if err != nil {
		log.Printf("Failed to fetch container stats for %s: %v", rule.ContainerID, err)
		return
	}
	defer stats.Body.Close()

	var containerStats types.StatsJSON
	if err := json.NewDecoder(stats.Body).Decode(&containerStats); err != nil {
		log.Printf("Error decoding stats JSON: %v", err)
		return
	}

	cpuPercent := calculateCPUPercent(containerStats)
	memUsage := float64(containerStats.MemoryStats.Usage) / (1024 * 1024) // in MB

	if rule.Type == HighCPU && cpuPercent > rule.Threshold {
		sendAlert(rule, "High CPU usage: "+strconv.FormatFloat(cpuPercent, 'f', 2, 64)+"%")
	}
	if rule.Type == HighMemory && memUsage > rule.Threshold {
		sendAlert(rule, "High Memory usage: "+strconv.FormatFloat(memUsage, 'f', 2, 64)+" MB")
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

func checkLogPattern(rule AlertRule) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Printf("Error creating Docker client: %v", err)
		return
	}
	defer cli.Close()

	logOptions := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "100",
	}

	out, err := cli.ContainerLogs(ctx, rule.ContainerID, logOptions)
	if err != nil {
		log.Printf("Failed to fetch logs: %v", err)
		return
	}
	defer out.Close()

	logBytes, err := io.ReadAll(out)
	if err != nil {
		log.Printf("Failed to read logs: %v", err)
		return
	}
	logs := string(logBytes)

	if strings.Contains(logs, rule.Pattern) {
		sendAlert(rule, "Log pattern matched: '"+rule.Pattern+"' found")
	}
}

func sendAlert(rule AlertRule, message string) {
	log.Printf("[ALERT] %s => %s", rule.ContainerID, message)

	if rule.SlackWebhook != "" {
		sendSlackNotification(rule.SlackWebhook, message)
	}

	if rule.Email != "" {
		err := SendEmailNotification(
			rule.Email,
			"[DockScope Alert] "+string(rule.Type),
			fmt.Sprintf("Container: %s triggered %s.\nMessage: %s\nThreshold: %.2f",
				rule.ContainerID, rule.Type, message, rule.Threshold),
		)
		if err != nil {
			log.Printf("Failed to send email alert: %v", err)
		}
	}

	go handleAutoRecovery(rule)
}

func handleAutoRecovery(rule AlertRule) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Printf("AutoRecovery: Failed to create Docker client: %v", err)
		return
	}
	defer cli.Close()

	ctx := context.Background()

	if rule.AutoRestart {
		log.Printf("[AutoRecovery] Restarting container: %s", rule.ContainerID)
		if err := cli.ContainerRestart(ctx, rule.ContainerID, container.StopOptions{}); err != nil {
			log.Printf("[AutoRecovery] Failed to restart container: %v", err)
		}
	}

	if rule.AutoStop {
		log.Printf("[AutoRecovery] Stopping container: %s", rule.ContainerID)
		if err := cli.ContainerStop(ctx, rule.ContainerID, container.StopOptions{}); err != nil {
			log.Printf("[AutoRecovery] Failed to stop container: %v", err)
		}
	}
}




// Placeholder for Slack alert
func sendSlackNotification(webhookURL string, message string) {
	log.Printf("[Slack Alert] Webhook: %s | Message: %s", webhookURL, message)
	// TODO: Send actual POST request
}

// Placeholder for SMTP or email service
func sendEmailNotification(email string, message string) {
	log.Printf("[Email Alert] To: %s | Message: %s", email, message)
}
