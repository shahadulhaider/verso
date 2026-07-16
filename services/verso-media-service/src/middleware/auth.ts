import type { FastifyRequest, FastifyReply } from "fastify";
import { verifyToken } from "../lib/jwt.js";

export async function requireAuth(
  request: FastifyRequest,
  reply: FastifyReply,
): Promise<void> {
  const header = request.headers.authorization;
  if (!header?.startsWith("Bearer ")) {
    reply
      .code(401)
      .send({ error: "unauthorized", message: "Missing or invalid Authorization header" });
    return;
  }

  try {
    const token = header.slice(7);
    const payload = await verifyToken(token);
    (request as FastifyRequest & { userId: string }).userId = payload.sub;
  } catch {
    reply
      .code(401)
      .send({ error: "unauthorized", message: "Invalid or expired token" });
  }
}
