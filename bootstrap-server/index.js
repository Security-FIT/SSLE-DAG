import fs from 'fs';
import YAML from 'yaml';
import { WebSocketServer } from 'ws';
import pino from 'pino';
import { exit } from 'process';
import { v4 as uuidv4 } from 'uuid';
import * as dotenv from 'dotenv'

import State from './src/State.js';
import Communication from './src/Communication.js';
import Config from './src/Config.js';

const logger = pino();
dotenv.config();

const VOLUME_ENV = 'VOLUME_PATH';
let pathToLoad;

// Load input configuration
if (Object.hasOwn(process.env, VOLUME_ENV)) {
    logger.info('Environment volume path loaded');
    if (fs.lstatSync(process.env[VOLUME_ENV]).isDirectory()) {
        logger.info('Volume loaded successfully, production environment ready...');
        pathToLoad = `${process.env[VOLUME_ENV]}config.yaml`;
    } else {
        logger.error('Environment volume path not loaded, aborting...');
        exit(1);
    }
} else {
    // Load default config
    logger.error('Environment volume path not loaded, aborting...');
    exit(1);
}

const fileConfig = fs.readFileSync(pathToLoad).toString();
const config = YAML.parse(fileConfig);

const PORT = config.bootstrap.port;
const TARGET_NUM_NODES = config.environment.nodes;

const configChecker = new Config(logger);

logger.info(`Configuration loaded`);
logger.info(config);
if (!configChecker.checkConfig(config)) {
    logger.error('Invalid configuration. Aborting...');
    exit(1);
}
logger.info(`Configuration valid`);

const wss = new WebSocketServer({ port: PORT });

// Start with the BOOTING state
const state = new State();
const communication = new Communication(
    logger,
    wss,
    state,
    TARGET_NUM_NODES,
    Object.hasOwn(process.env, VOLUME_ENV) ? process.env[VOLUME_ENV] : '.',
    stopBootstrapServer,
);
logger.info(`Starting in state ${state.state}`);

wss.on('connection', (ws) => {
    ws.on('error', logger.error);
    ws.id = uuidv4();
    
    ws.on('message', (data) => {
        communication.processMessage(ws, data);
    });
});

wss.on('listening', () => {
    // Change state from BOOTING
    const newState = state.incrementState();
    logger.info(`Changing state to ${newState}`);

    logger.info(`Expecting ${TARGET_NUM_NODES} nodes to connect...`);
});

function stopBootstrapServer() {
    logger.info('Bootstrapping successfuly finished');
    logger.info('Shutting down in 10 seconds...');
    // Stop the server after 10 seconds for demonstration
    setTimeout(() => {
        wss.close(() => {
            logger.info('Websocket server stopped');
            exit(0);
        });
    }, 10000);
}
