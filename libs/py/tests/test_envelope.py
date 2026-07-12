"""Tests for verso.envelope module."""

from verso.envelope import (
    EventEnvelope,
    create_envelope,
    marshal_envelope,
    unmarshal_envelope,
)


def test_create_envelope_has_ulid():
    """create_envelope generates a valid ULID event_id."""
    envelope = create_envelope(
        event_type="order.created",
        producer="order-service",
        partition_key="order-123",
        payload=b'{"amount": 100}',
    )
    assert len(envelope.event_id) == 26  # ULID string length
    assert envelope.type == "order.created"
    assert envelope.producer == "order-service"
    assert envelope.partition_key == "order-123"
    assert envelope.schema_version == 1


def test_create_envelope_iso_timestamp():
    """create_envelope produces an ISO UTC timestamp."""
    envelope = create_envelope(
        event_type="test.event",
        producer="test",
        partition_key="key",
        payload=b"{}",
    )
    assert "T" in envelope.occurred_at
    assert envelope.occurred_at.endswith("+00:00")


def test_marshal_unmarshal_roundtrip():
    """marshal/unmarshal preserves all fields."""
    original = create_envelope(
        event_type="user.registered",
        producer="auth-service",
        partition_key="user-456",
        payload=b'{"email": "test@example.com"}',
        trace_id="abc-123",
        schema_version=2,
    )

    data = marshal_envelope(original)
    restored = unmarshal_envelope(data)

    assert restored.event_id == original.event_id
    assert restored.type == original.type
    assert restored.occurred_at == original.occurred_at
    assert restored.producer == original.producer
    assert restored.trace_id == original.trace_id
    assert restored.partition_key == original.partition_key
    assert restored.payload == original.payload
    assert restored.schema_version == original.schema_version
