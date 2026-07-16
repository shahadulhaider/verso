#!/bin/bash
set -euo pipefail

# ┌─────────────────────────────────────────────────────────────────┐
# │ seed-openlibrary.sh — Import ≥10,000 works from Open Library   │
# │                                                                 │
# │ Prerequisites:                                                  │
# │   1. Download the OL works dump (JSON lines, ~5GB compressed):  │
# │      https://openlibrary.org/developers/dumps                   │
# │   2. Decompress: xz -dk ol_dump_works_latest.txt.xz            │
# │   3. Set DUMP_FILE env var to the path of the decompressed file │
# │                                                                 │
# │ Usage:                                                          │
# │   DUMP_FILE=./ol_dump_works_latest.txt ./seed-openlibrary.sh    │
# │                                                                 │
# │ Env vars:                                                       │
# │   DUMP_FILE    — path to decompressed OL works dump (required)  │
# │   BFF_URL      — BFF endpoint (default: http://localhost:8010)  │
# │   TARGET_COUNT — works to import (default: 10000)               │
# │   RATE_LIMIT   — requests per second (default: 10)              │
# │   SKIP_LINES   — lines to skip from start (default: 0)         │
# └─────────────────────────────────────────────────────────────────┘

BFF_URL="${BFF_URL:-http://localhost:8010}"
SEED_EMAIL="${SEED_EMAIL:-seed@verso.app}"
SEED_PASSWORD="${SEED_PASSWORD:-SeedUser1234!}"
TARGET_COUNT="${TARGET_COUNT:-10000}"
RATE_LIMIT="${RATE_LIMIT:-10}"
SKIP_LINES="${SKIP_LINES:-0}"
DUMP_FILE="${DUMP_FILE:-}"

if [ -z "$DUMP_FILE" ]; then
  echo "ERROR: DUMP_FILE is required."
  echo ""
  echo "Download the Open Library works dump:"
  echo "  curl -O https://openlibrary.org/data/ol_dump_works_latest.txt.xz"
  echo "  xz -dk ol_dump_works_latest.txt.xz"
  echo ""
  echo "Then run:"
  echo "  DUMP_FILE=./ol_dump_works_latest.txt $0"
  exit 1
fi

if [ ! -f "$DUMP_FILE" ]; then
  echo "ERROR: File not found: $DUMP_FILE"
  exit 1
fi

echo "=== Verso Open Library Seed ==="
echo "BFF:    $BFF_URL"
echo "Target: $TARGET_COUNT works"
echo "Rate:   $RATE_LIMIT req/s"
echo "Dump:   $DUMP_FILE"
echo ""

# ── Auth ────────────────────────────────────────────────────────
echo "--- Authenticating ---"
REGISTER_RESP=$(curl -sf -X POST "$BFF_URL/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$SEED_EMAIL\",\"password\":\"$SEED_PASSWORD\",\"displayName\":\"Seed User\"}" 2>/dev/null || true)

if [ -z "$REGISTER_RESP" ] || [ "$(echo "$REGISTER_RESP" | jq -r '.accessToken // empty' 2>/dev/null)" = "" ]; then
  REGISTER_RESP=$(curl -sf -X POST "$BFF_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$SEED_EMAIL\",\"password\":\"$SEED_PASSWORD\"}")
fi

TOKEN=$(echo "$REGISTER_RESP" | jq -r '.accessToken')
if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo "ERROR: Could not obtain auth token"
  exit 1
fi
echo "  Token obtained"
echo ""

# ── Rate limiter ────────────────────────────────────────────────
INTERVAL=$(awk "BEGIN {printf \"%.4f\", 1.0/$RATE_LIMIT}")
LAST_REQUEST=0

