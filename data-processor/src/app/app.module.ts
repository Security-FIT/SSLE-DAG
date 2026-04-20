import { Module } from '@nestjs/common';
import { AppController } from './app.controller';
import { AppService } from './app.service';
import { ConfigModule } from '@nestjs/config';
import configuration from 'src/config/configuration';
import { CommandModule } from 'src/commands/command.module';

@Module({
    imports: [
        // first import as first initialization
        ConfigModule.forRoot({
            isGlobal: true, // to get access to it in every component
            load: [configuration],
        }),
        CommandModule,
    ],
    controllers: [AppController],
    providers: [AppService],
})
export class AppModule {}
