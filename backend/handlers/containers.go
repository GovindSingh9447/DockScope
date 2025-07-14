package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func ListContainersHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		http.Error(w, "Docker client error", http.StatusInternalServerError)
		log.Printf("Docker client error: %v", err)
		return
	}

	// Query Params
	hostIDsFilter := strings.Split(r.URL.Query().Get("host_id"), ",")
	nameFilter := strings.ToLower(r.URL.Query().Get("name"))
	imageFilter := strings.ToLower(r.URL.Query().Get("image"))
	labelFilter := strings.ToLower(r.URL.Query().Get("label"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	sortBy := r.URL.Query().Get("sort_by")   // name, cpu, memory
	order := r.URL.Query().Get("order")      // asc or desc

	var results []ContainerInfo

	// --- Master containers ---
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		http.Error(w, "Failed to list containers", http.StatusInternalServerError)
		log.Printf("List error: %v", err)
		return
	}

	for _, c := range containers {
		name := strings.TrimPrefix(c.Names[0], "/")
		info := ContainerInfo{
			ID:     c.ID[:12],
			Name:   name,
			Image:  c.Image,
			HostID: "master",
		}
		if matchesFilters(info, hostIDsFilter, nameFilter, imageFilter, labelFilter) {
			results = append(results, info)
		}
	}

	// --- Agent containers ---
	alertsMutex.RLock()
	for hostID, containers := range agentMetrics {
		for _, c := range containers {
			info := ContainerInfo{
				ID:         c.ID,
				Name:       c.Name,
				Image:      c.Image,
				CPUPercent: c.CPUPercent,
				MemoryMB:   c.MemoryMB,
				Logs:       c.Logs,
				HostID:     hostID,
			}
			if matchesFilters(info, hostIDsFilter, nameFilter, imageFilter, labelFilter) {
				results = append(results, info)
			}
		}
	}
	alertsMutex.RUnlock()

	// --- Sorting ---
	sort.Slice(results, func(i, j int) bool {
		switch sortBy {
		case "cpu":
			if order == "desc" {
				return results[i].CPUPercent > results[j].CPUPercent
			}
			return results[i].CPUPercent < results[j].CPUPercent
		case "memory":
			if order == "desc" {
				return results[i].MemoryMB > results[j].MemoryMB
			}
			return results[i].MemoryMB < results[j].MemoryMB
		case "name":
			fallthrough
		default:
			if order == "desc" {
				return results[i].Name > results[j].Name
			}
			return results[i].Name < results[j].Name
		}
	})

	// --- Pagination ---
	if offset > len(results) {
		offset = len(results)
	}
	end := offset + limit
	if limit <= 0 || end > len(results) {
		end = len(results)
	}
	results = results[offset:end]

	// --- Response ---
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func matchesFilters(c ContainerInfo, hostIDs []string, name, image, label string) bool {
	// HostID filter
	if hostIDs[0] != "" && !contains(hostIDs, c.HostID) {
		return false
	}
	// Name filter
	if name != "" && !strings.Contains(strings.ToLower(c.Name), name) {
		return false
	}
	// Image filter
	if image != "" && !strings.Contains(strings.ToLower(c.Image), image) {
		return false
	}
	// Label filtering (placeholder, if you add labels)
	if label != "" && !strings.Contains(strings.ToLower(c.Image), label) {
		return false
	}
	return true
}

func contains(list []string, item string) bool {
	for _, v := range list {
		if strings.TrimSpace(v) == item {
			return true
		}
	}
	return false
}

