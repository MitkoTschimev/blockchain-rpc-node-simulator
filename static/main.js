class RPCSimulator {
    constructor() {
        this.ws = null;
        this.subscriptions = new Map();
        this.nextId = 1;
        this.setupEventListeners();
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
        const wsEndpoint = chainId === 'solana' ?
            `ws://${window.location.host}/ws/solana` :
            `ws://${window.location.host}/ws/evm/${chainId}`;

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
            method: chainId === 'solana' ? 'slotSubscribe' : 'eth_subscribe',
            params: chainId === 'solana' ? [] : ['newHeads']
        };

        this.ws.send(JSON.stringify(request));
    }

    async unsubscribe(subscriptionId) {
        if (!this.ws) return;

        const chainId = document.getElementById('chainSelect').value;
        const request = {
            jsonrpc: '2.0',
            id: this.nextId++,
            method: chainId === 'solana' ? 'slotUnsubscribe' : 'eth_unsubscribe',
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
}

const chainIdToName = {
    '1': 'ethereum',
    '10': 'optimism',
    '42161': 'arbitrum',
    '43114': 'avalanche',
    '8453': 'base',
    '56': 'binance'
};

// Initialize the simulator
const simulator = new RPCSimulator(); 