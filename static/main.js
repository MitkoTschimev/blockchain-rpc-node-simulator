class RPCSimulator {
    constructor() {
        this.ws = null;
        this.subscriptions = new Map();
        this.nextId = 1;
        this.setupEventListeners();
        this.setupConnectionTracking();
        this.setupBlockTracking();
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

        // Add error simulation event listeners
        document.getElementById('addErrorBtn').addEventListener('click', () => this.addError());
        document.getElementById('clearErrorsBtn').addEventListener('click', () => this.clearErrors());
        document.getElementById('errorTemplate').addEventListener('change', (e) => this.selectErrorTemplate(e.target.value));

        // Add custom response event listeners
        document.getElementById('setCustomResponseBtn').addEventListener('click', () => this.setCustomResponse());
        document.getElementById('clearCustomResponseBtn').addEventListener('click', () => this.clearCustomResponse());

        // Load predefined errors and current error configs
        this.loadPredefinedErrors();
        this.loadErrorConfigs();
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

    setupBlockTracking() {
        let retryCount = 0;
        const maxRetries = 5;
        const retryDelay = 2000; // 2 seconds

        const connect = () => {
            const evtSource = new EventSource('/sse/blocks');

            evtSource.onopen = () => {
                retryCount = 0; // Reset retry count on successful connection
                this.log('Block tracking active', 'info');
                document.getElementById('latestBlocks').classList.remove('disconnected');
            };

            evtSource.onmessage = (event) => {
                try {
                    const blocks = JSON.parse(event.data);
                    this.updateLatestBlocks(blocks);
                } catch (error) {
                    this.log('Error parsing block data: ' + error.message, 'error');
                }
            };

            evtSource.onerror = (error) => {
                document.getElementById('latestBlocks').classList.add('disconnected');
                this.log('Block tracking error: ' + (error.message || 'Connection lost'), 'error');
                evtSource.close();

                // Attempt to reconnect with exponential backoff
                if (retryCount < maxRetries) {
                    const delay = retryDelay * Math.pow(2, retryCount);
                    retryCount++;
                    this.log(`Retrying block tracking in ${delay/1000} seconds... (Attempt ${retryCount}/${maxRetries})`, 'warning');
                    setTimeout(connect, delay);
                } else {
                    this.log('Failed to establish block tracking after multiple attempts', 'error');
                }
            };
        };

        // Add styles for disconnected state
        const style = document.createElement('style');
        style.textContent = `
            #latestBlocks.disconnected {
                opacity: 0.6;
                position: relative;
            }
            #latestBlocks.disconnected::after {
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
            .block-updated {
                animation: blockUpdate 0.3s ease-in-out;
            }
            @keyframes blockUpdate {
                0% { background-color: #fff; }
                50% { background-color: #2ecc71; }
                100% { background-color: #fff; }
            }
        `;
        document.head.appendChild(style);

        // Start the connection
        connect();
    }

    updateLatestBlocks(blocks) {
        const tbody = document.getElementById('latestBlocksBody');

        // Store previous block numbers for animation
        const previousBlocks = {};
        tbody.querySelectorAll('tr').forEach(tr => {
            const chainId = tr.dataset.chainId;
            const blockNum = tr.querySelector('.block-number')?.textContent;
            if (chainId && blockNum) {
                previousBlocks[chainId] = blockNum;
            }
        });

        tbody.innerHTML = '';

        // Chain name mapping
        const chainNames = {
            '1': 'Ethereum',
            '10': 'Optimism',
            '56': 'Binance',
            '100': 'Gnosis',
            '130': 'Unichain',
            '137': 'Polygon',
            '146': 'Sonic',
            '250': 'Fantom',
            '324': 'Zksync',
            '8217': 'Kaia',
            '8453': 'Base',
            '42161': 'Arbitrum',
            '43114': 'Avalanche',
            '59144': 'Linea',
            '501': 'Solana'
        };

        // Sort chains by name
        const sortedChains = Object.entries(blocks).sort((a, b) => {
            const nameA = chainNames[a[0]] || a[0];
            const nameB = chainNames[b[0]] || b[0];
            return nameA.localeCompare(nameB);
        });

        // Create table rows
        sortedChains.forEach(([chainId, block]) => {
            const tr = document.createElement('tr');
            tr.dataset.chainId = chainId;

            const blockNumber = block.number.toString();
            const isUpdated = previousBlocks[chainId] && previousBlocks[chainId] !== blockNumber;
            const timestamp = new Date(block.timestamp * 1000).toLocaleTimeString();

            // Truncate hash for display
            const hashDisplay = block.hash ?
                (block.hash.substring(0, 10) + '...' + block.hash.substring(block.hash.length - 8)) :
                'N/A';

            tr.className = isUpdated ? 'block-updated' : '';
            tr.innerHTML = `
                <td style="padding: 8px; border-bottom: 1px solid #ddd;">
                    ${chainNames[chainId] || chainId}
                </td>
                <td style="text-align: right; padding: 8px; border-bottom: 1px solid #ddd;">
                    <span class="block-number">${blockNumber}</span>
                </td>
                <td style="text-align: left; padding: 8px; border-bottom: 1px solid #ddd; font-family: monospace; font-size: 0.85rem;">
                    <span title="${block.hash}">${hashDisplay}</span>
                </td>
                <td style="text-align: right; padding: 8px; border-bottom: 1px solid #ddd; font-size: 0.85rem;">
                    ${timestamp}
                </td>
            `;
            tbody.appendChild(tr);
        });
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

    async loadPredefinedErrors() {
        try {
            const response = await fetch('/control/errors/predefined');
            if (response.ok) {
                const data = await response.json();
                const select = document.getElementById('errorTemplate');

                // Clear existing options except the first one
                select.innerHTML = '<option value="">-- Select Error Template --</option>';

                // Add predefined errors
                Object.entries(data.predefined_errors).forEach(([key, error]) => {
                    const option = document.createElement('option');
                    option.value = key;
                    option.textContent = `${error.message} (${error.code})`;
                    select.appendChild(option);
                });
            }
        } catch (error) {
            this.log(`Failed to load predefined errors: ${error.message}`, 'error');
        }
    }

    async loadErrorConfigs() {
        const chainId = document.getElementById('chainSelect').value;
        const chainName = chainId === '501' ? 'solana' : chainIdToName[chainId];

        try {
            const response = await fetch(`/control/errors/list?chain=${chainName}`);
            if (response.ok) {
                const data = await response.json();
                this.updateErrorList(data.error_configs || []);
            }
        } catch (error) {
            this.log(`Failed to load error configs: ${error.message}`, 'error');
        }
    }

    async selectErrorTemplate(templateKey) {
        if (!templateKey) return;

        try {
            const response = await fetch('/control/errors/predefined');
            if (response.ok) {
                const data = await response.json();
                const error = data.predefined_errors[templateKey];

                if (error) {
                    // Fill in the custom fields
                    document.getElementById('customErrorCode').value = error.code;
                    document.getElementById('customErrorMessage').value = error.message;
                    document.getElementById('customErrorData').value = error.data || '';
                    document.getElementById('customErrorProbability').value = '0.1';
                    document.getElementById('customErrorMethods').value = error.methods ? error.methods.join(', ') : '';
                }
            }
        } catch (error) {
            this.log(`Failed to load error template: ${error.message}`, 'error');
        }
    }

    async addError() {
        const chainId = document.getElementById('chainSelect').value;
        const chainName = chainId === '501' ? 'solana' : chainIdToName[chainId];

        const code = parseInt(document.getElementById('customErrorCode').value);
        const message = document.getElementById('customErrorMessage').value;
        const data = document.getElementById('customErrorData').value;
        const probability = parseFloat(document.getElementById('customErrorProbability').value);
        const methodsStr = document.getElementById('customErrorMethods').value;

        if (!code || !message || isNaN(probability)) {
            this.log('Please fill in error code, message, and probability', 'error');
            return;
        }

        if (probability < 0 || probability > 1) {
            this.log('Probability must be between 0 and 1', 'error');
            return;
        }

        const methods = methodsStr ? methodsStr.split(',').map(m => m.trim()).filter(m => m) : [];

        const errorConfig = {
            code: code,
            message: message,
            data: data,
            probability: probability,
            methods: methods
        };

        const response = await this.sendControlRequest('/control/errors/add', {
            chain: chainName,
            error_config: errorConfig
        });

        if (response.ok) {
            this.log(`Added error configuration: ${message} (${code})`);
            this.loadErrorConfigs();
            // Clear form
            document.getElementById('customErrorCode').value = '';
            document.getElementById('customErrorMessage').value = '';
            document.getElementById('customErrorData').value = '';
            document.getElementById('customErrorProbability').value = '';
            document.getElementById('customErrorMethods').value = '';
            document.getElementById('errorTemplate').value = '';
        } else {
            this.log(`Failed to add error configuration: ${response.statusText}`, 'error');
        }
    }

    async clearErrors() {
        const chainId = document.getElementById('chainSelect').value;
        const chainName = chainId === '501' ? 'solana' : chainIdToName[chainId];

        const response = await this.sendControlRequest('/control/errors/clear', {
            chain: chainName
        });

        if (response.ok) {
            this.log('Cleared all error configurations');
            this.loadErrorConfigs();
        } else {
            this.log(`Failed to clear error configurations: ${response.statusText}`, 'error');
        }
    }

    async removeError(index) {
        const chainId = document.getElementById('chainSelect').value;
        const chainName = chainId === '501' ? 'solana' : chainIdToName[chainId];

        const response = await this.sendControlRequest('/control/errors/remove', {
            chain: chainName,
            index: index
        });

        if (response.ok) {
            this.log(`Removed error configuration at index ${index}`);
            this.loadErrorConfigs();
        } else {
            this.log(`Failed to remove error configuration: ${response.statusText}`, 'error');
        }
    }

    updateErrorList(errorConfigs) {
        const errorList = document.getElementById('errorList');

        if (!errorConfigs || errorConfigs.length === 0) {
            errorList.innerHTML = '<div style="color: #666;">No errors configured</div>';
            return;
        }

        errorList.innerHTML = errorConfigs.map((error, index) => {
            const methodsText = error.methods && error.methods.length > 0
                ? `<br><small>Methods: ${error.methods.join(', ')}</small>`
                : '';
            return `
                <div style="padding: 0.5rem; background: #f5f5f5; margin-bottom: 0.5rem; border-radius: 4px; display: flex; justify-content: space-between; align-items: start;">
                    <div>
                        <strong>${error.message}</strong> (${error.code})
                        <br><small>Probability: ${(error.probability * 100).toFixed(1)}%</small>
                        ${methodsText}
                    </div>
                    <button class="btn btn-danger" style="padding: 0.25rem 0.5rem; font-size: 0.8rem;" onclick="simulator.removeError(${index})">Remove</button>
                </div>
            `;
        }).join('');
    }

    async setCustomResponse() {
        const chainId = document.getElementById('chainSelect').value;
        const chainName = chainId === '501' ? 'solana' : chainIdToName[chainId];

        const customResponseJson = document.getElementById('customResponseJson').value.trim();
        const methodsStr = document.getElementById('customResponseMethods').value.trim();

        if (!customResponseJson) {
            this.log('Please enter a custom JSON response', 'error');
            return;
        }

        // Validate JSON
        try {
            JSON.parse(customResponseJson);
        } catch (e) {
            this.log('Invalid JSON format: ' + e.message, 'error');
            return;
        }

        const methods = methodsStr ? methodsStr.split(',').map(m => m.trim()).filter(m => m) : [];

        const response = await this.sendControlRequest('/control/response/custom', {
            chain: chainName,
            custom_response: customResponseJson,
            enabled: true,
            methods: methods
        });

        if (response.ok) {
            const methodsText = methods.length > 0 ? ` for methods: ${methods.join(', ')}` : ' for all methods';
            this.log('Custom response enabled' + methodsText);
            this.updateCustomResponseStatus(true, methods);
        } else {
            const errorText = await response.text();
            this.log(`Failed to set custom response: ${errorText}`, 'error');
        }
    }

    async clearCustomResponse() {
        const chainId = document.getElementById('chainSelect').value;
        const chainName = chainId === '501' ? 'solana' : chainIdToName[chainId];

        const response = await this.sendControlRequest('/control/response/custom', {
            chain: chainName,
            custom_response: '',
            enabled: false,
            methods: []
        });

        if (response.ok) {
            this.log('Custom response disabled');
            this.updateCustomResponseStatus(false, []);
        } else {
            this.log(`Failed to clear custom response: ${response.statusText}`, 'error');
        }
    }

    updateCustomResponseStatus(enabled, methods) {
        const statusDiv = document.getElementById('customResponseStatus');
        if (enabled) {
            const methodsText = methods.length > 0
                ? `<br><small>Methods: ${methods.join(', ')}</small>`
                : '<br><small>Applies to all methods</small>';
            statusDiv.innerHTML = `<span style="color: var(--success); font-weight: bold;">Status: Enabled</span>${methodsText}`;
        } else {
            statusDiv.innerHTML = '<span style="color: #666;">Status: Disabled</span>';
        }
    }
}

const chainIdToName = {
    '1': 'ethereum',
    '10': 'optimism',
    '56': 'binance',
    '100': 'gnosis',
    '130': 'unichain',
    '137': 'polygon',
    '146': 'sonic',
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