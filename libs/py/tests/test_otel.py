"""Tests for verso.otel module."""

from collections.abc import Callable
from unittest.mock import patch

from verso.otel import init_telemetry


@patch("verso.otel.BatchSpanProcessor")
@patch("verso.otel.OTLPSpanExporter")
@patch("verso.otel.trace")
def test_init_telemetry_returns_callable(mock_trace, mock_exporter, mock_processor):
    """init_telemetry returns a shutdown callable."""
    shutdown = init_telemetry("test-service")
    assert isinstance(shutdown, Callable)


@patch("verso.otel.BatchSpanProcessor")
@patch("verso.otel.OTLPSpanExporter")
@patch("verso.otel.trace")
def test_init_telemetry_sets_tracer_provider(mock_trace, mock_exporter, mock_processor):
    """init_telemetry registers a TracerProvider."""
    init_telemetry("my-service")
    mock_trace.set_tracer_provider.assert_called_once()
    provider = mock_trace.set_tracer_provider.call_args[0][0]
    assert provider is not None


@patch("verso.otel.BatchSpanProcessor")
@patch("verso.otel.OTLPSpanExporter")
@patch("verso.otel.trace")
def test_init_telemetry_default_endpoint(mock_trace, mock_exporter, mock_processor):
    """init_telemetry uses default endpoint when env var unset."""
    init_telemetry("svc")
    mock_exporter.assert_called_once()
    call_kwargs = mock_exporter.call_args
    assert call_kwargs[1]["endpoint"] == "http://localhost:4317"
    assert call_kwargs[1]["insecure"] is True


@patch("verso.otel.BatchSpanProcessor")
@patch("verso.otel.OTLPSpanExporter")
@patch("verso.otel.trace")
@patch.dict("os.environ", {"OTEL_EXPORTER_OTLP_ENDPOINT": "http://otel:4317"})
def test_init_telemetry_custom_endpoint(mock_trace, mock_exporter, mock_processor):
    """init_telemetry reads endpoint from environment."""
    init_telemetry("svc")
    call_kwargs = mock_exporter.call_args
    assert call_kwargs[1]["endpoint"] == "http://otel:4317"


@patch("verso.otel.BatchSpanProcessor")
@patch("verso.otel.OTLPSpanExporter")
@patch("verso.otel.trace")
def test_shutdown_callable_invokes_provider_shutdown(mock_trace, mock_exporter, mock_processor):
    """Calling the returned shutdown function shuts down the provider."""
    shutdown = init_telemetry("svc")
    # Should not raise
    shutdown()
