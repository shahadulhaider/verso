"""Tests for verso.logger module."""

import json

import structlog

from verso.logger import create_logger, with_trace_id


def test_create_logger_returns_bound_logger():
    """create_logger returns a structlog bound logger."""
    logger = create_logger("test-service")
    # structlog.get_logger returns BoundLoggerLazyProxy which is usable as BoundLogger
    assert hasattr(logger, "info")
    assert hasattr(logger, "bind")


def test_create_logger_binds_service_name(capsys):
    """create_logger binds the service name to log output."""
    logger = create_logger("my-svc")
    logger.info("hello")
    captured = capsys.readouterr()
    parsed = json.loads(captured.out.strip())
    assert parsed["service"] == "my-svc"
    assert parsed["event"] == "hello"
    assert "timestamp" in parsed
    assert parsed.get("log_level") == "info" or parsed.get("level") == "info"


def test_with_trace_id_binds_trace(capsys):
    """with_trace_id adds trace_id to log output."""
    logger = create_logger("trace-svc")
    traced = with_trace_id(logger, "abc-trace-123")
    traced.info("traced event")
    captured = capsys.readouterr()
    parsed = json.loads(captured.out.strip())
    assert parsed["trace_id"] == "abc-trace-123"
