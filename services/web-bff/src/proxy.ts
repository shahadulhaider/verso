import { request as undiciRequest } from "undici";
import type { FastifyRequest, FastifyReply } from "fastify";

export interface ProxyOptions {
  /** Full backend URL (e.g. http://verso-identity-service:8001/v1/auth/login) */
  target: string;
  /** HTTP method override; defaults to the incoming request method */
  method?: string;
}

/**
 * Forward an incoming Fastify request to a backend service.
 *
 * - Forwards Authorization header if present
 * - Forwards Content-Type header if present
 * - Forwards request body for non-GET requests
 * - Streams the backend response back with status + headers
 * - 5-second timeout
 */
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
    const res = await undiciRequest(opts.target, {
      method: method as "GET" | "POST" | "PUT" | "DELETE" | "PATCH",
      headers,
      body: hasBody ? JSON.stringify(req.body) : undefined,
      signal: AbortSignal.timeout(5000),
    });

    // Forward response content-type
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
