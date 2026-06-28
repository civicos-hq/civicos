import { Injectable, NotFoundException } from '@nestjs/common';
import { CommunitiesRepository } from './communities.repository';
import type { CreateCommunityDto } from './dto/create-community.dto';

@Injectable()
export class CommunitiesService {
  constructor(private readonly repository: CommunitiesRepository) {}

  async findAll() { return this.repository.findAll(); }

  async findOne(id: string) {
    const community = await this.repository.findById(id);
    if (!community) throw new NotFoundException('Community not found');
    return community;
  }

  async create(dto: CreateCommunityDto, createdById: string) {
    return this.repository.create({ ...dto, createdById });
  }

  async join(communityId: string, userId: string) {
    return this.repository.addMember(communityId, userId);
  }
}
