import { describe, it, expect, beforeAll, afterAll } from "vitest";
import * as jose from "jose";
import { JWKSValidator } from "./index.js";
import { createServer, type Server } from "node:http";

/**
 * Spin up a tiny HTTP server that serves a JWKS endpoint,
 * sign a JWT with the corresponding private key, then validate it.
 */
describe("JWKSValidator", () => {
  let server: Server;
  let port: number;
  let privateKey: CryptoKey;
  let jwksJson: string;

  beforeAll(async () => {
    // Generate an RSA key pair
    const { publicKey, privateKey: pk } = await jose.generateKeyPair("RS256", {
      extractable: true,
    });
    privateKey = pk as CryptoKey;

    // Export public key as JWK
    const publicJwk = await jose.exportJWK(publicKey);
    publicJwk.kid = "test-key-1";
    publicJwk.alg = "RS256";
    publicJwk.use = "sig";
    jwksJson = JSON.stringify({ keys: [publicJwk] });

    // Start JWKS server
    server = createServer((req, res) => {
      res.writeHead(200, { "content-type": "application/json" });
      res.end(jwksJson);
    });

    await new Promise<void>((resolve) => {
      server.listen(0, "127.0.0.1", () => resolve());
    });
    const addr = server.address();
    port = typeof addr === "object" && addr !== null ? addr.port : 0;
  });

  afterAll(async () => {
    await new Promise<void>((resolve) => server.close(() => resolve()));
  });

  it("validates a properly signed JWT", async () => {
    const token = await new jose.SignJWT({
      userId: "user-42",
      roles: ["reader", "admin"],
    })
      .setProtectedHeader({ alg: "RS256", kid: "test-key-1" })
      .setSubject("user-42")
      .setIssuer("https://auth.verso.dev")
      .setExpirationTime("1h")
      .sign(privateKey);

    const validator = new JWKSValidator(
      `http://127.0.0.1:${port}/.well-known/jwks.json`,
    );
    const claims = await validator.validate(token);

    expect(claims.userId).toBe("user-42");
    expect(claims.sub).toBe("user-42");
    expect(claims.iss).toBe("https://auth.verso.dev");
    expect(claims.roles).toEqual(["reader", "admin"]);
    expect(typeof claims.exp).toBe("number");
    expect(claims.exp).toBeGreaterThan(Math.floor(Date.now() / 1000));
  });

  it("rejects an expired JWT", async () => {
    const token = await new jose.SignJWT({ userId: "user-1" })
      .setProtectedHeader({ alg: "RS256", kid: "test-key-1" })
      .setSubject("user-1")
      .setIssuer("https://auth.verso.dev")
      .setExpirationTime("-1h") // already expired
      .sign(privateKey);

    const validator = new JWKSValidator(
      `http://127.0.0.1:${port}/.well-known/jwks.json`,
    );

    await expect(validator.validate(token)).rejects.toThrow();
  });

  it("rejects a JWT signed with a different key", async () => {
    const { privateKey: otherKey } = await jose.generateKeyPair("RS256");
    const token = await new jose.SignJWT({ userId: "user-1" })
      .setProtectedHeader({ alg: "RS256", kid: "test-key-1" })
      .setSubject("user-1")
      .setIssuer("https://auth.verso.dev")
      .setExpirationTime("1h")
      .sign(otherKey);

    const validator = new JWKSValidator(
      `http://127.0.0.1:${port}/.well-known/jwks.json`,
    );

    await expect(validator.validate(token)).rejects.toThrow();
  });
});
