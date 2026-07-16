import { NodeSDK } from "@opentelemetry/sdk-node";
import { OTLPTraceExporter } from "@opentelemetry/exporter-trace-otlp-grpc";

const sdk = new NodeSDK({
  serviceName: "verso-feed-service",
  traceExporter: new OTLPTraceExporter(),
});

sdk.start();
