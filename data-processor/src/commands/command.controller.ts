import { Controller, Get, Param } from '@nestjs/common';
import { CommandService } from './command.service';

@Controller()
export class CommandController {
    constructor(private readonly commandService: CommandService) {}

    @Get('commands/nodes-ready')
    getNodesReady(): string {
        return this.commandService.getNodesReady();
    }

    @Get('commands/ping')
    sendPing(): string {
        this.commandService.sendPing();
        return '';
    }

    @Get('commands/genesis-block-data')
    retrieveGenesisBlockData(): string {
        this.commandService.retrieveGenesisBlockData();
        return '';
    }

    @Get('commands/start-blockchain')
    sendStartBlockchainRequest(): string {
        this.commandService.sendStartBlockchainRequest();
        return '';
    }

    @Get('commands/generate-and-start-blockchain')
    generateAndStartBlockchain(): string {
        this.commandService.generateAndStartBlockchain();
        return '';
    }

    @Get('commands/distribute-public-keys')
    distributePublicKeys(): string {
        this.commandService.sendPublicKeyDistributionRequest();
        return '';
    }

    @Get('commands/send-transaction-pack/:size')
    sendTransactionPack(@Param('size') size: number): string {
        this.commandService.sendTransactionPack(size);
        return '';
    }

    @Get('commands/stop-blockchain')
    stopBlockchain(): string {
        this.commandService.stopBlockchain();
        return '';
    }
}
