class RPCSimulator {
    constructor() {
        this.ws = null;
        this.subscriptions = new Map();
        this.nextId = 1;
        this.setupEventListeners();
        this.setupConnectionTracking();
    }

    setupEventListeners() {
        document.getElementById('connectBtn').addEventListener('click', () => this.connect());
        document.getElementById('disconnectBtn').addEventListener('click', () => this.disconnect());
        document.getElementById('setBlockBtn').addEventListener('click', () => this.setBlockNumber());
        document.getElementById('pauseBlockBtn').addEventListener('click', () => this.pauseBlock());
        document.getElementById('resumeBlockBtn').addEventListener('click', () => this.resumeBlock());
        document.getElementById('setIntervalBtn').addEventListener('click', () => this.setBlockInterval());
        document.getElementById('dropConnectionsBtn').addEventListener('click', () => this.dropConnections());
        
        // New event listeners for chain disruptions
        document.getElementById('setTimeoutBtn').addEventListener('click', () => this.setTimeout());
        document.getElementById('clearTimeoutBtn').addEventListener('click', () => this.clearTimeout());
        document.getElementById('interruptBlocksBtn').addEventListener('click', () => this.interruptBlocks());
        document.getElementById('triggerReorgBtn').addEventListener('click', () => this.triggerReorg());
        
        // Add latency control event listener
        document.getElementById('setLatencyBtn').addEventListener('click', () => this.setLatency());
        
        // Add error probability control event listener
        document.getElementById('setErrorProbabilityBtn').addEventListener('click', () => this.setErrorProbability());

        // Add logs per block control event listener
        document.getElementById('setLogsPerBlockBtn').addEventListener('click', () => this.setLogsPerBlock());
    }

    log(message, type = 'info') {
        const logContainer = document.getElementById('eventLog');
        const entry = document.createElement('div');
        entry.className = `log-entry log-${type}`;
        entry.textContent = `${new Date().toLocaleTimeString()} - ${message}`;
        logContainer.appendChild(entry);
        logContainer.scrollTop = logContainer.scrollHeight;
    }

    updateConnectionStatus(connected) {
        const status = document.getElementById('connectionStatus');
        const connectBtn = document.getElementById('connectBtn');
        const disconnectBtn = document.getElementById('disconnectBtn');

        status.innerHTML = connected ? 
            '<span class="badge badge-success">Connected</span>' :
            '<span class="badge badge-danger">Disconnected</span>';
        
        connectBtn.style.display = connected ? 'none' : 'inline-block';
        disconnectBtn.style.display = connected ? 'inline-block' : 'none';
    }

    updateSubscriptions() {
        const container = document.getElementById('subscriptions');
        container.innerHTML = '';
        
        this.subscriptions.forEach((value, id) => {
            const li = document.createElement('li');
            li.innerHTML = `
                <span>${value.type} - ID: ${id}</span>
                <button class="btn btn-danger" onclick="simulator.unsubscribe('${id}')">Unsubscribe</button>
            `;
            container.appendChild(li);
        });
    }

    async connect() {
        const chainId = document.getElementById('chainSelect').value;
        const wsEndpoint = `ws://${window.location.host}/ws/chain/${chainId}`;

        try {
            this.ws = new WebSocket(wsEndpoint);
            
            this.ws.onopen = () => {
                this.log('Connected to RPC server');
                this.updateConnectionStatus(true);
                this.subscribe();
            };

            this.ws.onclose = () => {
                this.log('Disconnected from RPC server', 'error');
                this.updateConnectionStatus(false);
                this.subscriptions.clear();
                this.updateSubscriptions();
            };

            this.ws.onmessage = (event) => {
                const data = JSON.parse(event.data);
                if (data.method === 'eth_subscription' || data.method === 'slotNotification') {
                    this.log(`New ${data.method} notification: ${JSON.stringify(data.params.result)}`);
                } else {
                    this.log(`Received: ${event.data}`);
                }
            };

            this.ws.onerror = (error) => {
                this.log(`WebSocket error: ${error.message}`, 'error');
            };
        } catch (error) {
            this.log(`Connection error: ${error.message}`, 'error');
        }
    }

