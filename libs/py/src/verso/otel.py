"""OpenTelemetry initialization for tracing."""

from __future__ import annotations

import os
from collections.abc import Callable

from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor


def init_telemetry(service_name: str) -> Callable[[], None]:
    """Initialize OpenTelemetry tracing and return a shutdown callable.

    Reads OTEL_EXPORTER_OTLP_ENDPOINT from environment.
    Defaults to http://localhost:4317 if unset.
    """
    endpoint = os.environ.get(
        "OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317"
    )

    resource = Resource.create({"service.name": service_name})
    provider = TracerProvider(resource=resource)

    exporter = OTLPSpanExporter(endpoint=endpoint, insecure=True)
    processor = BatchSpanProcessor(exporter)
    provider.add_span_processor(processor)

    trace.set_tracer_provider(provider)

    def shutdown() -> None:
        provider.shutdown()

    return shutdown
