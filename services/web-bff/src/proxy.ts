import { request as undiciRequest } from "undici";
import CircuitBreaker from "opossum";
import type { FastifyRequest, FastifyReply } from "fastify";

export interface ProxyOptions {
  target: string;
  method?: string;
}

interface ProxyCallArgs {
  target: string;
  method: string;
  headers: Record<string, string>;
  body: string | undefined;
}

async function rawProxy(args: ProxyCallArgs) {
  const res = await undiciRequest(args.target, {
    method: args.method as "GET" | "POST" | "PUT" | "DELETE" | "PATCH",
    headers: args.headers,
    body: args.body,
    signal: AbortSignal.timeout(5000),
  });
  const body = await res.body.text();
  return { statusCode: res.statusCode, headers: res.headers, body };
}

const breaker = new CircuitBreaker(rawProxy, {
  timeout: 6000,
  errorThresholdPercentage: 50,
  resetTimeout: 10000,
  volumeThreshold: 5,
});

export async function proxyRequest(
  req: FastifyRequest,
  reply: FastifyReply,
  opts: ProxyOptions,
): Promise<void> {
  const method = opts.method ?? req.method;

  const headers: Record<string, string> = {};
  if (req.headers.authorization) {
    headers["authorization"] = req.headers.authorization;
  }
  if (req.headers["content-type"]) {
    headers["content-type"] = req.headers["content-type"] as string;
  }

  const hasBody = method !== "GET" && method !== "HEAD" && req.body != null;

  try {
    const res = await breaker.fire({
      target: opts.target,
      method,
      headers,
      body: hasBody ? JSON.stringify(req.body) : undefined,
    });

    const contentType = res.headers["content-type"];
    if (contentType) {
      reply.header("content-type", contentType);
    }

    reply.status(res.statusCode);
    reply.send(res.body);
  } catch (err: unknown) {
    if (err instanceof DOMException && err.name === "TimeoutError") {
      reply
        .status(504)
        .header("content-type", "application/problem+json")
        .send({
          type: "https://httpstatuses.com/504",
          title: "Gateway Timeout",
          status: 504,
          detail: "Backend service did not respond within 5 seconds",
        });
      return;
    }

    req.log.error(err, "Proxy request failed");
    reply
      .status(502)
      .header("content-type", "application/problem+json")
      .send({
        type: "https://httpstatuses.com/502",
        title: "Bad Gateway",
        status: 502,
        detail: "Backend service unavailable",
      });
  }
}
