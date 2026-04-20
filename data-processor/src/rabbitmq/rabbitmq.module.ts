import { forwardRef, Module } from '@nestjs/common';
import { ProducerService } from './producer.service';
import { ConsumerService } from './consumer.service';
import { CommandService } from 'src/commands/command.service';
import { CommandModule } from 'src/commands/command.module';

@Module({
  providers: [ProducerService, ConsumerService],
  imports: [forwardRef(() => CommandModule)],
  exports: [ProducerService],
})
export class RabbitmqModule {}