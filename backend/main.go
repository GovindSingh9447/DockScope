package main

import (
	"log"
	"net/http"
	//"time"

	"dockscope/backend/handlers"
	"dockscope/backend/middleware"
	"dockscope/backend/logger"
	"dockscope/backend/db"
)

func main() {
	mux := http.NewServeMux()

	// Core Docker endpoints
	mux.Handle("/containers", middleware.CORS(http.HandlerFunc(handlers.ListContainersHandler)))
	mux.Handle("/logs/", middleware.CORS(http.HandlerFunc(handlers.GetContainerLogsHandler)))
	mux.Handle("/wslogs", middleware.CORS(http.HandlerFunc(handlers.WSLogsHandler)))

	// Metrics from central server (GET) or agents (POST)
	mux.Handle("/metrics", middleware.CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetContainerMetricsHandler(w, r)
		case http.MethodPost:
			handlers.ReceiveAgentMetricsHandler(w, r)

			// ✅ Enhanced agent metrics log
			host := r.Header.Get("X-Host-ID")
			if host == "" {
				host = r.RemoteAddr
			}
			//now := time.Now().Format("15:04:05")

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	// Metrics WebSocket
	mux.Handle("/wsmetrics", middleware.CORS(http.HandlerFunc(handlers.WSContainerMetricsHandler)))

	// Export endpoints
	mux.Handle("/export/containers", middleware.CORS(http.HandlerFunc(handlers.ExportContainerListHandler)))
	mux.Handle("/export/logs", middleware.CORS(http.HandlerFunc(handlers.ExportContainerLogsHandler)))

	// Health Check
	mux.Handle("/health", middleware.CORS(http.HandlerFunc(handlers.HealthHandler)))

	// Alert management
	mux.Handle("/alerts", middleware.CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlers.CreateAlertRuleHandler(w, r)
		case http.MethodGet:
			handlers.ListAlertRulesHandler(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})))

	// New: Receive logs from agents

	/*
	mux.Handle("/agent/logs", middleware.CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlers.ReceiveAgentLogsHandler(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}))) */

	// UI static fallback
	mux.Handle("/ui/", middleware.CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "backend/index.html")
	})))

	// Start background tasks
	handlers.StartMonitoring()
	// handlers.LoadAlertsFromFile()
	db.InitDB()
	handlers.InitInflux()
	handlers.LoadAlertEventsFromFile()
	logger.InitLogger("whalewatch.log")

	port := ":9448"
	log.Printf("Server started on %s", port)
	log.Fatal(http.ListenAndServe(port, mux))
}

