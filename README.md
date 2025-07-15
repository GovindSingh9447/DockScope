# 🐳 DockScope

**DockScope** is a lightweight, agent-based Docker container monitoring and alerting system.  
It collects real-time metrics and logs from Docker containers across multiple hosts and visualizes them in a React dashboard.  
The backend is written in Go, with InfluxDB for time-series storage and SQLite for event persistence.

---

## 📁 Project Structure

```
dockscope/
├── agent/            # Agent service running on Docker hosts to collect metrics/logs
├── backend/          # Go backend server with REST APIs and alert engine
│   ├── handlers/     # HTTP API handlers (metrics, logs, alerts)
│   ├── db/           # SQLite and InfluxDB storage
│   ├── data/         # Static alert rules and fired events
│   ├── logger/       # Centralized structured logging
│   ├── logstore/     # In-memory and persisted logs
│   └── main.go       # Backend entrypoint
├── frontend/         # React frontend dashboard (Vite/CRA)
├── go.mod / go.sum   # Go module dependencies
```

---

## 🚀 Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/GovindSingh9447/dockscope.git
cd dockscope
```

---

### 2. Backend Setup (Go + InfluxDB + SQLite)

```bash
cd backend
go mod tidy
go run main.go
```

Make sure InfluxDB and metrics.db are correctly initialized in `backend/db/`.

---

### 3. Agent Setup

Run this on each Docker host you want to monitor:

```bash
cd agent
go run main.go
```

The agent will collect container metrics/logs and send them to the backend.

---

### 4. Frontend Setup (React)

If your teammate has already added the frontend:

```bash
cd frontend
npm install
npm run dev
```

If you need to create it:

```bash
npx create-vite@latest frontend --template react
cd frontend
npm install
npm run dev
```

---

## 🌐 API Endpoints

| Method | Endpoint                | Description                      |
|--------|-------------------------|----------------------------------|
| GET    | `/containers`           | List running containers          |
| GET    | `/metrics`              | Real-time container metrics      |
| GET    | `/logs?id=<id>`         | Logs of specific container       |
| GET    | `/alerts`               | Get current alert rules/status   |
| POST   | `/agent/metrics`        | Agent sends metrics              |
| POST   | `/agent/logs`           | Agent sends logs                 |

---

## 📊 Data Storage

- **SQLite** — Stores alerts, triggered events
- **InfluxDB** — Time-series metrics
- **In-memory** — Cached logs and real-time data

---

## 🛠 Tech Stack

- **Backend:** Go (Golang)
- **Frontend:** React (Vite)
- **Databases:** SQLite + InfluxDB
- **Agents:** Lightweight Go binaries running per host

---

## 👨‍💻 Contributors

- **Sahana Naveen** — Frontend, UI/UX
- **Govind Singh** — Backend, Architecture 
---

## 🛡 License

MIT License © 2025 Govind Singh

---

> Light. Fast. Docker-native monitoring. No Prometheus required.
