import { createParamDecorator, type ExecutionContext } from '@nestjs/common';
import type { JwtPayload } from '@civicos/types';

export const CurrentUser = createParamDecorator(
  (_data: unknown, ctx: ExecutionContext): JwtPayload => {
    return ctx.switchToHttp().getRequest().user as JwtPayload;
  },
);
