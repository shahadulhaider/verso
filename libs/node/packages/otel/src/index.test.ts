import { describe, it, expect, afterEach } from "vitest";
import { initTelemetry } from "./index.js";

describe("initTelemetry", () => {
  let telemetry: { shutdown: () => Promise<void> } | undefined;

  afterEach(async () => {
    if (telemetry) {
      await telemetry.shutdown();
      telemetry = undefined;
    }
  });

  it("returns an object with a shutdown function", () => {
    telemetry = initTelemetry("test-service");
    expect(telemetry).toBeDefined();
    expect(typeof telemetry.shutdown).toBe("function");
  });

  it("shutdown resolves without error", async () => {
    telemetry = initTelemetry("test-service");
    await expect(telemetry.shutdown()).resolves.toBeUndefined();
    telemetry = undefined; // already shut down
  });
});
