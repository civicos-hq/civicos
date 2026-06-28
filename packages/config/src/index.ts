import { z } from 'zod';

// Configuration is validated at startup — the app exits with a clear error
// if any required variable is missing or malformed (Playbook: Ch. 3, Config Management)

const baseSchema = z.object({
  NODE_ENV: z.enum(['development', 'test', 'production']).default('development'),
  DATABASE_URL: z.string().url(),
  REDIS_URL: z.string().url(),
  NATS_URL: z.string().url(),
  JWT_SECRET: z.string().min(32, 'JWT_SECRET must be at least 32 characters'),
  JWT_EXPIRES_IN: z.string().default('7d'),
  JWT_REFRESH_EXPIRES_IN: z.string().default('30d'),
});

const identitySchema = baseSchema.extend({
  IDENTITY_SERVICE_PORT: z.coerce.number().default(3001),
});

const communitySchema = baseSchema.extend({
  COMMUNITY_SERVICE_PORT: z.coerce.number().default(3002),
  STORAGE_ENDPOINT: z.string().optional(),
  STORAGE_ACCESS_KEY: z.string().optional(),
  STORAGE_SECRET_KEY: z.string().optional(),
  STORAGE_BUCKET: z.string().default('civicos-uploads'),
});

const gatewaySchema = baseSchema.extend({
  API_GATEWAY_PORT: z.coerce.number().default(3000),
  IDENTITY_SERVICE_URL: z.string().url().default('http://localhost:3001'),
  COMMUNITY_SERVICE_URL: z.string().url().default('http://localhost:3002'),
});

export type BaseEnv = z.infer<typeof baseSchema>;
export type IdentityEnv = z.infer<typeof identitySchema>;
export type CommunityEnv = z.infer<typeof communitySchema>;
export type GatewayEnv = z.infer<typeof gatewaySchema>;

function parse<T extends z.ZodTypeAny>(schema: T): z.infer<T> {
  const result = schema.safeParse(process.env);
  if (!result.success) {
    console.error('❌ Invalid environment variables:');
    console.error(JSON.stringify(result.error.flatten().fieldErrors, null, 2));
    process.exit(1);
  }
  return result.data;
}

export const validateIdentityEnv = () => parse(identitySchema);
export const validateCommunityEnv = () => parse(communitySchema);
export const validateGatewayEnv = () => parse(gatewaySchema);
