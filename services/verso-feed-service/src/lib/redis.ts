import { Redis } from "ioredis";

let client: Redis | null = null;

export function getRedis(redisUrl: string): Redis {
  if (!client) {
    client = new Redis(redisUrl, {
      maxRetriesPerRequest: 3,
      retryStrategy(times: number) {
        return Math.min(times * 200, 2000);
      },
    });
  }
  return client;
}

export async function closeRedis(): Promise<void> {
  if (client) {
    await client.quit();
    client = null;
  }
}
