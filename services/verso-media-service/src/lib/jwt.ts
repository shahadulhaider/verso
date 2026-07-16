import * as jose from "jose";

let jwks: ReturnType<typeof jose.createRemoteJWKSet> | null = null;

export function initJwks(jwksUrl: string): void {
  jwks = jose.createRemoteJWKSet(new URL(jwksUrl));
}

export interface JwtPayload {
  sub: string;
  [key: string]: unknown;
}

export async function verifyToken(token: string): Promise<JwtPayload> {
  if (!jwks) throw new Error("JWKS not initialized");
  const { payload } = await jose.jwtVerify(token, jwks);
  if (!payload.sub) throw new Error("Token missing sub claim");
  return payload as JwtPayload;
}