    disconnect() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }

    async subscribe() {
        if (!this.ws) return;

        const chainId = document.getElementById('chainSelect').value;
        const request = {
            jsonrpc: '2.0',
            id: this.nextId++,
            method: chainId === '501' ? 'slotSubscribe' : 'eth_subscribe',
            params: chainId === '501' ? [] : ['newHeads']
        };

        this.ws.send(JSON.stringify(request));
    }

    async unsubscribe(subscriptionId) {
        if (!this.ws) return;

        const chainId = document.getElementById('chainSelect').value;
        const request = {
            jsonrpc: '2.0',
            id: this.nextId++,
            method: chainId === '501' ? 'slotUnsubscribe' : 'eth_unsubscribe',
            params: [subscriptionId]
        };

        this.ws.send(JSON.stringify(request));
        this.subscriptions.delete(subscriptionId);
        this.updateSubscriptions();
    }

    async setBlockNumber() {
        const blockNumber = document.getElementById('blockNumber').value;
        const chainId = document.getElementById('chainSelect').value;
        
        const response = await this.sendControlRequest('/control/block/set', {
            chain: chainId === 'solana' ? 'solana' : chainIdToName[chainId],
            block_number: parseInt(blockNumber)
        });

        if (response.ok) {
            this.log(`Block number set to ${blockNumber}`);
        } else {
            this.log(`Failed to set block number: ${response.statusText}`, 'error');
        }
    }

    async pauseBlock() {
        const chainId = document.getElementById('chainSelect').value;
        const response = await this.sendControlRequest('/control/block/pause', {
            chain: chainId === 'solana' ? 'solana' : chainIdToName[chainId]
        });

        if (response.ok) {
            this.log('Block increment paused');
        } else {
            this.log(`Failed to pause block increment: ${response.statusText}`, 'error');
        }
    }

    async resumeBlock() {
        const chainId = document.getElementById('chainSelect').value;
        const response = await this.sendControlRequest('/control/block/resume', {
            chain: chainId === 'solana' ? 'solana' : chainIdToName[chainId]
        });

        if (response.ok) {
            this.log('Block increment resumed');
        } else {
            this.log(`Failed to resume block increment: ${response.statusText}`, 'error');
        }
    }

    async setBlockInterval() {
        const interval = document.getElementById('intervalSeconds').value;
        const chainId = document.getElementById('chainSelect').value;
        
        const response = await this.sendControlRequest('/control/block/interval', {
            chain: chainId === 'solana' ? 'solana' : chainIdToName[chainId],
            interval_seconds: parseFloat(interval)
        });

        if (response.ok) {
            this.log(`Block interval set to ${interval} seconds`);
        } else {
            this.log(`Failed to set block interval: ${response.statusText}`, 'error');
        }
    }

    async setTimeout() {
        const duration = document.getElementById('timeoutDuration').value;
        const chainId = document.getElementById('chainSelect').value;
        
        const response = await this.sendControlRequest('/control/timeout/set', {
            chain: chainId === 'solana' ? 'solana' : chainIdToName[chainId],
            duration_seconds: parseFloat(duration)
        });

        if (response.ok) {
            this.log(`Response timeout set to ${duration} seconds`);
        } else {
            this.log(`Failed to set timeout: ${response.statusText}`, 'error');
        }
    }

    async clearTimeout() {
        const chainId = document.getElementById('chainSelect').value;
        
        const response = await this.sendControlRequest('/control/timeout/clear', {
            chain: chainId === 'solana' ? 'solana' : chainIdToName[chainId]
        });

        if (response.ok) {
            this.log('Response timeout cleared');
        } else {
            this.log(`Failed to clear timeout: ${response.statusText}`, 'error');
        }
    }

    async interruptBlocks() {
        const duration = document.getElementById('interruptDuration').value;
        const chainId = document.getElementById('chainSelect').value;
        
        const response = await this.sendControlRequest('/control/block/interrupt', {
            chain: chainId === 'solana' ? 'solana' : chainIdToName[chainId],
            duration_seconds: parseFloat(duration)
        });

        if (response.ok) {
            this.log(`Block emissions interrupted for ${duration} seconds`);
        } else {
            this.log(`Failed to interrupt blocks: ${response.statusText}`, 'error');
        }
    }

    async triggerReorg() {
        const blocks = document.getElementById('reorgBlocks').value;
        const chainId = document.getElementById('chainSelect').value;
        
        const response = await this.sendControlRequest('/control/chain/reorg', {
            chain: chainId === 'solana' ? 'solana' : chainIdToName[chainId],
            blocks: parseInt(blocks)
        });

        if (response.ok) {
            this.log(`Chain reorganization triggered for ${blocks} blocks`);
        } else {
            this.log(`Failed to trigger reorg: ${response.statusText}`, 'error');
        }
    }

    async dropConnections() {
        const duration = document.getElementById('blockDuration').value;
        const data = duration ? { block_duration_seconds: parseInt(duration) } : {};
        
        const response = await this.sendControlRequest('/control/connections/drop', data);

        if (response.ok) {
            this.log('Dropped all connections' + (duration ? ` and blocked new connections for ${duration} seconds` : ''));
            // If we have an active connection, disconnect it
            if (this.ws) {
                this.disconnect();
            }
        } else {
            this.log(`Failed to drop connections: ${response.statusText}`, 'error');
        }
    }

    async setLatency() {
        const latency = document.getElementById('latencyMs').value;
        const chainId = document.getElementById('chainSelect').value;
        
        const response = await this.sendControlRequest('/control/chain/latency', {
            chain: chainId === '501' ? 'solana' : chainIdToName[chainId],
            latency_ms: parseInt(latency)
        });

        if (response.ok) {
            this.log(`Latency set to ${latency}ms for ${chainIdToName[chainId]}`);
        } else {
            this.log(`Failed to set latency: ${response.statusText}`, 'error');
        }
    }

    async setErrorProbability() {
        const probability = document.getElementById('errorProbability').value;
        const chainId = document.getElementById('chainSelect').value;

        const response = await this.sendControlRequest('/control/chain/error-probability', {
            chain: chainId === '501' ? 'solana' : chainIdToName[chainId],
            error_probability: parseFloat(probability)
        });

        if (response.ok) {
            this.log(`Error probability set to ${probability} for ${chainIdToName[chainId]}`);
        } else {
            this.log(`Failed to set error probability: ${response.statusText}`, 'error');
        }
    }

    async setLogsPerBlock() {
        const logsPerBlock = document.getElementById('logsPerBlock').value;
        const chainId = document.getElementById('chainSelect').value;

        const response = await this.sendControlRequest('/control/chain/logs-per-block', {
            chain: chainId === '501' ? 'solana' : chainIdToName[chainId],
            logs_per_block: parseInt(logsPerBlock)
        });

        if (response.ok) {
            this.log(`Logs per block set to ${logsPerBlock} for ${chainIdToName[chainId]}`);
        } else {
            this.log(`Failed to set logs per block: ${response.statusText}`, 'error');
        }
    }

    async sendControlRequest(endpoint, data) {
        try {
            return await fetch(endpoint, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(data)
            });
        } catch (error) {
            this.log(`Control request failed: ${error.message}`, 'error');
            return { ok: false, statusText: error.message };
        }
    }

    setupConnectionTracking() {
        let retryCount = 0;
        const maxRetries = 5;
        const retryDelay = 2000; // 2 seconds

        const connect = () => {
            const evtSource = new EventSource('/sse/connections');
            
            evtSource.onopen = () => {
                retryCount = 0; // Reset retry count on successful connection
                this.log('Connection tracking active', 'info');
                document.getElementById('connectionCounts').classList.remove('disconnected');
            };
            
            evtSource.onmessage = (event) => {
                try {
                    const connections = JSON.parse(event.data);
                    this.updateConnectionCounts(connections);
                    
                    // Update the connection status in the UI if we have an active WebSocket
                    if (this.ws) {
                        const chainId = document.getElementById('chainSelect').value;
                        const count = connections[chainId] || 0;
                        if (count === 0) {
                            // Our connection might have been dropped externally
                            this.updateConnectionStatus(false);
                            this.ws = null;
                            this.subscriptions.clear();
                            this.updateSubscriptions();
                        }
                    }
                } catch (error) {
                    this.log('Error parsing connection data: ' + error.message, 'error');
                }
            };

            evtSource.onerror = (error) => {
                document.getElementById('connectionCounts').classList.add('disconnected');
                this.log('Connection tracking error: ' + (error.message || 'Connection lost'), 'error');
                evtSource.close();
                
                // Attempt to reconnect with exponential backoff
                if (retryCount < maxRetries) {
                    const delay = retryDelay * Math.pow(2, retryCount);
                    retryCount++;
                    this.log(`Retrying connection in ${delay/1000} seconds... (Attempt ${retryCount}/${maxRetries})`, 'warning');
                    setTimeout(connect, delay);
                } else {
                    this.log('Failed to establish connection tracking after multiple attempts', 'error');
                }
            };
        };

        // Add styles for disconnected state
        const style = document.createElement('style');
        style.textContent = `
            #connectionCounts.disconnected {
                opacity: 0.6;
                position: relative;
            }
            #connectionCounts.disconnected::after {
                content: 'Reconnecting...';
                position: absolute;
                top: 50%;
                left: 50%;
                transform: translate(-50%, -50%);
                background: var(--warning);
                color: var(--dark);
                padding: 4px 8px;
                border-radius: 4px;
                font-size: 0.8rem;
                font-weight: 500;
            }
        `;
        document.head.appendChild(style);

        // Start the connection
        connect();
    }

    updateConnectionCounts(connections) {
        const tbody = document.getElementById('connectionCountsBody');
        tbody.innerHTML = '';

        // Chain name mapping
        const chainNames = {
            '1': 'Ethereum',
            '10': 'Optimism',
            '56': 'Binance',
            '100': 'Gnosis',
            '137': 'Polygon',
            '250': 'Fantom',
            '324': 'Zksync',
            '8217': 'Kaia',
            '8453': 'Base',
            '42161': 'Arbitrum',
            '43114': 'Avalanche',
            '59144': 'Linea'
        };

        // Sort chains by name
        const sortedChains = Object.entries(connections).sort((a, b) => {
            const nameA = chainNames[a[0]] || a[0];
            const nameB = chainNames[b[0]] || b[0];
            return nameA.localeCompare(nameB);
        });

        // Create table rows with animation
        sortedChains.forEach(([chainId, count]) => {
            const tr = document.createElement('tr');
            const prevCount = tr.querySelector('.badge')?.textContent || '0';
            
            tr.innerHTML = `
                <td style="padding: 8px; border-bottom: 1px solid #ddd;">
                    ${chainNames[chainId] || chainId}
                </td>
                <td style="text-align: right; padding: 8px; border-bottom: 1px solid #ddd;">
                    <span class="badge ${count > 0 ? 'badge-success' : 'badge-danger'} 
                          ${count !== parseInt(prevCount) ? 'badge-updated' : ''}">${count}</span>
                </td>
            `;
            tbody.appendChild(tr);
        });

        // Add animation for updated values
        const style = document.createElement('style');
        style.textContent = `
            @keyframes badgeUpdate {
                0% { transform: scale(1); }
                50% { transform: scale(1.2); }
                100% { transform: scale(1); }
            }
            .badge-updated {
                animation: badgeUpdate 0.3s ease-in-out;
            }
        `;
        document.head.appendChild(style);
    }
}

const chainIdToName = {
    '1': 'ethereum',
    '10': 'optimism',
    '56': 'binance',
    '100': 'gnosis',
    '137': 'polygon',
    '250': 'fantom',
    '324': 'zksync',
    '8217': 'kaia',
    '8453': 'base',
    '42161': 'arbitrum',
    '43114': 'avalanche',
    '59144': 'linea'
};

// Initialize the simulator
const simulator = new RPCSimulator(); 