import { JWKSValidator, type Claims } from "@verso/jwt";

let validator: JWKSValidator | null = null;

export type JwtPayload = Claims & { [key: string]: unknown };

export function initJwks(jwksUrl: string): void {
  validator = new JWKSValidator(jwksUrl);
}

export async function verifyToken(token: string): Promise<JwtPayload> {
  if (!validator) throw new Error("JWKS not initialized");
  return validator.validate(token) as Promise<JwtPayload>;
}
