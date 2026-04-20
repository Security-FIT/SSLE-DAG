import { Injectable, Logger } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import {
    Block,
    GenesisBlock as GenesisBlock,
    GenesisTransaction,
    NodeInfo,
    NodePublicKeys,
    Transaction,
} from './command.types';
import { ProducerService } from 'src/rabbitmq/producer.service';
import { CommandsMessageTypes } from './command.types';
import { DateTime } from 'luxon';
import { v4 as uuidv4 } from 'uuid';

const TOKEN_FRACTION_SIZE = 1000;

@Injectable()
export class CommandService {
    nodes: NodeInfo[];
    nodesReady: string[];
    nodesPublicKeys: NodePublicKeys[];
    genesisBlockReady: boolean;

    blocks: any[];

    sendStartBlockchainTogether: boolean;

    constructor(
        private readonly configService: ConfigService,
        private readonly producerService: ProducerService,
    ) {
        this.nodes = this.configService.getOrThrow('nodes');
        Logger.log(
            `Node length loaded from configuration: ${this.nodes.length}`,
        );

        this.nodesPublicKeys = [];
        this.nodesReady = [];
        this.genesisBlockReady = false;
        this.blocks = [];

        this.sendStartBlockchainTogether = false;
    }

    sendPing() {
        // const nodes: NodeInfo[] = this.configService.getOrThrow('nodes');
        // Logger.log(nodes);
        this.producerService.publishMessage(
            {
                type: CommandsMessageTypes.EXECUTE_PING,
                headers: {},
                genesisBlock: null,
            },
            this.nodes[0].id,
        );
    }

    retrieveGenesisBlockData() {
        // Randomly select node from the list of nodes
        const nodeToPublish =
            this.nodes[Math.floor(Math.random() * this.nodes.length)];

        this.producerService.publishMessage(
            {
                type: CommandsMessageTypes.EXECUTE_GENESIS_BLOCK_BUILD,
                headers: {},
                genesisBlock: null,
            },
            nodeToPublish.id,
        );
    }

    buildGensisBlock(): GenesisBlock {
        const coinbase = '0000000000000000000000000000000000000000';
        const timestamp = DateTime.now().toUnixInteger();

        const transactions: GenesisTransaction[] = [];
        for (const node of this.nodesPublicKeys) {
            const tx: GenesisTransaction = {
                createdAt: timestamp,
                recipient: node.publicKey,
                sender: coinbase,
                amount: 32 * TOKEN_FRACTION_SIZE,
            };

            transactions.push(tx);
        }

        const block: GenesisBlock = {
            // id: '00000000-0000-0000-0000-000000000000', // UUID with zeroes only
            hash: '0000000000000000000000000000000000000000000000000000000000000000',
            number: 0,
            createdAt: timestamp,
            transactions: transactions,
            author: coinbase,
        };

        this.blocks.push(block);
        return block;
    }

    async sendGenesisBlock() {
        // Build genesis block if does not exist
        let genesisBlock: GenesisBlock;
        if (!this.genesisBlockReady) {
            genesisBlock = this.buildGensisBlock();
            this.genesisBlockReady = true;
        } else {
            genesisBlock = this.blocks[0];
        }

        Logger.log('Publishing genesis block:');
        // Store block into DB
        // ...

        // Publish block to all nodes
        for (const node of this.nodes) {
            Logger.log('Publishing genesis block to node', node.id);

            this.producerService.publishMessage(
                {
                    type: CommandsMessageTypes.RETRIEVE_GENESIS_BLOCK,
                    headers: {},
                    genesisBlock: genesisBlock,
                },
                node.id,
            );
        }

        if (this.sendStartBlockchainTogether) {
            setTimeout(() => {
                this.sendStartBlockchainRequest();
            }, 1000);
        }
    }

    async sendTransactionPack(numberOfTransactions: number) {
        const transactions: Transaction[] = [];

        for (let i = 0; i < numberOfTransactions; i++) {
            let hash = uuidv4().replaceAll('-', '');

            const tx: Transaction = {
                hash: hash,
                createdAt: DateTime.now().toUnixInteger(),
                recipient: 'A',
                sender: 'B',
                amount: 2 * TOKEN_FRACTION_SIZE,
            };

            transactions.push(tx);
        }

        // Publish transaction pack to all nodes
        for (const node of this.nodes) {
            Logger.log('Publishing transaction pack to node', node.id);

            this.producerService.publishMessage(
                {
                    type: CommandsMessageTypes.GATHER_TRANSACTION_PACK,
                    headers: {},
                    genesisBlock: null,
                    transactions: transactions,
                },
                node.id,
            );
        }
    }

    async sendStartBlockchainRequest() {
        this.producerService.publishMessage(
            {
                type: CommandsMessageTypes.START_BLOCKCHAIN,
                headers: {},
                genesisBlock: null,
            },
            this.nodes[0].id,
        );
    }

    async sendPublicKeyDistributionRequest() {
        this.producerService.publishMessage(
            {
                type: CommandsMessageTypes.REQUEST_PUBLIC_KEY_DISTRIBUTION,
                headers: {},
                genesisBlock: null,
            },
            this.nodes[0].id,
        );
    }

    async stopBlockchain() {
        // Publish stop blockchain command to all nodes
        for (const node of this.nodes) {
            Logger.log(
                'Publishing stop blockchain command pack to node',
                node.id,
            );
            this.producerService.publishMessage(
                {
                    type: CommandsMessageTypes.STOP_BLOCKCHAIN,
                    headers: {},
                    genesisBlock: null,
                },
                node.id,
            );
        }
    }

    async nodeReady(nodeId: string) {
        Logger.log('Node is ready:', nodeId);
        this.nodesReady.push(nodeId);
    }

    getNodesReady(): string {
        return `${this.nodesReady.length}/${this.nodes.length}`;
    }

    generateAndStartBlockchain() {
        this.sendStartBlockchainTogether = true;
        this.retrieveGenesisBlockData();
    }

    appendPublicKeyToNode(nodeId: string, publicKey: string) {
        this.nodesPublicKeys.push({
            id: nodeId,
            publicKey: publicKey,
        });
    }

    isUndefinedOrNull(obj: Object): boolean {
        return typeof obj === 'undefined' || obj === null;
    }
}
