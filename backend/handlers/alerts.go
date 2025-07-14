package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"dockscope/backend/logger"
)

// CreateAlertRuleHandler handles creation of new alert rules
func CreateAlertRuleHandler(w http.ResponseWriter, r *http.Request) {
	var rule AlertDefinition
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if rule.ID == "" || rule.ContainerID == "" || rule.Type == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	alertsMutex.Lock()
	alertDefinitions = append(alertDefinitions, rule)
	alertsMutex.Unlock()

	SaveAlertRulesToFile("data/alert_rules.json", alertRules)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Alert rule created"))
}

// ListAlertRulesHandler returns current alert rules
func ListAlertRulesHandler(w http.ResponseWriter, r *http.Request) {
	alertsMutex.RLock()
	defer alertsMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alertRules)
}

func SaveAlertRulesToFile(filename string, rules []AlertRule) {
	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		log.Printf("[ERROR] Failed to marshal alert rules: %v\n", err)
		return
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		log.Printf("[ERROR] Failed to write alert rules to file: %v\n", err)
	}
}

// EvaluateAlerts checks if any alert rules are triggered by the incoming metrics
func EvaluateAlerts(payload AgentPayload) {
	for _, container := range payload.Containers {
		for _, alert := range alertDefinitions {
			if !alert.Enabled || alert.HostID != payload.HostID || alert.ContainerID != container.ID {
				continue
			}

			triggered := false
			message := ""

			avg, err := QueryAverageMetric(alert.Type, container.ID, payload.HostID, 5*time.Minute)
			if err != nil {
				logger.Error("Failed to fetch metric from InfluxDB: %v", err)
				continue
			}

			switch alert.Type {
			case "high_cpu":
				if avg > alert.Threshold {
					triggered = true
					message = "CPU usage exceeded threshold"
				}
			case "high_memory":
				if avg > alert.Threshold {
					triggered = true
					message = "Memory usage exceeded threshold"
				}
			}

			if triggered {
				logger.Warn("[ALERT] %s (%s) triggered on container %s: %s", alert.Type, alert.ID, container.ID, message)

				event := AlertEvent{
					AlertID:     alert.ID,
					HostID:      payload.HostID,
					ContainerID: container.ID,
					Type:        alert.Type,
					Message:     message,
					Timestamp:   time.Now(),
				}

				alertEvents = append(alertEvents, event)
				SaveAlertEventsToFile()

				if alert.Email != "" {
					subject := "[DockScope Alert] " + alert.Type + " on " + container.Name
					body := "Alert ID: " + alert.ID + "\nContainer: " + container.Name + "\nType: " + alert.Type + "\n\nMessage:\n" + message
					go SendEmailNotification(alert.Email, subject, body)
				}
			}
		}
	}
}

