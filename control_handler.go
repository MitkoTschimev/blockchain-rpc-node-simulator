package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

type ControlResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type BlockRequest struct {
	Chain       string `json:"chain"`
	BlockNumber uint64 `json:"block_number"`
}

type TimeoutRequest struct {
	Chain           string  `json:"chain"`
	DurationSeconds float64 `json:"duration_seconds"`
}

type InterruptRequest struct {
	Chain           string  `json:"chain"`
	DurationSeconds float64 `json:"duration_seconds"`
}

type ReorgRequest struct {
	Chain  string `json:"chain"`
	Blocks int    `json:"blocks"`
}

func handleControlEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/control/connections/drop", handleDropConnections)
	mux.HandleFunc("/control/block/set", handleSetBlock)
	mux.HandleFunc("/control/block/pause", handlePauseBlock)
	mux.HandleFunc("/control/block/resume", handleResumeBlock)
	mux.HandleFunc("/control/block/pause_updates", handlePauseUpdates)
	mux.HandleFunc("/control/block/resume_updates", handleResumeUpdates)
	mux.HandleFunc("/control/block/interval", handleSetBlockInterval)
	mux.HandleFunc("/control/block/interrupt", handleInterruptBlocks)
	mux.HandleFunc("/control/timeout/set", handleSetTimeout)
	mux.HandleFunc("/control/timeout/clear", handleClearTimeout)
	mux.HandleFunc("/control/chain/reorg", handleChainReorg)
	mux.HandleFunc("/control/latency", handleSetLatency)
	mux.HandleFunc("/control/chain/error-probability", handleSetErrorProbability)
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
		Chain       string `json:"chain"`
		BlockNumber uint64 `json:"block_number"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, ControlResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	if req.Chain == "solana" {
		atomic.StoreUint64(&solanaNode.SlotNumber, req.BlockNumber)
		subManager.BroadcastNewBlock("solana", req.BlockNumber)
		jsonResponse(w, http.StatusOK, ControlResponse{
			Success: true,
			Message: "Slot number updated for Solana",
		})
		return
	}

	chain, ok := supportedChains[req.Chain]
	if !ok {
		jsonResponse(w, http.StatusBadRequest, ControlResponse{
			Success: false,
			Message: fmt.Sprintf("Unsupported chain: %s", req.Chain),
		})
		return
	}

	atomic.StoreUint64(&chain.BlockNumber, req.BlockNumber)
	subManager.BroadcastNewBlock(req.Chain, req.BlockNumber)

	jsonResponse(w, http.StatusOK, ControlResponse{
		Success: true,
		Message: fmt.Sprintf("Block number updated for chain %s", req.Chain),
	})
}

func handlePauseBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, ControlResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	var req struct {
		Chain string `json:"chain"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		jsonResponse(w, http.StatusBadRequest, ControlResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	if req.Chain == "solana" {
		atomic.StoreUint32(&solanaNode.SlotIncrement, 1)
		jsonResponse(w, http.StatusOK, ControlResponse{
			Success: true,
			Message: "Slot increment paused for Solana",
		})
		return
	}

	if req.Chain == "" {
		// Pause all chains including Solana
		for _, chain := range supportedChains {
			atomic.StoreUint32(&chain.BlockIncrement, 1)
		}
		atomic.StoreUint32(&solanaNode.SlotIncrement, 1)
		jsonResponse(w, http.StatusOK, ControlResponse{
			Success: true,
			Message: "Block/slot increment paused for all chains",
		})
		return
	}

	chain, ok := supportedChains[req.Chain]
	if !ok {
		jsonResponse(w, http.StatusBadRequest, ControlResponse{
			Success: false,
			Message: fmt.Sprintf("Unsupported chain: %s", req.Chain),
		})
		return
	}

	atomic.StoreUint32(&chain.BlockIncrement, 1)
	jsonResponse(w, http.StatusOK, ControlResponse{
		Success: true,
		Message: fmt.Sprintf("Block increment paused for chain %s", req.Chain),
	})
}

func handleResumeBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, ControlResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	var req struct {
		Chain string `json:"chain"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		jsonResponse(w, http.StatusBadRequest, ControlResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	if req.Chain == "solana" {
		atomic.StoreUint32(&solanaNode.SlotIncrement, 0)
		jsonResponse(w, http.StatusOK, ControlResponse{
			Success: true,
			Message: "Slot increment resumed for Solana",
		})
		return
	}

	if req.Chain == "" {
		// Resume all chains including Solana
		for _, chain := range supportedChains {
			atomic.StoreUint32(&chain.BlockIncrement, 0)
		}
		atomic.StoreUint32(&solanaNode.SlotIncrement, 0)
		jsonResponse(w, http.StatusOK, ControlResponse{
			Success: true,
			Message: "Block/slot increment resumed for all chains",
		})
		return
	}

	chain, ok := supportedChains[req.Chain]
	if !ok {
		jsonResponse(w, http.StatusBadRequest, ControlResponse{
			Success: false,
			Message: fmt.Sprintf("Unsupported chain: %s", req.Chain),
		})
		return
	}

	atomic.StoreUint32(&chain.BlockIncrement, 0)
	jsonResponse(w, http.StatusOK, ControlResponse{
		Success: true,
		Message: fmt.Sprintf("Block increment resumed for chain %s", req.Chain),
	})
}

func handlePauseUpdates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse optional duration and chain
	var request struct {
		Chain           string `json:"chain"`
		DurationSeconds int    `json:"duration_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil && err != io.EOF {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.Chain == "" {
		// Pause all chains
		for _, chain := range supportedChains {
			atomic.StoreUint32(&chain.BlockIncrement, 1)
		}
		log.Printf("Block updates paused for all chains")

		// If duration is specified, schedule resume for all chains
		if request.DurationSeconds > 0 {
			go func() {
				time.Sleep(time.Duration(request.DurationSeconds) * time.Second)
				for _, chain := range supportedChains {
					atomic.StoreUint32(&chain.BlockIncrement, 0)
				}
				log.Printf("Block updates resumed for all chains after %d seconds", request.DurationSeconds)
			}()
		}
	} else {
		chain, ok := supportedChains[request.Chain]
		if !ok {
			http.Error(w, fmt.Sprintf("Unsupported chain: %s", request.Chain), http.StatusBadRequest)
			return
		}

		atomic.StoreUint32(&chain.BlockIncrement, 1)
		log.Printf("Block updates paused for chain %s", request.Chain)

		// If duration is specified, schedule resume
		if request.DurationSeconds > 0 {
			go func() {
				time.Sleep(time.Duration(request.DurationSeconds) * time.Second)
				atomic.StoreUint32(&chain.BlockIncrement, 0)
				log.Printf("Block updates resumed for chain %s after %d seconds", request.Chain, request.DurationSeconds)
			}()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Block updates paused",
	})
}

func handleResumeUpdates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Chain string `json:"chain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil && err != io.EOF {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.Chain == "" {
		// Resume all chains
		for _, chain := range supportedChains {
			atomic.StoreUint32(&chain.BlockIncrement, 0)
		}
		log.Printf("Block updates resumed for all chains")
	} else {
		chain, ok := supportedChains[request.Chain]
		if !ok {
			http.Error(w, fmt.Sprintf("Unsupported chain: %s", request.Chain), http.StatusBadRequest)
			return
		}

		atomic.StoreUint32(&chain.BlockIncrement, 0)
		log.Printf("Block updates resumed for chain %s", request.Chain)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Block updates resumed",
	})
}

func handleSetBlockInterval(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, ControlResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	var req struct {
		Chain    string  `json:"chain"`
		Interval float64 `json:"interval_seconds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, ControlResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	if req.Interval <= 0 {
		jsonResponse(w, http.StatusBadRequest, ControlResponse{
			Success: false,
			Message: "Interval must be greater than 0",
		})
		return
	}

	interval := time.Duration(req.Interval * float64(time.Second))

	if req.Chain == "solana" {
		solanaNode.SlotInterval = interval
		log.Printf("Slot interval updated for Solana: %v", interval)
		jsonResponse(w, http.StatusOK, ControlResponse{
			Success: true,
			Message: fmt.Sprintf("Slot interval updated to %v for Solana", interval),
		})
		return
	}

	if req.Chain == "" {
		// Update all chains including Solana
		for name, chain := range supportedChains {
			chain.BlockInterval = interval
			log.Printf("Block interval updated for %s: %v", name, interval)
		}
		solanaNode.SlotInterval = interval
		log.Printf("Slot interval updated for Solana: %v", interval)
		jsonResponse(w, http.StatusOK, ControlResponse{
			Success: true,
			Message: fmt.Sprintf("Block/slot interval updated to %v for all chains", interval),
		})
		return
	}

	chain, ok := supportedChains[req.Chain]
	if !ok {
		jsonResponse(w, http.StatusBadRequest, ControlResponse{
			Success: false,
			Message: fmt.Sprintf("Unsupported chain: %s", req.Chain),
		})
		return
	}

	chain.BlockInterval = interval
	log.Printf("Block interval updated for %s: %v", req.Chain, interval)

	jsonResponse(w, http.StatusOK, ControlResponse{
		Success: true,
		Message: fmt.Sprintf("Block interval updated to %v for chain %s", interval, req.Chain),
	})
}

