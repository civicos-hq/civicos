import { Module } from '@nestjs/common';
import { GatewayController } from './gateway.controller';

// The API Gateway proxies requests to downstream services.
// As the platform grows, replace this with @nestjs/microservices or a dedicated proxy.
@Module({
  controllers: [GatewayController],
})
export class AppModule {}