rate_limit() {
  local NOW
  NOW=$(python3 -c "import time; print(time.time())" 2>/dev/null || date +%s)
  local ELAPSED
  ELAPSED=$(awk "BEGIN {printf \"%.4f\", $NOW - $LAST_REQUEST}")
  local SLEEP_FOR
  SLEEP_FOR=$(awk "BEGIN {s=$INTERVAL-$ELAPSED; if(s>0) printf \"%.4f\", s; else print 0}")
  if (( $(awk "BEGIN {print ($SLEEP_FOR > 0)}") )); then
    sleep "$SLEEP_FOR"
  fi
  LAST_REQUEST=$(python3 -c "import time; print(time.time())" 2>/dev/null || date +%s)
}

# ── Parse and import ────────────────────────────────────────────
# OL dump format (tab-separated):
#   type  key  revision  last_modified  json_blob
# The json_blob contains: title, description (string or {value:"..."}),
# subjects (array), first_publish_date, authors [{author:{key:"/authors/..."}}]

SUCCESS=0
FAILED=0
SKIPPED=0
LINE_NUM=0

echo "--- Importing works ---"

while IFS=$'\t' read -r TYPE KEY REVISION LAST_MOD JSON_BLOB; do
  LINE_NUM=$((LINE_NUM + 1))

  if [ "$LINE_NUM" -le "$SKIP_LINES" ]; then
    continue
  fi

  if [ "$SUCCESS" -ge "$TARGET_COUNT" ]; then
    break
  fi

  TITLE=$(echo "$JSON_BLOB" | jq -r '.title // empty' 2>/dev/null)
  if [ -z "$TITLE" ]; then
    SKIPPED=$((SKIPPED + 1))
    continue
  fi

  # Extract description (can be string or {type, value} object)
  DESC=$(echo "$JSON_BLOB" | jq -r '
    if .description | type == "string" then .description
    elif .description.value then .description.value
    else empty
    end' 2>/dev/null | head -c 500)

  # Extract first_publish_date → year
  PUB_YEAR=$(echo "$JSON_BLOB" | jq -r '.first_publish_date // empty' 2>/dev/null | grep -oE '[0-9]{4}' | head -1)

  # Build payload
  if [ -n "$PUB_YEAR" ] && [ "$PUB_YEAR" -gt 0 ] 2>/dev/null; then
    PAYLOAD=$(jq -n \
      --arg title "$TITLE" \
      --arg desc "${DESC:-}" \
      --argjson year "$PUB_YEAR" \
      '{title: $title, description: (if $desc == "" then null else $desc end), originalLanguage: "en", originalPublicationYear: $year}')
  else
    PAYLOAD=$(jq -n \
      --arg title "$TITLE" \
      --arg desc "${DESC:-}" \
      '{title: $title, description: (if $desc == "" then null else $desc end), originalLanguage: "en"}')
  fi

  rate_limit

  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "$BFF_URL/api/v1/books" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "$PAYLOAD")

  if [ "$HTTP_CODE" = "201" ]; then
    SUCCESS=$((SUCCESS + 1))
    if [ $((SUCCESS % 100)) -eq 0 ]; then
      echo "  Progress: $SUCCESS / $TARGET_COUNT (line $LINE_NUM, $FAILED failed, $SKIPPED skipped)"
    fi
  elif [ "$HTTP_CODE" = "409" ]; then
    SKIPPED=$((SKIPPED + 1))
  else
    FAILED=$((FAILED + 1))
    if [ $((FAILED % 50)) -eq 0 ]; then
      echo "  Warning: $FAILED failures so far (last HTTP $HTTP_CODE for '$TITLE')"
    fi
  fi

done < "$DUMP_FILE"

echo ""
echo "--- Results ---"
echo "  Created:  $SUCCESS"
echo "  Skipped:  $SKIPPED (no title or duplicate)"
echo "  Failed:   $FAILED"
echo "  Lines:    $LINE_NUM"
echo ""

if [ "$SUCCESS" -ge "$TARGET_COUNT" ]; then
  echo "=== Open Library seed complete: $SUCCESS works imported ==="
else
  echo "=== Partial import: $SUCCESS / $TARGET_COUNT works (dump may have fewer qualifying entries) ==="
  echo "  Try increasing SKIP_LINES=$LINE_NUM and running again with a different dump section."
fi
