import { Injectable, Logger, OnModuleInit } from '@nestjs/common';
import amqp, { ChannelWrapper } from 'amqp-connection-manager';
import { Channel } from 'amqplib';
import { CommandService } from 'src/commands/command.service';
import {
    MessageType,
    RMQMessageGenesisBlockBuild,
    RMQMessagePing,
} from 'src/commands/command.types';

import { appendFile, writeFile } from 'fs';
import * as path from 'path';
import { exit } from 'process';

@Injectable()
export class ConsumerService implements OnModuleInit {
    private readonly logger = new Logger(ConsumerService.name);
    private readonly dataCsvHeader =
        'nodeId,row,col,depth,transactionsCount,hash,number,createdAt,merkleRoot,author,previousBlockHash,previousSecondBlockHash,commitmentHash,commitmentSecret\n';
    private genesisBlockLogged: boolean;

    private channelWrapper: ChannelWrapper;
    dataPath: any;
    constructor(private readonly commandService: CommandService) {
        const connection = amqp.connect(process.env['RABBITMQ_URL']);
        this.channelWrapper = connection.createChannel({
            setup: async (channel: Channel) => {
                await channel.assertExchange('blockchain_data', 'direct', {
                    durable: false,
                });
                await this.channelWrapper.assertQueue('blockchain_data', {
                    durable: false,
                });
                return this.channelWrapper.bindQueue(
                    'blockchain_data',
                    'blockchain_data',
                    '',
                );
            },
        });
        this.genesisBlockLogged = false;
    }

    public async onModuleInit() {
        try {
            if (
                typeof process.env['VOLUME_PATH'] === 'undefined' ||
                process.env['VOLUME_PATH'] === null
            ) {
                this.logger.error('VOLUME_PATH environment variable not set');
                exit(1);
            } else {
                this.dataPath = path.join(
                    process.env['VOLUME_PATH'],
                    'data.csv',
                );
                this.logger.log('Data volume path:', this.dataPath);

                writeFile(this.dataPath, this.dataCsvHeader, (err) => {
                    if (err) {
                        this.logger.error('Error writing data to file', err);
                    }
                });
            }

            await this.channelWrapper.consume(
                'blockchain_data',
                (message) => {
                    this.logger.log(
                        `Received message: ${message.content.toString()}`,
                    );

                    const messageJson = JSON.parse(message.content.toString());

                    switch (messageJson.type) {
                        case MessageType.BCHAIN_INIT_COMMITMENT:
                            break;
                        case MessageType.BCHAIN_COMMITMENT:
                            // Process commitment by storing it in DB, locally, etc.
                            break;
                        case MessageType.BCHAIN_INIT_BUCKET_INFO:
                            break;
                        case MessageType.BCHAIN_BUCKET_CHECK:
                            break;
                        case MessageType.BCHAIN_SYNC_TIME:
                            break;
                        case MessageType.BCHAIN_BLOCK:
                            this.logBlockProcess(messageJson);
                            break;
                        case MessageType.PING:
                            this.processPingMessage(messageJson);
                            break;
                        case MessageType.GENESIS_BLOCK_BUILD:
                            this.processGenesisBlockBuildMessage(messageJson);
                            break;
                        case MessageType.INFO:
                            break;
                        case MessageType.TRANSACTION_PACK:
                            break;
                        case MessageType.PUB_KEY_EXCHANGE:
                            break;
                        case MessageType.NODE_READY:
                            this.commandService.nodeReady(messageJson.nodeId);
                            break;
                        case MessageType.INVALID_MESSAGE:
                            break;
                        default:
                            this.logger.error(
                                'Unknown message type:',
                                message.content.toString(),
                            );
                            break;
                    }
                },
                { noAck: true },
            );
        } catch (error) {
            this.logger.error('Error starting the consumer service:', error);
        }
    }

    private processPingMessage(message: RMQMessagePing) {}

    private processGenesisBlockBuildMessage(
        message: RMQMessageGenesisBlockBuild,
    ) {
        this.commandService.appendPublicKeyToNode(
            message.nodeId,
            message.publicKey,
        );

        if (
            this.commandService.nodesPublicKeys.length >=
            this.commandService.nodes.length
        ) {
            this.commandService.sendGenesisBlock();
        }
    }

    private logBlockProcess(messageJson: any) {
        if (messageJson.nodeId === '-1') {
            return;
        }

        // Log genesis block only once
        if (messageJson.block.author === '0000000000000000000000000000000000000000') {
            if (this.genesisBlockLogged) {
                return;
            }
            else {
                this.genesisBlockLogged = true;
            }
        }

        this.logger.log(
            `${messageJson.nodeId} mined block at row ${messageJson.block.row} for col ${messageJson.block.col}`,
        );

        const csvLine =
            messageJson.nodeId +
            ',' +
            messageJson.block.row +
            ',' +
            messageJson.block.col +
            ',' +
            messageJson.block.depth +
            ',' +
            messageJson.block.transactions.length +
            ',' +
            messageJson.block.hash +
            ',' +
            messageJson.block.number +
            ',' +
            messageJson.block.createdAt +
            ',' +
            messageJson.block.merkleRoot +
            ',' +
            messageJson.block.author +
            ',' +
            messageJson.block.previousBlockHash +
            ',' +
            messageJson.block.previousSecondBlockHash +
            ',' +
            messageJson.block.commitmentHash +
            ',' +
            messageJson.block.commitmentSecret +
            '\n';

        appendFile(this.dataPath, csvLine, (err) => {
            if (err) {
                this.logger.error('Error writing data to file', err);
            }
        });
    }
}
