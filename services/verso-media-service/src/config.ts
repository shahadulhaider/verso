import { z } from "zod";

const configSchema = z.object({
  port: z.coerce.number().default(8005),
  databaseUrl: z.string(),
  minio: z.object({
    endpoint: z.string(),
    accessKey: z.string(),
    secretKey: z.string(),
    bucket: z.string().default("verso-media"),
    region: z.string().default("us-east-1"),
  }),
  jwksUrl: z.string(),
  presignedUrlExpiry: z.coerce.number().default(3600),
  maxFileSize: z.coerce.number().default(10 * 1024 * 1024),
});

export type Config = z.infer<typeof configSchema>;

export function loadConfig(): Config {
  return configSchema.parse({
    port: process.env.SERVICE_PORT,
    databaseUrl: process.env.DATABASE_URL,
    minio: {
      endpoint: process.env.MINIO_ENDPOINT,
      accessKey: process.env.MINIO_ACCESS_KEY,
      secretKey: process.env.MINIO_SECRET_KEY,
      bucket: process.env.MINIO_BUCKET,
      region: process.env.MINIO_REGION,
    },
    jwksUrl: process.env.JWKS_URL,
    presignedUrlExpiry: process.env.PRESIGNED_URL_EXPIRY,
    maxFileSize: process.env.MAX_FILE_SIZE,
  });
}
