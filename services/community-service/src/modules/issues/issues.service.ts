import { Injectable, NotFoundException } from '@nestjs/common';
import { IssuesRepository } from './issues.repository';
import type { CreateIssueDto } from './dto/create-issue.dto';

@Injectable()
export class IssuesService {
  constructor(private readonly repository: IssuesRepository) {}

  async findAll(filters: { communityId?: string; status?: string }) {
    return this.repository.findAll(filters);
  }

  async findOne(id: string) {
    const issue = await this.repository.findById(id);
    if (!issue) throw new NotFoundException('Issue not found');
    return issue;
  }

  async create(dto: CreateIssueDto, reportedById: string) {
    return this.repository.create({ ...dto, reportedById });
  }

  async upvote(id: string, userId: string) {
    return this.repository.incrementUpvote(id, userId);
  }

  async updateStatus(id: string, status: string) {
    return this.repository.updateStatus(id, status);
  }
}
