package main

import (
	"log"
	"net/http"
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
	currentBlock         uint64 = 1 // Start from block 1
	blockIncrementPaused uint32 = 0 // 0 = running, 1 = paused
	subManager                  = NewSubscriptionManager()
)

func main() {
	// Start block number incrementer
	go func() {
		for {
			time.Sleep(5 * time.Second)
			if atomic.LoadUint32(&blockIncrementPaused) == 0 {
				newBlock := atomic.AddUint64(&currentBlock, 1)
				log.Printf("Block number increased to: %d", newBlock)
				subManager.BroadcastNewBlock(newBlock)
			}
		}
	}()

	// Create a new ServeMux for better route handling
	mux := http.NewServeMux()

	// WebSocket endpoints
	mux.HandleFunc("/ws/evm", handleEVMWebSocket)
	mux.HandleFunc("/ws/solana", handleSolanaWebSocket)

	// Control endpoints
	handleControlEndpoints(mux)

	port := ":8545"
	log.Printf("Starting RPC simulator on port %s", port)
	log.Printf("EVM endpoint: ws://localhost%s/ws/evm", port)
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

func handleEVMWebSocket(w http.ResponseWriter, r *http.Request) {
	if IsBlocked() {
		http.Error(w, "Server is temporarily unavailable", http.StatusServiceUnavailable)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		response, err := handleEVMRequest(message, conn)
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

func handleSolanaWebSocket(w http.ResponseWriter, r *http.Request) {
	if IsBlocked() {
		http.Error(w, "Server is temporarily unavailable", http.StatusServiceUnavailable)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
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
