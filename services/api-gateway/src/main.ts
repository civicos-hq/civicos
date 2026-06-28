import 'reflect-metadata';
import { NestFactory } from '@nestjs/core';
import { FastifyAdapter, type NestFastifyApplication } from '@nestjs/platform-fastify';
import { Logger } from '@nestjs/common';
import { AppModule } from './app.module';
import { validateGatewayEnv } from '@civicos/config';

async function bootstrap(): Promise<void> {
  const env = validateGatewayEnv();
  const logger = new Logger('ApiGateway');

  const app = await NestFactory.create<NestFastifyApplication>(
    AppModule,
    new FastifyAdapter({ logger: false }),
  );

  app.enableCors({ origin: process.env.ALLOWED_ORIGIN ?? 'http://localhost:5173', credentials: true });

  await app.listen(env.API_GATEWAY_PORT, '0.0.0.0');
  logger.log(`API Gateway running on port ${env.API_GATEWAY_PORT}`);
  logger.log(`→ Identity Service: ${env.IDENTITY_SERVICE_URL}`);
  logger.log(`→ Community Service: ${env.COMMUNITY_SERVICE_URL}`);
}

bootstrap();
