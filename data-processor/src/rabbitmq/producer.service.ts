import {
    HttpException,
    HttpStatus,
    Injectable,
} from '@nestjs/common';
import amqp, { ChannelWrapper } from 'amqp-connection-manager';
import { Channel } from 'amqplib';
import { Logger } from '@nestjs/common';
import { CommandMessage } from 'src/commands/command.types';

@Injectable()
export class ProducerService {
    private readonly logger = new Logger(ProducerService.name);

    private channelWrapper: ChannelWrapper;
    constructor() {
        const connection = amqp.connect(process.env['RABBITMQ_URL']);
        this.channelWrapper = connection.createChannel({
            setup: (channel: Channel) => {
                return channel.assertExchange('commands_exchange', 'direct', {
                    durable: false,
                });
            },
        });
    }

    async publishMessage(message: CommandMessage, routingKey: string) {
        try {
            const messageStr = JSON.stringify(message);
            this.logger.log(
                `Publishing message: ${messageStr} with routing key: ${routingKey}`,
            );
            await this.channelWrapper.publish(
                'commands_exchange',
                routingKey,
                Buffer.from(messageStr, 'utf8'),
            );
        } catch (error) {
            this.logger.error(error);
            throw new HttpException(
                'Erro sending a message to the queue',
                HttpStatus.INTERNAL_SERVER_ERROR,
            );
        }
    }
}
