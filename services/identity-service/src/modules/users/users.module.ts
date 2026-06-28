import { Module } from '@nestjs/common';
import { UsersRepository } from './users.repository';
import { PrismaService } from '../../prisma.service';

@Module({
  providers: [UsersRepository, PrismaService],
  exports: [UsersRepository],
})
export class UsersModule {}
