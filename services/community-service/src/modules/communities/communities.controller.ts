import { Controller, Get, Post, Body, Param, UseGuards } from '@nestjs/common';
import { CommunitiesService } from './communities.service';
import { CreateCommunityDto } from './dto/create-community.dto';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';
import type { JwtPayload } from '@civicos/types';

@Controller('communities')
export class CommunitiesController {
  constructor(private readonly communitiesService: CommunitiesService) {}

  @Get()
  findAll() { return this.communitiesService.findAll(); }

  @Get(':id')
  findOne(@Param('id') id: string) { return this.communitiesService.findOne(id); }

  @Post()
  @UseGuards(JwtAuthGuard)
  create(@Body() dto: CreateCommunityDto, @CurrentUser() user: JwtPayload) {
    return this.communitiesService.create(dto, user.sub);
  }

  @Post(':id/join')
  @UseGuards(JwtAuthGuard)
  join(@Param('id') id: string, @CurrentUser() user: JwtPayload) {
    return this.communitiesService.join(id, user.sub);
  }
}
