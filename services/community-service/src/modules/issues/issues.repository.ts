import { Injectable } from '@nestjs/common';
import { PrismaService } from '../../prisma.service';

@Injectable()
export class IssuesRepository {
  constructor(private readonly db: PrismaService) {}

  async findAll(filters: { communityId?: string; status?: string }) {
    return this.db.issue.findMany({
      where: {
        ...(filters.communityId && { communityId: filters.communityId }),
        ...(filters.status && { status: filters.status as never }),
      },
      orderBy: { createdAt: 'desc' },
    });
  }

  async findById(id: string) {
    return this.db.issue.findUnique({ where: { id } });
  }

  async create(data: { title: string; description: string; category: string; communityId: string; reportedById: string; location?: string }) {
    return this.db.issue.create({ data: data as never });
  }

  async incrementUpvote(id: string, _userId: string) {
    return this.db.issue.update({ where: { id }, data: { upvoteCount: { increment: 1 } } });
  }

  async updateStatus(id: string, status: string) {
    return this.db.issue.update({ where: { id }, data: { status: status as never } });
  }
}
