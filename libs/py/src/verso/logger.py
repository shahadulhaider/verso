"""Structured logging via structlog."""

from __future__ import annotations

import structlog


def create_logger(service_name: str) -> structlog.stdlib.BoundLogger:
    """Create a structured JSON logger with the service name bound.

    Configures structlog for JSON output with timestamps and log levels.
    """
    structlog.configure(
        processors=[
            structlog.contextvars.merge_contextvars,
            structlog.stdlib.add_log_level,
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.processors.StackInfoRenderer(),
            structlog.processors.format_exc_info,
            structlog.processors.JSONRenderer(),
        ],
        wrapper_class=structlog.stdlib.BoundLogger,
        context_class=dict,
        logger_factory=structlog.PrintLoggerFactory(),
        cache_logger_on_first_use=True,
    )

    logger = structlog.get_logger(service=service_name)
    return logger


def with_trace_id(
    logger: structlog.stdlib.BoundLogger, trace_id: str
) -> structlog.stdlib.BoundLogger:
    """Return a new logger with trace_id bound."""
    return logger.bind(trace_id=trace_id)
