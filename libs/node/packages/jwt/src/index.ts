import * as jose from "jose";

/** Validated JWT claims extracted after verification. */
export interface Claims {
  userId: string;
  roles: string[];
  sub: string;
  iss: string;
  exp: number;
}

/**
 * JWKS-based JWT validator.
 * Fetches and caches a remote JWKS endpoint, then verifies token
 * signatures and expiry using the `jose` library.
 */
export class JWKSValidator {
  private readonly jwks: ReturnType<typeof jose.createRemoteJWKSet>;
  private readonly jwksUrl: string;

  /**
   * @param jwksUrl - URL of the JWKS endpoint (e.g. https://auth.example.com/.well-known/jwks.json)
   * @param refreshIntervalMs - How often to refresh the JWKS cache (default: 10 minutes).
   *   Maps to jose's cooldownDuration option.
   */
  constructor(jwksUrl: string, refreshIntervalMs: number = 600_000) {
    this.jwksUrl = jwksUrl;
    this.jwks = jose.createRemoteJWKSet(new URL(jwksUrl), {
      cooldownDuration: refreshIntervalMs,
    });
  }

  /**
   * Validate a JWT token against the remote JWKS.
   * Verifies signature and checks expiry.
   *
   * @param token - Raw JWT string (e.g. from Authorization: Bearer header)
   * @returns Parsed and validated Claims
   * @throws If signature is invalid, token is expired, or claims are missing
   */
  async validate(token: string): Promise<Claims> {
    const { payload } = await jose.jwtVerify(token, this.jwks);

    const userId = (payload.userId as string) ?? (payload.sub as string) ?? "";
    const roles = Array.isArray(payload.roles)
      ? (payload.roles as string[])
      : [];

    if (!payload.sub) {
      throw new Error("JWT missing required 'sub' claim");
    }
    if (!payload.iss) {
      throw new Error("JWT missing required 'iss' claim");
    }
    if (typeof payload.exp !== "number") {
      throw new Error("JWT missing required 'exp' claim");
    }

    return {
      userId,
      roles,
      sub: payload.sub,
      iss: payload.iss,
      exp: payload.exp,
    };
  }
}
