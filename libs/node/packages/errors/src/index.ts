/** RFC 9457 Problem Detail for HTTP APIs. */
export interface ProblemDetail {
  type: string;
  title: string;
  status: number;
  detail?: string;
  instance?: string;
}

/**
 * Create a ProblemDetail object.
 */
export function createProblem(
  status: number,
  title: string,
  detail?: string,
): ProblemDetail {
  return {
    type: `https://httpstatuses.com/${status}`,
    title,
    status,
    ...(detail !== undefined && { detail }),
  };
}

/** Fastify-compatible reply interface (minimal). */
interface FastifyReply {
  status(code: number): FastifyReply;
  header(name: string, value: string): FastifyReply;
  send(payload: unknown): FastifyReply;
}

/**
 * Send a ProblemDetail as an HTTP response (Fastify-compatible).
 * Sets content-type to application/problem+json per RFC 9457.
 */
export function sendProblem(reply: FastifyReply, problem: ProblemDetail): void {
  reply
    .status(problem.status)
    .header("content-type", "application/problem+json")
    .send(problem);
}

// ── Standard problem factories ──────────────────────────────────────

export function notFound(detail?: string): ProblemDetail {
  return createProblem(404, "Not Found", detail);
}

export function badRequest(detail?: string): ProblemDetail {
  return createProblem(400, "Bad Request", detail);
}

export function unauthorized(detail?: string): ProblemDetail {
  return createProblem(401, "Unauthorized", detail);
}

export function forbidden(detail?: string): ProblemDetail {
  return createProblem(403, "Forbidden", detail);
}

export function internalError(detail?: string): ProblemDetail {
  return createProblem(500, "Internal Server Error", detail);
}

export function conflict(detail?: string): ProblemDetail {
  return createProblem(409, "Conflict", detail);
}
