# RPC Simulator

A WebSocket-based RPC simulator that supports both EVM (Ethereum) and Solana blockchain protocols. This server is designed for testing and development purposes, allowing you to simulate blockchain node behavior and control various aspects of the simulation.

## Features

- Dual protocol support (EVM and Solana)
- Real-time block/slot updates
- Subscription-based updates
- Controllable block progression
- Connection management with temporary outage simulation
- Configurable behavior through REST endpoints

## Setup

### Prerequisites

- Go 1.16 or higher
- [gorilla/websocket](https://github.com/gorilla/websocket)

### Installation

1. Clone the repository
2. Install dependencies:
```bash
go mod init rpc-simulator
go get github.com/gorilla/websocket
```

3. Run the server:
```bash
# Regular run
go run .

# With hot reload (using air)
air
```

The server will start on port 8545 by default.

## Endpoints

### WebSocket Endpoints

1. EVM Endpoint: `ws://localhost:8545/ws/evm`
2. Solana Endpoint: `ws://localhost:8545/ws/solana`

### HTTP Endpoints

1. EVM Endpoint: `http://localhost:8545/evm`
2. Solana Endpoint: `http://localhost:8545/solana`

Both HTTP endpoints accept POST requests with JSON-RPC 2.0 formatted bodies.

## Supported Methods

### EVM Methods

1. WebSocket and HTTP:
   - `eth_chainId` - Get the current chain ID
   - `eth_blockNumber` - Get the current block number
   - `eth_getBalance` - Get account balance (mock)
   - `getHealth` - Get node health status

2. WebSocket Only:
   - `eth_subscribe` - Subscribe to updates
   - `eth_unsubscribe` - Unsubscribe from updates

Example HTTP requests:
```bash
# Get health status
curl -X POST http://localhost:8545/evm \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getHealth"}'

# Get chain ID
curl -X POST http://localhost:8545/evm \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"eth_chainId"}'
```

### Solana Methods

1. WebSocket and HTTP:
   - `getSlot` - Get current slot number
   - `getVersion` - Get node version info
   - `getHealth` - Get node health status

2. WebSocket Only:
   - `slotSubscribe` - Subscribe to slot updates
   - `slotUnsubscribe` - Unsubscribe from updates

Example HTTP requests:
```bash
# Get health status
curl -X POST http://localhost:8545/solana \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getHealth"}'

# Get current slot
curl -X POST http://localhost:8545/solana \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getSlot"}'
```

## Response Formats

### Health Check Response
Both chains return the same format for health checks:
```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": "ok"
}
```

### Error Response Format
All endpoints follow the JSON-RPC 2.0 error format:
```json
{
    "jsonrpc": "2.0",
    "id": null,
    "error": {
        "code": -32700,
        "message": "Parse error",
        "data": null
    }
}
```

Common error codes:
- `-32700`: Parse error
- `-32600`: Invalid Request
- `-32601`: Method not found
- `-32602`: Invalid params
- `-32603`: Internal error

### Subscription Notifications

1. EVM Block Notifications:
```json
{
    "jsonrpc": "2.0",
    "method": "eth_subscription",
    "params": {
        "subscription": "123",
        "result": {
            "number": "0x1b4"
        }
    }
}
```

2. Solana Slot Notifications:
```json
{
    "jsonrpc": "2.0",
    "method": "slotNotification",
    "params": {
        "subscription": 123,
        "result": {
            "parent": 435,
            "root": 432,
            "slot": 436
        }
    }
}
```

Note: Subscription IDs are returned as strings for EVM and numbers for Solana.

## Testing Tools

### Using wscat for WebSocket Testing

1. Install wscat:
```bash
npm install -g wscat
```

2. Connect to endpoints:
```bash
# EVM
wscat -c ws://localhost:8545/ws/evm

# Solana
wscat -c ws://localhost:8545/ws/solana
```

### Using curl for HTTP Testing

1. Test EVM endpoints:
```bash
# Health check
curl -X POST http://localhost:8545/evm \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getHealth"}'

# Chain ID
curl -X POST http://localhost:8545/evm \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"eth_chainId"}'

# Block number
curl -X POST http://localhost:8545/evm \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber"}'
```

2. Test Solana endpoints:
```bash
# Health check
curl -X POST http://localhost:8545/solana \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getHealth"}'

# Get slot
curl -X POST http://localhost:8545/solana \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getSlot"}'

# Get version
curl -X POST http://localhost:8545/solana \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getVersion"}'
```

## Control API

The server provides REST endpoints to control its behavior:

### Connection Management

**Drop all active connections:**
```bash
# Just drop all connections
curl -X POST http://localhost:8545/control/connections/drop

# Drop all connections and block new ones for 30 seconds
curl -X POST http://localhost:8545/control/connections/drop \
  -H "Content-Type: application/json" \
  -d '{"block_duration_seconds": 30}'
```

Response:
```json
{
    "success": true,
    "message": "Dropped all connections and blocked new connections for specified duration"
}
```

When blocking duration is specified:
- All existing connections are immediately dropped
- New connection attempts during the blocking period receive HTTP 503 (Service Unavailable)
- After the specified duration, the server automatically starts accepting new connections

### Block Control

**Set specific block number:**
```bash
curl -X POST http://localhost:8545/control/block/set \
  -H "Content-Type: application/json" \
  -d '{"block_number": 1000}'
```

Response:
```json
{
    "success": true,
    "message": "Block number updated"
}
```

**Pause block increment:**
```bash
curl -X POST http://localhost:8545/control/block/pause
```

Response:
```json
{
    "success": true,
    "message": "Block increment paused"
}
```

**Resume block increment:**
```bash
curl -X POST http://localhost:8545/control/block/resume
```

Response:
```json
{
    "success": true,
    "message": "Block increment resumed"
}
```

## Testing Scenarios

### 1. Testing Reconnection Logic

1. Subscribe to updates
2. Drop connections with temporary unavailability:
```bash
curl -X POST http://localhost:8545/control/connections/drop \
  -H "Content-Type: application/json" \
  -d '{"block_duration_seconds": 10}'
```
3. Observe client reconnection behavior:
   - Initial reconnection attempts should fail with HTTP 503
   - After 10 seconds, reconnection should succeed

### 2. Testing Block Reorgs

1. Subscribe to block updates
2. Set block to a lower number:
```bash
curl -X POST http://localhost:8545/control/block/set \
  -H "Content-Type: application/json" \
  -d '{"block_number": 100}'
```
3. Observe how client handles the block reorganization

### 3. Testing Timeout Handling

1. Subscribe to updates
2. Pause block increment:
```bash
curl -X POST http://localhost:8545/control/block/pause
```
3. Test client timeout behavior
4. Resume updates:
```bash
curl -X POST http://localhost:8545/control/block/resume
```

## Quick Test Commands

### Installing wscat
```bash
npm install -g wscat
```

### EVM Testing Commands

1. Connect to EVM endpoint:
```bash
wscat -c ws://localhost:8545/ws/evm
```

2. Get chain ID:
```bash
{"jsonrpc": "2.0", "method": "eth_chainId", "params": [], "id": 1}
```

3. Get current block number:
```bash
{"jsonrpc": "2.0", "method": "eth_blockNumber", "params": [], "id": 1}
```

4. Subscribe to new blocks:
```bash
{"jsonrpc": "2.0", "method": "eth_subscribe", "params": ["newHeads"], "id": 1}
```

5. Unsubscribe (replace SUBSCRIPTION_ID with the ID received from subscribe):
```bash
{"jsonrpc": "2.0", "method": "eth_unsubscribe", "params": ["SUBSCRIPTION_ID"], "id": 1}
```

6. Get balance (mock):
```bash
{"jsonrpc": "2.0", "method": "eth_getBalance", "params": ["0x742d35Cc6634C0532925a3b844Bc454e4438f44e"], "id": 1}
```

### Solana Testing Commands

1. Connect to Solana endpoint:
```bash
wscat -c ws://localhost:8545/ws/solana
```

2. Get current slot:
```bash
{"jsonrpc": "2.0", "method": "getSlot", "params": [], "id": 1}
```

3. Get version info:
```bash
{"jsonrpc": "2.0", "method": "getVersion", "params": [], "id": 1}
```

4. Subscribe to slot updates:
```bash
{"jsonrpc": "2.0", "method": "slotSubscribe", "params": [], "id": 1}
```

5. Unsubscribe (replace SUBSCRIPTION_ID with the ID received from subscribe):
```bash
{"jsonrpc": "2.0", "method": "slotUnsubscribe", "params": ["SUBSCRIPTION_ID"], "id": 1}
```

### Testing Control Endpoints

1. Drop all connections:
```bash
curl -X POST http://localhost:8545/control/connections/drop
```

2. Drop connections and block for 30 seconds:
```bash
curl -X POST http://localhost:8545/control/connections/drop \
  -H "Content-Type: application/json" \
  -d '{"block_duration_seconds": 30}'
```

3. Set specific block number:
```bash
curl -X POST http://localhost:8545/control/block/set \
  -H "Content-Type: application/json" \
  -d '{"block_number": 1000}'
```

4. Pause block increment:
```bash
curl -X POST http://localhost:8545/control/block/pause
```

5. Resume block increment:
```bash
curl -X POST http://localhost:8545/control/block/resume
```

### Common Testing Scenarios

1. Test subscription and block updates:
```bash
# Terminal 1 - Connect and subscribe to EVM updates
wscat -c ws://localhost:8545/ws/evm
> {"jsonrpc": "2.0", "method": "eth_subscribe", "params": ["newHeads"], "id": 1}

# Terminal 2 - Connect and subscribe to Solana updates
wscat -c ws://localhost:8545/ws/solana
> {"jsonrpc": "2.0", "method": "slotSubscribe", "params": [], "id": 1}

# Terminal 3 - Control block progression
curl -X POST http://localhost:8545/control/block/set -H "Content-Type: application/json" -d '{"block_number": 1000}'
```

2. Test connection dropping:
```bash
# Terminal 1 - Subscribe to updates
wscat -c ws://localhost:8545/ws/evm
> {"jsonrpc": "2.0", "method": "eth_subscribe", "params": ["newHeads"], "id": 1}

# Terminal 2 - Drop connections after 5 seconds
sleep 5 && curl -X POST http://localhost:8545/control/connections/drop
```

3. Test block pausing:
```bash
# Terminal 1 - Subscribe to updates
wscat -c ws://localhost:8545/ws/evm
> {"jsonrpc": "2.0", "method": "eth_subscribe", "params": ["newHeads"], "id": 1}

# Terminal 2 - Control block progression
curl -X POST http://localhost:8545/control/block/pause
sleep 10
curl -X POST http://localhost:8545/control/block/resume
```

## Default Behavior

- Block/slot number starts at 1
- Automatically increments every 5 seconds
- Returns mainnet chain ID (0x1) for EVM
- Simulated Solana version: 1.14.10
- All origins are allowed for WebSocket connections
- Failed connections are automatically cleaned up

## Error Handling

The server uses standard JSON-RPC 2.0 error codes:
- `-32700`: Parse error
- `-32600`: Invalid Request
- `-32601`: Method not found
- `-32602`: Invalid params
- `-32603`: Internal error

## Development

### Adding New Methods

1. HTTP and WebSocket Methods:
   - Add the method to the appropriate handler (`evm_handler.go` or `solana_handler.go`)
   - Method will be available via both HTTP and WebSocket endpoints

2. WebSocket-Only Methods:
   - Add subscription-related logic to the handler
   - Update the `SubscriptionManager` if needed
   - Add notification format to the handler

### Testing New Methods

1. Unit Tests:
```bash
# Test specific handler
go test -v -run TestEVMHandler
go test -v -run TestSolanaHandler

# Test all
go test -v ./...
```

2. Manual Testing:
   - Use curl for HTTP endpoints
   - Use wscat for WebSocket endpoints
   - Use the control endpoints to simulate different scenarios

### Logging

The simulator provides comprehensive logging:

1. Connection Events:
   - Client connections/disconnections
   - Subscription creation/removal
   - Connection cleanup

2. Message Logging:
   - All incoming HTTP and WebSocket messages
   - Subscription notifications
   - Error messages

3. Block/Slot Updates:
   - Regular block number increments
   - Broadcast notifications

### Thread Safety

The simulator implements thread-safe operations:

1. WebSocket Connections:
   - Protected by mutex for concurrent writes
   - Safe cleanup on disconnection

2. Subscriptions:
   - Thread-safe subscription management
   - Protected access to subscription list
   - Safe concurrent broadcasts

3. Block Updates:
   - Atomic operations for block number updates
   - Thread-safe broadcasting to subscribers

### Hot Reload Development

The project supports hot reload using `air`. To use it:

1. Install air:
```bash
go install github.com/cosmtrek/air@latest
```

2. Run with hot reload:
```bash
air
```

The server will automatically restart when you make changes to the code.

### VS Code Tasks

The project includes VS Code tasks for common operations:

1. Run RPC Server (with hot reload):
   - Command: `air`
   - Default build task

2. Run RPC Server (without hot reload):
   - Command: `go run .`

To run these tasks:
1. Open VS Code command palette (Cmd/Ctrl + Shift + P)
2. Type "Tasks: Run Task"
3. Select the desired task

### Control API Usage

The simulator provides control endpoints to simulate various scenarios:

1. Drop Connections:
```bash
# Drop all connections
curl -X POST http://localhost:8545/control/connections/drop

# Drop and block for 30 seconds
curl -X POST http://localhost:8545/control/connections/drop \
  -H "Content-Type: application/json" \
  -d '{"block_duration_seconds": 30}'
```

2. Control Block/Slot Progression:
```bash
# Set specific block number
curl -X POST http://localhost:8545/control/block/set \
  -H "Content-Type: application/json" \
  -d '{"block_number": 1000}'

# Pause progression
curl -X POST http://localhost:8545/control/block/pause

# Resume progression
curl -X POST http://localhost:8545/control/block/resume
```

### Common Testing Scenarios

1. Connection Handling:
   - Test client reconnection logic
   - Verify subscription cleanup
   - Test concurrent connections

2. Subscription Management:
   - Verify subscription creation/removal
   - Test concurrent subscriptions
   - Check notification delivery

3. Error Handling:
   - Invalid JSON-RPC requests
   - Malformed parameters
   - Connection errors

4. Performance Testing:
   - Multiple concurrent clients
   - High subscription count
   - Rapid connection/disconnection

### Best Practices

1. Error Handling:
   - Always return proper JSON-RPC error responses
   - Include meaningful error messages
   - Clean up resources on error

2. Logging:
   - Log all incoming messages
   - Log important state changes
   - Include relevant context in logs

3. Thread Safety:
   - Use mutexes for shared resources
   - Use atomic operations for counters
   - Protect WebSocket writes

4. Testing:
   - Write unit tests for new methods
   - Test error conditions
   - Test concurrent operations

## License

MIT License

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a new Pull Request

Please ensure:
- Tests pass
- Code is formatted (go fmt)
- Documentation is updated
- Thread safety is maintained 