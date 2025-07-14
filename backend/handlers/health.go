package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/docker/docker/client"
	_ "github.com/lib/pq" // PostgreSQL driver
)

type HealthResponse struct {
	Status        string    `json:"status"`
	Time          time.Time `json:"time"`
	Message       string    `json:"message"`
	DockerHealthy bool      `json:"docker_healthy"`
	DBHealthy     bool      `json:"db_healthy"`
}

// HealthHandler performs health checks for Docker and DB
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	dockerHealthy := checkDocker(ctx)
	dbHealthy := checkDB()

	status := "ok"
	message := "Dockscope backend is healthy"
	if !dockerHealthy || !dbHealthy {
		status = "degraded"
		message = "One or more dependencies are unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	resp := HealthResponse{
		Status:        status,
		Time:          time.Now(),
		Message:       message,
		DockerHealthy: dockerHealthy,
		DBHealthy:     dbHealthy,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// checkDocker verifies Docker daemon is reachable
func checkDocker(ctx context.Context) bool {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return false
	}
	_, err = cli.Ping(ctx)
	return err == nil
}

// checkDB verifies database connectivity (PostgreSQL example)
func checkDB() bool {
	// You can also read this from env or config
	connStr := os.Getenv("DOCKSCOPE_DB_CONN")
	if connStr == "" {
		// Default fallback for local testing
		connStr = "postgres://user:password@localhost:5432/dbname?sslmode=disable"
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return false
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	return err == nil
}
