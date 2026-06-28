import { Module } from '@nestjs/common';
import { CommunitiesModule } from './modules/communities/communities.module';
import { IssuesModule } from './modules/issues/issues.module';

@Module({
  imports: [CommunitiesModule, IssuesModule],
})
export class AppModule {}
