// File: backend/logstore/logstore.go
package logstore

import (
	"database/sql"
	"log"
	"os"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func InitDB() {
	var err error
	if _, err := os.Stat("logs.db"); os.IsNotExist(err) {
		os.Create("logs.db")
	}
	db, err = sql.Open("sqlite3", "logs.db")
	if err != nil {
		log.Fatal(err)
	}

	createTable := `CREATE TABLE IF NOT EXISTS container_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		container_id TEXT,
		level TEXT,
		message TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}
}

func SaveLog(containerID, level, message string) {
	stmt, err := db.Prepare("INSERT INTO container_logs(container_id, level, message) VALUES (?, ?, ?)")
	if err != nil {
		log.Println("Prepare failed:", err)
		return
	}
	defer stmt.Close()
	_, err = stmt.Exec(containerID, level, message)
	if err != nil {
		log.Println("Exec failed:", err)
	}
}

func QueryLogsByDate(start, end string) ([]map[string]string, error) {
	rows, err := db.Query(`SELECT container_id, level, message, timestamp FROM container_logs WHERE timestamp BETWEEN ? AND ? ORDER BY timestamp`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []map[string]string
	for rows.Next() {
		var cid, level, msg, ts string
		if err := rows.Scan(&cid, &level, &msg, &ts); err != nil {
			return nil, err
		}
		logs = append(logs, map[string]string{
			"container_id": cid,
			"level":        level,
			"message":      msg,
			"timestamp":    ts,
		})
	}
	return logs, nil
}
