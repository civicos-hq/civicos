import { Controller, All, Req, Res, Param } from '@nestjs/common';
import axios from 'axios';
import type { FastifyRequest, FastifyReply } from 'fastify';

const SERVICE_MAP: Record<string, string> = {
  auth: process.env.IDENTITY_SERVICE_URL ?? 'http://localhost:3001',
  users: process.env.IDENTITY_SERVICE_URL ?? 'http://localhost:3001',
  communities: process.env.COMMUNITY_SERVICE_URL ?? 'http://localhost:3002',
  issues: process.env.COMMUNITY_SERVICE_URL ?? 'http://localhost:3002',
  petitions: process.env.COMMUNITY_SERVICE_URL ?? 'http://localhost:3002',
};

// Simple pass-through gateway — routes /api/<service>/... to the correct service.
// TODO(civicos-gateway-1): add rate limiting, request tracing, and auth middleware.
@Controller('api')
export class GatewayController {
  @All(':service/*')
  async proxy(
    @Param('service') service: string,
    @Req() req: FastifyRequest,
    @Res() reply: FastifyReply,
  ): Promise<void> {
    const upstream = SERVICE_MAP[service];
    if (!upstream) {
      reply.code(404).send({ success: false, code: 'SERVICE_NOT_FOUND', message: `Unknown service: ${service}` });
      return;
    }

    const url = `${upstream}${req.url}`;
    try {
      const response = await axios({
        method: req.method as never,
        url,
        data: req.body,
        headers: { ...req.headers, host: undefined },
        validateStatus: () => true,
      });
      reply.code(response.status).send(response.data);
    } catch (err) {
      reply.code(502).send({ success: false, code: 'UPSTREAM_ERROR', message: 'Service unavailable' });
    }
  }
}
