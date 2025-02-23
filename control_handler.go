package main

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

type ControlResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func handleControlEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/control/connections/drop", handleDropConnections)
	mux.HandleFunc("/control/block/set", handleSetBlock)
	mux.HandleFunc("/control/block/pause", handlePauseBlockIncrement)
	mux.HandleFunc("/control/block/resume", handleResumeBlockIncrement)
}

func jsonResponse(w http.ResponseWriter, status int, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

func handleDropConnections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, ControlResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	var req struct {
		BlockDuration int `json:"block_duration_seconds"` // Duration in seconds to block new connections
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// If no body provided or invalid, just drop connections without blocking
		subManager.DropAllConnections()
		jsonResponse(w, http.StatusOK, ControlResponse{
			Success: true,
			Message: "Dropped all connections",
		})
		return
	}

	subManager.DropAllConnections()
	if req.BlockDuration > 0 {
		BlockConnections(time.Duration(req.BlockDuration) * time.Second)
		jsonResponse(w, http.StatusOK, ControlResponse{
			Success: true,
			Message: "Dropped all connections and blocked new connections for specified duration",
		})
	} else {
		jsonResponse(w, http.StatusOK, ControlResponse{
			Success: true,
			Message: "Dropped all connections",
		})
	}
}

func handleSetBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, ControlResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	var req struct {
		BlockNumber uint64 `json:"block_number"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, ControlResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	atomic.StoreUint64(&currentBlock, req.BlockNumber)
	subManager.BroadcastNewBlock(req.BlockNumber)

	jsonResponse(w, http.StatusOK, ControlResponse{
		Success: true,
		Message: "Block number updated",
	})
}

func handlePauseBlockIncrement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, ControlResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	atomic.StoreUint32(&blockIncrementPaused, 1)
	jsonResponse(w, http.StatusOK, ControlResponse{
		Success: true,
		Message: "Block increment paused",
	})
}

func handleResumeBlockIncrement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, ControlResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	atomic.StoreUint32(&blockIncrementPaused, 0)
	jsonResponse(w, http.StatusOK, ControlResponse{
		Success: true,
		Message: "Block increment resumed",
	})
}
