import { NestFactory } from '@nestjs/core';
import { AppModule } from './app/app.module';
import { Logger } from '@nestjs/common';

async function bootstrap() {  
  const app = await NestFactory.create(AppModule);

  const HTTP_PORT = process.env.PORT ?? 8000;
  app.enableCors({
    origin: '*',
    methods: 'GET,HEAD,PUT,PATCH,POST,DELETE'
  });
  await app.listen(HTTP_PORT);

  Logger.log(`🚀 HTTP Server running on http://localhost:${HTTP_PORT}`);
  Logger.log(`📡 RabbitMQ Microservice is listening...`);
}

bootstrap();
