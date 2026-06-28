import { Module } from '@nestjs/common';
import { IssuesController } from './issues.controller';
import { IssuesService } from './issues.service';
import { IssuesRepository } from './issues.repository';
import { PrismaService } from '../../prisma.service';

@Module({
  controllers: [IssuesController],
  providers: [IssuesService, IssuesRepository, PrismaService],
})
export class IssuesModule {}
