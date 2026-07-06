#!/usr/bin/env bash
set -euo pipefail

DB_NAME="rss_atlas"

echo "==> Creating database: $DB_NAME"
createdb "$DB_NAME" 2>/dev/null || echo "    (maybe already exists — skipping)"

echo "==> Applying schema..."
psql -d "$DB_NAME" -f "$(dirname "$0")/001_create_schema.sql"

echo "==> Done.  Database '$DB_NAME' is ready."
echo "    Connect:  psql $DB_NAME"