import 'reflect-metadata';
import { NestFactory } from '@nestjs/core';
import { FastifyAdapter, type NestFastifyApplication } from '@nestjs/platform-fastify';
import { ValidationPipe, Logger } from '@nestjs/common';
import { AppModule } from './app.module';
import { validateCommunityEnv } from '@civicos/config';

async function bootstrap(): Promise<void> {
  const env = validateCommunityEnv();
  const logger = new Logger('CommunityService');

  const app = await NestFactory.create<NestFastifyApplication>(
    AppModule,
    new FastifyAdapter({ logger: false }),
  );

  app.setGlobalPrefix('api');
  app.useGlobalPipes(new ValidationPipe({ whitelist: true, transform: true }));
  app.enableCors({ origin: process.env.ALLOWED_ORIGIN ?? 'http://localhost:5173', credentials: true });

  await app.listen(env.COMMUNITY_SERVICE_PORT, '0.0.0.0');
  logger.log(`Community Service running on port ${env.COMMUNITY_SERVICE_PORT}`);
}

bootstrap();
