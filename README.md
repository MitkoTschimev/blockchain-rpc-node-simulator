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

## WebSocket Endpoints

### EVM Endpoint (`ws://localhost:8545/ws/evm`)

Supported methods:
- `eth_chainId`: Get the current chain ID
- `eth_blockNumber`: Get the current block number
- `eth_getBalance`: Get account balance (mock)
- `eth_subscribe`: Subscribe to updates
- `eth_unsubscribe`: Unsubscribe from updates

Example usage:
```javascript
// Subscribe to new blocks
{
    "jsonrpc": "2.0",
    "method": "eth_subscribe",
    "params": ["newHeads"],
    "id": 1
}

// Unsubscribe
{
    "jsonrpc": "2.0",
    "method": "eth_unsubscribe",
    "params": ["subscription_id"],
    "id": 2
}
```

### Solana Endpoint (`ws://localhost:8545/ws/solana`)

Supported methods:
- `getSlot`: Get current slot number
- `getVersion`: Get node version info
- `slotSubscribe`: Subscribe to slot updates
- `slotUnsubscribe`: Unsubscribe from updates

Example usage:
```javascript
// Subscribe to slot updates
{
    "jsonrpc": "2.0",
    "method": "slotSubscribe",
    "params": [],
    "id": 1
}

// Unsubscribe
{
    "jsonrpc": "2.0",
    "method": "slotUnsubscribe",
    "params": ["subscription_id"],
    "id": 2
}
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

To add new features or modify behavior:
1. EVM methods: Modify `evm_handler.go`
2. Solana methods: Modify `solana_handler.go`
3. Control endpoints: Modify `control_handler.go`
4. Subscription logic: Modify `subscription.go`

## Testing

The project includes a comprehensive test suite covering all major components. Here's how to run the tests:

### Running Tests

1. Run all tests:
```bash
go test ./...
```

2. Run tests with verbose output:
```bash
go test -v ./...
```

3. Run a specific test:
```bash
# Example: Run only EVM handler tests
go test -v -run TestEVMHandler

# Example: Run only Solana handler tests
go test -v -run TestSolanaHandler
```

### Test Coverage

1. Check test coverage percentage:
```bash
go test -cover ./...
```

2. Generate detailed coverage report:
```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out
```

### Test Components

The test suite includes:
- Connection management tests (`connection_controller_test.go`)
- Subscription handling tests (`subscription_test.go`)
- EVM RPC method tests (`rpc_handler_test.go`)
- Solana RPC method tests (`rpc_handler_test.go`)
- Concurrent operation tests
- Error handling tests

### Running Tests During Development

When developing new features, it's recommended to:
1. Write tests first (TDD approach)
2. Run tests frequently with `-v` flag for detailed output
3. Check coverage for new code
4. Run the full test suite before committing changes

## License

MIT License 