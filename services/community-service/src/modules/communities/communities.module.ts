import { Module } from '@nestjs/common';
import { CommunitiesController } from './communities.controller';
import { CommunitiesService } from './communities.service';
import { CommunitiesRepository } from './communities.repository';
import { PrismaService } from '../../prisma.service';

@Module({
  controllers: [CommunitiesController],
  providers: [CommunitiesService, CommunitiesRepository, PrismaService],
  exports: [CommunitiesService],
})
export class CommunitiesModule {}
