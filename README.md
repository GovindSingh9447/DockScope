# 🐳 DockScope

**DockScope** is a lightweight, agent-based Docker container monitoring and alerting system.  
It collects real-time metrics and logs from Docker containers across multiple hosts and visualizes them in a React dashboard.  
The backend is written in Go, with InfluxDB for time-series storage and SQLite for event persistence.

---

## 📁 Project Structure

```
DockScope/
├── agent/                          # Lightweight Go-based agent to run on each Docker host
│   ├── main.go                     # Agent code to collect & push metrics/logs
│   ├── Dockerfile                 # Docker image for the agent
│   └── docker-compose.agent.yml   # Compose file to run agent on remote nodes

├── backend/                        # Central server that receives metrics, serves APIs, alerts
│   ├── main.go                     # Main entrypoint of the Go backend
│   ├── Dockerfile                 # Docker image for backend (includes frontend dist)
│   ├── docker-compose.master.yml  # Compose file to run backend + InfluxDB
│   ├── handlers/                  # API endpoints (logs, metrics, containers, etc.)
│   ├── logger/                    # Custom logging setup
│   ├── logstore/                  # In-memory or file-based log store
│   ├── middleware/                # Middleware (e.g., CORS)
│   ├── utils/                     # Utility functions
│   ├── influx/                    # InfluxDB client wrapper
│   ├── db/                        # Database helpers and configuration
│   │   └── influxdb/              # InfluxDB engine files (ignored in Git)
│   └── data/                      # Predefined alert rules and event logs

├── frontend/                      # React-based UI (built with Vite)
│   └── index.html                 # Entry file (actual dist/ gets bundled in Docker)

├── docker-compose.master.yml      # (copied from backend/) Master compose for server deployment
├── docker-compose.agent.yml       # (copied from agent/) Agent compose for remote VMs
├── .gitignore                     # Files & folders excluded from Git
├── LICENSE                        # MIT License
├── README.md                      # You’re reading it!
├── go.mod / go.sum                # Go dependencies

```

---

## 🚀 Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/GovindSingh9447/DockScope.git
cd DockScope
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

| Method | Endpoint         | Description                    |
| ------ | ---------------- | ------------------------------ |
| GET    | `/containers`    | List running containers        |
| GET    | `/metrics`       | Real-time container metrics    |
| GET    | `/logs?id=<id>`  | Logs of specific container     |
| GET    | `/alerts`        | Get current alert rules/status |
| POST   | `/agent/metrics` | Agent sends metrics            |
| POST   | `/agent/logs`    | Agent sends logs               |

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
