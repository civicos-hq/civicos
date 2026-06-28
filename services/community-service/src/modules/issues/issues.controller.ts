import { Controller, Get, Post, Patch, Body, Param, Query, UseGuards } from '@nestjs/common';
import { IssuesService } from './issues.service';
import { CreateIssueDto } from './dto/create-issue.dto';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';
import type { JwtPayload } from '@civicos/types';

@Controller('issues')
export class IssuesController {
  constructor(private readonly issuesService: IssuesService) {}

  @Get()
  findAll(@Query('communityId') communityId?: string, @Query('status') status?: string) {
    return this.issuesService.findAll({ communityId, status });
  }

  @Get(':id')
  findOne(@Param('id') id: string) { return this.issuesService.findOne(id); }

  @Post()
  @UseGuards(JwtAuthGuard)
  create(@Body() dto: CreateIssueDto, @CurrentUser() user: JwtPayload) {
    return this.issuesService.create(dto, user.sub);
  }

  @Post(':id/upvote')
  @UseGuards(JwtAuthGuard)
  upvote(@Param('id') id: string, @CurrentUser() user: JwtPayload) {
    return this.issuesService.upvote(id, user.sub);
  }

  @Patch(':id/status')
  @UseGuards(JwtAuthGuard)
  updateStatus(@Param('id') id: string, @Body('status') status: string) {
    return this.issuesService.updateStatus(id, status);
  }
}
