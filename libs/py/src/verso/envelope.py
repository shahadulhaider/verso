"""Event envelope for domain events."""

from __future__ import annotations

import json
from dataclasses import dataclass
from datetime import datetime, timezone

from ulid import ULID


@dataclass(frozen=True, slots=True)
class EventEnvelope:
    """Standardized event wrapper for all domain events."""

    event_id: str
    type: str
    occurred_at: str
    producer: str
    trace_id: str
    partition_key: str
    payload: bytes
    schema_version: int = 1


def create_envelope(
    event_type: str,
    producer: str,
    partition_key: str,
    payload: bytes,
    *,
    trace_id: str = "",
    schema_version: int = 1,
) -> EventEnvelope:
    """Create a new EventEnvelope with generated ULID and UTC timestamp."""
    return EventEnvelope(
        event_id=str(ULID()),
        type=event_type,
        occurred_at=datetime.now(timezone.utc).isoformat(),
        producer=producer,
        trace_id=trace_id,
        partition_key=partition_key,
        payload=payload,
        schema_version=schema_version,
    )


def marshal_envelope(envelope: EventEnvelope) -> bytes:
    """Serialize an EventEnvelope to JSON bytes."""
    data = {
        "event_id": envelope.event_id,
        "type": envelope.type,
        "occurred_at": envelope.occurred_at,
        "producer": envelope.producer,
        "trace_id": envelope.trace_id,
        "partition_key": envelope.partition_key,
        "payload": envelope.payload.decode("utf-8"),
        "schema_version": envelope.schema_version,
    }
    return json.dumps(data).encode("utf-8")


def unmarshal_envelope(data: bytes) -> EventEnvelope:
    """Deserialize JSON bytes into an EventEnvelope."""
    parsed = json.loads(data)
    return EventEnvelope(
        event_id=parsed["event_id"],
        type=parsed["type"],
        occurred_at=parsed["occurred_at"],
        producer=parsed["producer"],
        trace_id=parsed["trace_id"],
        partition_key=parsed["partition_key"],
        payload=parsed["payload"].encode("utf-8"),
        schema_version=parsed["schema_version"],
    )
