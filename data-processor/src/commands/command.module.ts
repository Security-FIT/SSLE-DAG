import { forwardRef, Module } from '@nestjs/common';
import { CommandService } from './command.service';
import { CommandController } from './command.controller';
import { RabbitmqModule } from 'src/rabbitmq/rabbitmq.module';

@Module({
    imports: [forwardRef(() => RabbitmqModule)],
    providers: [CommandService],
    exports: [CommandService],
    controllers: [CommandController],
})
export class CommandModule {}
