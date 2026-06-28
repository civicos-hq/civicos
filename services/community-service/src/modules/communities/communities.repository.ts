import { Injectable } from '@nestjs/common';
import { PrismaService } from '../../prisma.service';

@Injectable()
export class CommunitiesRepository {
  constructor(private readonly db: PrismaService) {}

  async findAll() {
    return this.db.community.findMany({ orderBy: { name: 'asc' } });
  }

  async findById(id: string) {
    return this.db.community.findUnique({ where: { id } });
  }

  async create(data: { name: string; slug: string; state: string; lga: string; createdById: string }) {
    return this.db.community.create({ data });
  }

  async addMember(communityId: string, userId: string) {
    // TODO(civicos-community-1): implement community membership table
    return { communityId, userId, joined: true };
  }
}
