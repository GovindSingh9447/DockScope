package db

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB() {
	var err error

	// Create data directory if not exists
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		os.Mkdir("data", 0755)
	}

	DB, err = sql.Open("sqlite3", "./data/metrics.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// Create table if not exists
	createStmt := `
	CREATE TABLE IF NOT EXISTS container_metrics (
		id TEXT,
		name TEXT,
		image TEXT,
		host_id TEXT,
		cpu_percent REAL,
		memory_mb REAL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = DB.Exec(createStmt)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
}

