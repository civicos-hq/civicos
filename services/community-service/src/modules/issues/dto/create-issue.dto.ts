import { IsString, IsEnum, IsOptional, MinLength } from 'class-validator';
import { IssueCategory } from '@civicos/types';

export class CreateIssueDto {
  @IsString() @MinLength(5) title!: string;
  @IsString() @MinLength(10) description!: string;
  @IsEnum(IssueCategory) category!: IssueCategory;
  @IsString() communityId!: string;
  @IsString() @IsOptional() location?: string;
}
