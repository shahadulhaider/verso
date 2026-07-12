import { NodeSDK } from "@opentelemetry/sdk-node";
import { OTLPTraceExporter } from "@opentelemetry/exporter-trace-otlp-grpc";
import { Resource } from "@opentelemetry/resources";
import { ATTR_SERVICE_NAME } from "@opentelemetry/semantic-conventions";

export interface Telemetry {
  shutdown: () => Promise<void>;
}

/**
 * Initialize OpenTelemetry tracing for a Node.js service.
 * Reads OTEL_EXPORTER_OTLP_ENDPOINT from environment.
 *
 * @param serviceName - Logical service name (e.g. "realtime-svc")
 * @returns Object with shutdown() to flush and close the SDK
 */
export function initTelemetry(serviceName: string): Telemetry {
  const endpoint = process.env.OTEL_EXPORTER_OTLP_ENDPOINT ?? "http://localhost:4317";

  const traceExporter = new OTLPTraceExporter({ url: endpoint });

  const sdk = new NodeSDK({
    resource: new Resource({
      [ATTR_SERVICE_NAME]: serviceName,
    }),
    traceExporter,
  });

  sdk.start();

  return {
    shutdown: async () => {
      await sdk.shutdown();
    },
  };
}
