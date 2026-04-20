import { DateTime } from 'luxon';

export enum MessageType {
    BCHAIN_INIT_COMMITMENT,
    BCHAIN_COMMITMENT,
    BCHAIN_INIT_BUCKET_INFO,
    BCHAIN_BUCKET_CHECK,
    BCHAIN_SYNC_TIME,
    BCHAIN_BLOCK,
    PING,
    GENESIS_BLOCK_BUILD,
    INFO,
    TRANSACTION_PACK,
    PUB_KEY_EXCHANGE,
    NODE_READY,
    INVALID_MESSAGE,
}

export type RMQMessagePing = {
    id: string;
    type: MessageType;
    nodeId: string;
    nodeTime: number;
};

export type RMQMessageGenesisBlockBuild = {
    type: MessageType;
    nodeId: string;
    publicKey: string;
};

export enum CommandsMessageTypes {
    EXECUTE_PING,
    EXECUTE_GENESIS_BLOCK_BUILD,
    RETRIEVE_GENESIS_BLOCK,
    START_BLOCKCHAIN,
    REQUEST_PUBLIC_KEY_DISTRIBUTION,
    GATHER_TRANSACTION_PACK,
    STOP_BLOCKCHAIN,
}

export type CommandMessage = {
    type: CommandsMessageTypes;
    headers: object;
    genesisBlock: GenesisBlock | null;
    transactions?: Transaction[];
};

// ====================
export type Block = {
    hash: string;
    number: number;
    createdAt: number; // timestamp
    transactions: Transaction[];
    merkleRoot: string;
    author: string; // public key
    previousBlockHash: string;
    previousSecondBlockHash: string;
    depth: number;
    row: number;
    col: number;
    commitmentHash: string;
    commitmentSecret: string;
};

export type Transaction = {
    hash: string;
    createdAt: number; // timestamp
    recipient: string; // public key
    sender: string; // public key
    amount: number;
};

// Special Genesis structures that omit hashes as they are computed inside nodes
export type GenesisBlock = {
    hash: string;
    number: number;
    createdAt: number; // timestamp
    transactions: GenesisTransaction[];
    author: string; // public key
};

export type GenesisTransaction = {
    createdAt: number; // timestamp
    recipient: string; // public key
    sender: string; // public key
    amount: number;
};

// Configuration specification
export type NetworkConfig = {
    hist_labels: number[];
    hist_values: number[];
    nodes: NodeIdentity[];
    edges: Edge[];
    nodes_view: NodeView[];
};

export type NodeIdentity = {
    id: number;
    label: string;
};

export type Edge = {
    from: number;
    to: number;
    label: string;
};

export type NodeView = {
    loc: Location;
    connections: Connection[];
};

export type Location = {
    id: string;
    name: string;
    lat: string;
    lon: string;
};

export type Connection = {
    id: string;
    delay: number;
    p2p_id: string;
};

// ====================

export type NodeInfo = {
    id: string;
    name: string;
    lat: string;
    lon: string;
    connections: Connection[];
};

export type NodePublicKeys = {
    id: string;
    publicKey: string;
};