func handleSetTimeout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TimeoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	chain := getChain(req.Chain)
	if chain == nil {
		http.Error(w, "Invalid chain", http.StatusBadRequest)
		return
	}

	chain.SetTimeout(time.Duration(req.DurationSeconds * float64(time.Second)))
	log.Printf("Set response timeout for %s: %v", req.Chain, req.DurationSeconds)

	w.WriteHeader(http.StatusOK)
}

func handleClearTimeout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Chain string `json:"chain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	chain := getChain(req.Chain)
	if chain == nil {
		http.Error(w, "Invalid chain", http.StatusBadRequest)
		return
	}

	chain.ClearTimeout()
	log.Printf("Cleared response timeout for %s", req.Chain)

	w.WriteHeader(http.StatusOK)
}

func handleInterruptBlocks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, ControlResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	var req InterruptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, ControlResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	if req.DurationSeconds <= 0 {
		jsonResponse(w, http.StatusBadRequest, ControlResponse{
			Success: false,
			Message: "Duration must be greater than 0",
		})
		return
	}

	chain := getChain(req.Chain)
	if chain == nil {
		jsonResponse(w, http.StatusBadRequest, ControlResponse{
			Success: false,
			Message: "Invalid chain",
		})
		return
	}

	// Interrupt block emissions for the specified duration
	chain.InterruptBlocks()

	// Schedule the resume after the duration
	go func() {
		time.Sleep(time.Duration(req.DurationSeconds * float64(time.Second)))
		chain.ResumeBlocks()
	}()

	jsonResponse(w, http.StatusOK, ControlResponse{
		Success: true,
		Message: fmt.Sprintf("Block emissions interrupted for %s for %.1f seconds", req.Chain, req.DurationSeconds),
	})
}

func handleChainReorg(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ReorgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	chain := getChain(req.Chain)
	if chain == nil {
		http.Error(w, "Invalid chain", http.StatusBadRequest)
		return
	}

	chain.TriggerReorg(req.Blocks)
	log.Printf("Triggered chain reorganization for %s: %d blocks", req.Chain, req.Blocks)

	w.WriteHeader(http.StatusOK)
}

// Helper function to get chain instance
func getChain(name string) Chain {
	if name == "solana" {
		return solanaNode
	}
	if chain, ok := supportedChains[name]; ok {
		return chain
	}
	return nil
}

func handleSetLatency(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Chain   string `json:"chain"`
		Latency int64  `json:"latency_ms"` // Latency in milliseconds
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert chain name to chain ID if a name was provided
	chainId := request.Chain
	for id, name := range chainIdToName {
		if name == request.Chain {
			chainId = id
			break
		}
	}

	// Set latency for the specified chain
	latencyDuration := time.Duration(request.Latency) * time.Millisecond
	if chainId == "501" {
		solanaNode.Latency = latencyDuration
		log.Printf("Set Solana latency to %dms", request.Latency)
	} else if chain, exists := supportedChains[chainIdToName[chainId]]; exists {
		chain.Latency = latencyDuration
		log.Printf("Set %s latency to %dms", chainIdToName[chainId], request.Latency)
	} else {
		http.Error(w, fmt.Sprintf("Unknown chain: %s", request.Chain), http.StatusBadRequest)
		return
	}

	// Save the updated configuration to chains.yaml
	config := ChainConfig{
		EVMChains: supportedChains,
		Solana:    solanaNode,
	}
	if err := SaveChainConfig("chains.yaml", &config); err != nil {
		log.Printf("Warning: Failed to save chain configuration: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"chain":   request.Chain,
		"latency": fmt.Sprintf("%dms", request.Latency),
	})
}

func handleSetErrorProbability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Chain            string  `json:"chain"`
		ErrorProbability float64 `json:"error_probability"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert chain name to chain ID
	var chainId string
	for id, name := range chainIdToName {
		if name == request.Chain {
			chainId = id
			break
		}
	}

	if chainId == "" {
		http.Error(w, "Invalid chain name", http.StatusBadRequest)
		return
	}

	// Validate error probability
	if request.ErrorProbability < 0 || request.ErrorProbability > 1 {
		http.Error(w, "Error probability must be between 0 and 1", http.StatusBadRequest)
		return
	}

	// Set error probability for the chain
	if chain, ok := supportedChains[request.Chain]; ok {
		chain.ErrorProbability = request.ErrorProbability
		log.Printf("Set error probability to %.2f for chain %s", request.ErrorProbability, request.Chain)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	} else {
		http.Error(w, "Chain not found", http.StatusNotFound)
	}
}
