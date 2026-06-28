import 'reflect-metadata';
import { NestFactory } from '@nestjs/core';
import { FastifyAdapter, type NestFastifyApplication } from '@nestjs/platform-fastify';
import { ValidationPipe, Logger } from '@nestjs/common';
import { AppModule } from './app.module';
import { validateIdentityEnv } from '@civicos/config';

async function bootstrap(): Promise<void> {
  const env = validateIdentityEnv();
  const logger = new Logger('IdentityService');

  const app = await NestFactory.create<NestFastifyApplication>(
    AppModule,
    new FastifyAdapter({ logger: false }),
  );

  app.setGlobalPrefix('api');

  // Validate and transform incoming DTOs automatically
  app.useGlobalPipes(
    new ValidationPipe({
      whitelist: true,       // Strip unknown properties
      forbidNonWhitelisted: true,
      transform: true,
      transformOptions: { enableImplicitConversion: true },
    }),
  );

  app.enableCors({ origin: process.env.ALLOWED_ORIGIN ?? 'http://localhost:5173', credentials: true });

  await app.listen(env.IDENTITY_SERVICE_PORT, '0.0.0.0');
  logger.log(`Identity Service running on port ${env.IDENTITY_SERVICE_PORT}`);
}

bootstrap();
