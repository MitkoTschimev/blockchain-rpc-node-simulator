package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for testing
		},
	}
	subManager = NewSubscriptionManager()

	// chainIdToName maps chainIds to their corresponding chain names
	chainIdToName = map[string]string{
		"1":     "ethereum",  // Ethereum Mainnet
		"10":    "optimism",  // Optimism
		"42161": "arbitrum",  // Arbitrum
		"43114": "avalanche", // Avalanche
		"8453":  "base",      // Base
		"56":    "binance",   // Binance Smart Chain
	}
)

func main() {
	// Start block number incrementer for each chain
	for chainName, chain := range supportedChains {
		go func(name string, c *EVMChain) {
			for {
				time.Sleep(c.BlockInterval)
				if atomic.LoadUint32(&c.BlockIncrement) == 0 {
					newBlock := atomic.AddUint64(&c.BlockNumber, 1)
					log.Printf("Block number increased for %s: %d (interval: %v)", name, newBlock, c.BlockInterval)
					subManager.BroadcastNewBlock(name, newBlock)
				}
			}
		}(chainName, chain)
	}

	// Create a new ServeMux for better route handling
	mux := http.NewServeMux()

	// WebSocket endpoints
	mux.HandleFunc("/ws/evm/", handleEVMWebSocketWithChain) // Note the trailing slash
	mux.HandleFunc("/ws/solana", handleSolanaWebSocket)

	// HTTP endpoints
	mux.HandleFunc("/evm", handleEVMHTTP)
	mux.HandleFunc("/solana", handleSolanaHTTP)

	// Control endpoints
	handleControlEndpoints(mux)

	// Get port from environment variable or use default
	port := os.Getenv("RPC_PORT")
	if port == "" {
		port = "8545"
	}
	port = ":" + port

	log.Printf("Starting RPC simulator on port %s", port)
	log.Printf("EVM endpoints:")
	for chainId, chainName := range chainIdToName {
		log.Printf("  %s: ws://localhost%s/ws/evm/%s", chainName, port, chainId)
	}
	log.Printf("Solana endpoint: ws://localhost%s/ws/solana", port)
	log.Printf("Control endpoints:")
	log.Printf("  POST /control/connections/drop - Drop all connections (optional: block_duration_seconds)")
	log.Printf("  POST /control/block/set - Set block number")
	log.Printf("  POST /control/block/pause - Pause block increment")
	log.Printf("  POST /control/block/resume - Resume block increment")

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

// wsConnWrapper wraps a *websocket.Conn to implement WSConn
type wsConnWrapper struct {
	*websocket.Conn
	writeMu sync.Mutex // Protects writes to the connection
}

func (w *wsConnWrapper) WriteMessage(messageType int, data []byte) error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	return w.Conn.WriteMessage(messageType, data)
}

func (w *wsConnWrapper) Close() error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	return w.Conn.Close()
}

// GetMessages implements WSConn for compatibility with tests
// This is not used in production, only needed to satisfy the interface
func (w *wsConnWrapper) GetMessages() [][]byte {
	return nil
}

// ClearMessages implements WSConn for compatibility with tests
// This is not used in production, only needed to satisfy the interface
func (w *wsConnWrapper) ClearMessages() {
	// No-op in production
}

// handleEVMWebSocketWithChain handles WebSocket connections for specific EVM chains
func handleEVMWebSocketWithChain(w http.ResponseWriter, r *http.Request) {
	// Extract chainId from URL path
	chainId := r.URL.Path[len("/ws/evm/"):]
	chainName, exists := chainIdToName[chainId]
	if !exists {
		http.Error(w, "Invalid chain ID", http.StatusBadRequest)
		return
	}

	log.Printf("EVM client connected to chain %s (chainId: %s)", chainName, chainId)
	if IsBlocked() {
		http.Error(w, "Server is temporarily unavailable", http.StatusServiceUnavailable)
		return
	}

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	conn := &wsConnWrapper{
		Conn: wsConn,
	}
	defer func() {
		count := subManager.CleanupConnection(conn)
		log.Printf("Cleaned up %d subscriptions for disconnected EVM client (chain: %s)", count, chainName)
		conn.Close()
	}()

	for {
		messageType, message, err := wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("EVM client disconnected unexpectedly from chain %s: %v", chainName, err)
			}
			break
		}

		// Parse the request to inject the chain name
		var request JSONRPCRequest
		if err := json.Unmarshal(message, &request); err != nil {
			log.Printf("Failed to parse request for chain %s: %v", chainName, err)
			break
		}

		// Inject chain name as the first parameter for methods that need it
		switch request.Method {
		case "eth_chainId", "eth_blockNumber", "eth_subscribe":
			// Prepend chain name to params
			newParams := make([]interface{}, 0, len(request.Params)+1)
			newParams = append(newParams, chainName)
			newParams = append(newParams, request.Params...)
			request.Params = newParams
		}

		// Re-marshal the modified request
		modifiedMessage, err := json.Marshal(request)
		if err != nil {
			log.Printf("Failed to marshal modified request for chain %s: %v", chainName, err)
			break
		}

		response, err := handleEVMRequest(modifiedMessage, conn)
		if err != nil {
			log.Printf("Handler error for chain %s: %v", chainName, err)
			break
		}

		if err := conn.WriteMessage(messageType, response); err != nil {
			log.Printf("Write error for chain %s: %v", chainName, err)
			break
		}
	}
}

func handleSolanaWebSocket(w http.ResponseWriter, r *http.Request) {
	if IsBlocked() {
		http.Error(w, "Server is temporarily unavailable", http.StatusServiceUnavailable)
		return
	}

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	conn := &wsConnWrapper{
		Conn: wsConn,
	}
	defer func() {
		count := subManager.CleanupConnection(conn)
		log.Printf("Cleaned up %d subscriptions for disconnected Solana client", count)
		conn.Close()
	}()

	for {
		messageType, message, err := wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Solana client disconnected unexpectedly: %v", err)
			}
			break
		}

		response, err := handleSolanaRequest(message, conn)
		if err != nil {
			log.Println("Handler error:", err)
			break
		}

		if err := conn.WriteMessage(messageType, response); err != nil {
			log.Println("Write error:", err)
			break
		}
	}
}

func handleEVMHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var message []byte
	var err error
	message, err = io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Only log non-health check messages
	var request JSONRPCRequest
	if err := json.Unmarshal(message, &request); err == nil && request.Method != "getHealth" {
		log.Printf("Incoming EVM HTTP message: %s", string(message))
	}

	// Create a mock connection for the request
	mockConn := NewMockWSConn()
	response, err := handleEVMRequest(message, mockConn)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func handleSolanaHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var message []byte
	var err error
	message, err = io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Only log non-health check messages
	var request JSONRPCRequest
	if err := json.Unmarshal(message, &request); err == nil && request.Method != "getHealth" {
		log.Printf("Incoming Solana HTTP message: %s", string(message))
	}

	// Create a mock connection for the request
	mockConn := NewMockWSConn()
	response, err := handleSolanaRequest(message, mockConn)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}
