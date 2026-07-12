import pino, { type Logger } from "pino";

/**
 * Create a structured JSON logger for a service.
 *
 * @param serviceName - Logical service name added to every log line
 * @returns A pino Logger instance
 */
export function createLogger(serviceName: string): Logger {
  return pino({
    name: serviceName,
    base: { service_name: serviceName },
    timestamp: pino.stdTimeFunctions.isoTime,
  });
}

/**
 * Create a child logger with a traceId pinned to every log line.
 *
 * @param logger - Parent pino logger
 * @param traceId - OpenTelemetry trace ID
 * @returns A child logger with traceId in every message
 */
export function withTraceId(logger: Logger, traceId: string): Logger {
  return logger.child({ traceId });
}

export type { Logger };
