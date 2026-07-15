#!/bin/bash
set -euo pipefail

BFF_URL="${BFF_URL:-http://localhost:8010}"

echo "=== Verso Seed Script ==="
echo "BFF URL: $BFF_URL"
echo ""

# Register a test user
echo "--- Registering user ---"
REGISTER_RESP=$(curl -sf -X POST "$BFF_URL/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"seed@verso.app","password":"SeedPass123!","displayName":"Seed User"}')
TOKEN=$(echo "$REGISTER_RESP" | jq -r '.accessToken')
echo "Token: ${TOKEN:0:30}..."
echo ""

# Create sample works
declare -a BOOKS=(
  '{"title":"Dune","description":"A science fiction novel set in the far future.","originalLanguage":"en","originalPublicationYear":1965}'
  '{"title":"Neuromancer","description":"A pioneering cyberpunk novel by William Gibson.","originalLanguage":"en","originalPublicationYear":1984}'
  '{"title":"Snow Crash","description":"A sci-fi novel exploring linguistics, computer science, and pizza delivery.","originalLanguage":"en","originalPublicationYear":1992}'
  '{"title":"The Great Gatsby","description":"A novel of the Jazz Age by F. Scott Fitzgerald.","originalLanguage":"en","originalPublicationYear":1925}'
  '{"title":"1984","description":"A dystopian social science fiction novel by George Orwell.","originalLanguage":"en","originalPublicationYear":1949}'
)

CREATED_IDS=()
echo "--- Creating works ---"
for book in "${BOOKS[@]}"; do
  TITLE=$(echo "$book" | jq -r '.title')
  RESP=$(curl -sf -X POST "$BFF_URL/api/v1/books" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "$book")
  WORK_ID=$(echo "$RESP" | jq -r '.id')
  CREATED_IDS+=("$WORK_ID")
  echo "  Created: $TITLE (id: $WORK_ID)"
done
echo ""
echo "=== Seeded ${#BOOKS[@]} books ==="
echo ""

# Wait for async event pipeline (outbox → Debezium → Redpanda → search indexer → OpenSearch)
WAIT_SECS="${WAIT_SECS:-15}"
echo "--- Waiting ${WAIT_SECS}s for event pipeline ---"
sleep "$WAIT_SECS"

# Verify search
echo "--- Verifying search ---"
SEARCH_RESP=$(curl -sf "$BFF_URL/api/v1/search?q=dune&type=work" || echo '{"results":[]}')
HITS=$(echo "$SEARCH_RESP" | jq '.results | length')
echo "Search 'dune': $HITS result(s)"

if [ "$HITS" -gt 0 ]; then
  echo ""
  echo "=== SEED COMPLETE — Full pipeline verified ==="
else
  echo ""
  echo "WARNING: Search returned 0 results. Pipeline may still be processing."
  echo "Try: curl $BFF_URL/api/v1/search?q=dune&type=work"
fi
