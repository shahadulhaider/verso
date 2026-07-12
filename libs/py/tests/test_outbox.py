"""Tests for verso.outbox module."""

from unittest.mock import AsyncMock, MagicMock

import pytest

from verso.envelope import create_envelope
from verso.outbox import CREATE_TABLE_SQL, insert_event, mark_delivered, pending_events


def test_create_table_sql_exists():
    """CREATE_TABLE_SQL is a non-empty string with expected DDL."""
    assert isinstance(CREATE_TABLE_SQL, str)
    assert "outbox_events" in CREATE_TABLE_SQL
    assert "CREATE TABLE" in CREATE_TABLE_SQL
    assert "aggregate_type" in CREATE_TABLE_SQL
    assert "delivered_at" in CREATE_TABLE_SQL


def test_create_table_sql_has_index():
    """CREATE_TABLE_SQL includes pending events index."""
    assert "idx_outbox_pending" in CREATE_TABLE_SQL
    assert "WHERE delivered_at IS NULL" in CREATE_TABLE_SQL


@pytest.mark.asyncio
async def test_insert_event_executes_sql():
    """insert_event calls conn.execute with correct parameters."""
    conn = AsyncMock()
    envelope = create_envelope(
        event_type="order.created",
        producer="order-svc",
        partition_key="order-1",
        payload=b'{"id": 1}',
    )

    await insert_event(conn, "Order", "order-1", envelope)

    conn.execute.assert_called_once()
    call_args = conn.execute.call_args[0]
    assert "INSERT INTO outbox_events" in call_args[0]
    assert call_args[1] == envelope.event_id
    assert call_args[2] == "Order"
    assert call_args[3] == "order-1"
    assert call_args[4] == "order.created"


@pytest.mark.asyncio
async def test_mark_delivered_executes_update():
    """mark_delivered updates the delivered_at field."""
    pool = AsyncMock()

    await mark_delivered(pool, "event-123")

    pool.execute.assert_called_once()
    call_args = pool.execute.call_args[0]
    assert "UPDATE outbox_events" in call_args[0]
    assert "delivered_at" in call_args[0]
    assert call_args[1] == "event-123"


@pytest.mark.asyncio
async def test_pending_events_fetches_undelivered():
    """pending_events queries for undelivered events with limit."""
    from verso.envelope import marshal_envelope

    envelope = create_envelope(
        event_type="test.event",
        producer="test",
        partition_key="key-1",
        payload=b'{"x": 1}',
    )
    payload_bytes = marshal_envelope(envelope)

    pool = AsyncMock()
    pool.fetch.return_value = [{"payload": payload_bytes}]

    results = await pending_events(pool, limit=50)

    assert len(results) == 1
    assert results[0].type == "test.event"
    pool.fetch.assert_called_once()
    call_args = pool.fetch.call_args[0]
    assert "WHERE delivered_at IS NULL" in call_args[0]
    assert call_args[1] == 50
