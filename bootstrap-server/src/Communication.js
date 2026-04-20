import fs from 'fs';
import WebSocket, { WebSocketServer } from 'ws';
import State from './State.js';
import pino from 'pino';

const OK = 'ok';

export default class Communication {
    /**
     *
     * @param {pino.Logger} logger logger instance
     * @param {WebSocketServer} wss WebSocket server
     * @param {State} state state instance
     * @param {number} targetNumNodes target number of nodes
     * @param {string} volumePath volume storage
     * @param {void} exitCallback callback to turn off server and whole bootstrap service once ready
     */
    constructor(logger, wss, state, targetNumNodes, volumePath, exitCallback) {
        this.logger = logger;
        this.wss = wss;
        this.state = state;
        this.targetNumNodes = targetNumNodes;
        this.volumePath = volumePath;
        this.exitCallback = exitCallback;

        // Map WS uuid to node P2P id
        this.nodesIdsMapping = {};

        // During RECEIVE_HELLO stores individual node ids as "id: false"
        // Purpose of value is fulfiled during SEND_CONF when nodes confirm that they received configuration
        this.nodesConfigs = {};

        // Store uniquely "id: true" for each node that confirmed that its ready by sending SEND_RDY message
        this.nodesReady = {};
        this.storedConfigs = 0;

        this.checkConfigInterval;
    }

    /**
     *
     * @param {WebSocket} ws websocket handler
     * @param {WebSocket.RawData} data received data
     */
    processMessage(ws, data) {
        const msg = JSON.parse(data);

        switch (this.state.getState()) {
            case 'BOOTING':
                this.logger.warn(`Received message ${msg.type} during booting`);
                break;
            case 'RECEIVE_HELLO':
                if (msg.type === 'hello') {
                    // Process node introductory message
                    if (!Object.hasOwn(this.nodesConfigs, msg.id)) {
                        this.nodesIdsMapping[ws.id] = msg.id;
                        this.nodesConfigs[msg.id] = false;
                        this.logger.info(
                            `Received hello from ${Object.keys(this.nodesConfigs).length}/${this.targetNumNodes}`,
                        );

                        ws.send(JSON.stringify({ type: 'hello', data: OK }));
                        if (Object.keys(this.nodesConfigs).length === this.targetNumNodes) {
                            const newState = this.state.incrementState();
                            this.logger.info(`Changing state to ${newState}`);

                            this.checkConfigInterval = setInterval(() => this.sendConfigurations(), 2500);
                        }
                    } else {
                        this.logger.error(`Received hello from already known node ${msg.id}`);
                    }
                } else {
                    this.invalidMessageType(msg, 'hello');
                }
                break;
            case 'SEND_CONF':
                if (msg.type === 'conf') {
                    // Node has successfully stored config
                    if (Object.hasOwn(this.nodesConfigs, msg.id) && !this.nodesConfigs[msg.id]) {
                        this.nodesConfigs[msg.id] = true;
                        this.storedConfigs++;
                    }

                    if (this.storedConfigs === this.targetNumNodes) {
                        const newState = this.state.incrementState();
                        this.logger.info(`Chaning state to ${newState}`);

                        this.sendReady();
                    }
                } else {
                    this.invalidMessageType(msg, 'conf');
                }
                break;
            case 'SEND_RDY':
                if (msg.type === 'ready') {
                    if (!Object.hasOwn(this.nodesReady, msg.id)) {
                        this.nodesReady[msg.id] = true;
                    }

                    if (Object.keys(this.nodesReady).length === this.targetNumNodes) {
                        const newState = this.state.incrementState();
                        this.logger.info(`Chaning state to ${newState}`);

                        // Callback to stop the bootstrap service
                        this.exitCallback();
                    }
                }
                break;
            case 'FINISH':
                break;
        }
    }

    sendConfigurations() {
        const configPath = `${this.volumePath}network.json`;
        let configFileExists = false;

        if (fs.existsSync(configPath) && fs.lstatSync(configPath).isFile()) {
            clearInterval(this.checkConfigInterval);

            const fileConfig = fs.readFileSync(configPath).toString();
            const jsonConfig = JSON.parse(fileConfig);

            // Modify json configuration to include unique peer ids
            this.alterConfig(jsonConfig);

            // Send unique configuration to each client
            let i = 0;
            this.logger.info(this.nodesIdsMapping);
            this.wss.clients.forEach((client) => {
                if (client.readyState === WebSocket.OPEN) {
                    client.send(JSON.stringify({ type: 'send_conf', data: jsonConfig.nodes_view[i] }));
                    i++;
                }
            });
        }
        else {
            this.logger.info('Config file not found, retrying...');
        }
    }

    sendReady() {
        this.wss.clients.forEach((client) => {
            if (client.readyState === WebSocket.OPEN) {
                client.send(JSON.stringify({ type: 'send_ready', data: OK }));
            }
        });
    }

    // Modify config inplace
    alterConfig(config) {
        const configIdMapping = {};
        const p2pIds = Object.values(this.nodesIdsMapping);

        for (let i = 0; i < config.nodes_view.length; i++) {
            configIdMapping[config.nodes_view[i].loc.id] = p2pIds[i];
        }

       for (let nodeView of config.nodes_view) {
            for (let connection of nodeView.connections) {
                connection.p2p_id = configIdMapping[connection.id];
            }
       }
    }

    // Log message that was received in other state than expected
    invalidMessageType(msg, expectedType) {
        this.logger.error(`Received ${msg.type} (expected: ${expectedType}) during ${this.state.getState()}`);
    }
}
