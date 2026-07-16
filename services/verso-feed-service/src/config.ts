import { z } from "zod";

const configSchema = z.object({
  port: z.coerce.number().default(8009),
  databaseUrl: z.string(),
  redisUrl: z.string(),
  kafkaBrokers: z.string().default("redpanda:9092"),
  socialServiceUrl: z.string(),
  jwksUrl: z.string(),
  timelineCacheTtl: z.coerce.number().default(7 * 24 * 60 * 60),
});

export type Config = z.infer<typeof configSchema>;

export function loadConfig(): Config {
  return configSchema.parse({
    port: process.env.SERVICE_PORT,
    databaseUrl: process.env.DATABASE_URL,
    redisUrl: process.env.REDIS_URL,
    kafkaBrokers: process.env.REDPANDA_BROKERS,
    socialServiceUrl: process.env.SOCIAL_SERVICE_URL,
    jwksUrl: process.env.JWKS_URL,
    timelineCacheTtl: process.env.TIMELINE_CACHE_TTL,
  });
}
