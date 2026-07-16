import {
  S3Client,
  PutObjectCommand,
  GetObjectCommand,
  HeadBucketCommand,
  CreateBucketCommand,
} from "@aws-sdk/client-s3";
import { getSignedUrl } from "@aws-sdk/s3-request-presigner";
import CircuitBreaker from "opossum";
import type { Config } from "../config.js";

export function createS3Client(config: Config): S3Client {
  return new S3Client({
    endpoint: config.minio.endpoint,
    region: config.minio.region,
    forcePathStyle: true,
    credentials: {
      accessKeyId: config.minio.accessKey,
      secretAccessKey: config.minio.secretKey,
    },
  });
}

export interface UploadParams {
  bucket: string;
  key: string;
  body: Buffer;
  contentType: string;
}

async function putObject(
  client: S3Client,
  params: UploadParams,
): Promise<void> {
  await client.send(
    new PutObjectCommand({
      Bucket: params.bucket,
      Key: params.key,
      Body: params.body,
      ContentType: params.contentType,
    }),
  );
}

async function presignGetObject(
  client: S3Client,
  bucket: string,
  key: string,
  expiresIn: number,
): Promise<string> {
  return getSignedUrl(
    client,
    new GetObjectCommand({ Bucket: bucket, Key: key }),
    { expiresIn },
  );
}

const BREAKER_OPTIONS = {
  timeout: 10_000,
  errorThresholdPercentage: 50,
  resetTimeout: 30_000,
  volumeThreshold: 5,
};

export function createUploadBreaker(client: S3Client) {
  return new CircuitBreaker(
    (params: UploadParams) => putObject(client, params),
    BREAKER_OPTIONS,
  );
}

export function createPresignBreaker(client: S3Client) {
  return new CircuitBreaker(
    (bucket: string, key: string, expiresIn: number) =>
      presignGetObject(client, bucket, key, expiresIn),
    BREAKER_OPTIONS,
  );
}

export async function ensureBucket(
  client: S3Client,
  bucket: string,
): Promise<void> {
  try {
    await client.send(new HeadBucketCommand({ Bucket: bucket }));
  } catch {
    await client.send(new CreateBucketCommand({ Bucket: bucket }));
  }
}
