#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

case "${1:-}" in
  --full)
    exec bash "$SCRIPT_DIR/seed-openlibrary.sh"
    ;;
  --sample|"")
    exec bash "$SCRIPT_DIR/seed-sample.sh"
    ;;
  *)
    echo "Usage: $0 [--sample | --full]"
    echo ""
    echo "  --sample  (default) Seed 50 well-known books"
    echo "  --full    Import ≥10,000 works from Open Library dump"
    exit 1
    ;;
esac
