import { IsString, MinLength, MaxLength, IsOptional } from 'class-validator';

export class CreateCommunityDto {
  @IsString() @MinLength(2) @MaxLength(100) name!: string;
  @IsString() @MinLength(2) slug!: string;
  @IsString() state!: string;
  @IsString() lga!: string;
  @IsString() @IsOptional() description?: string;
}
