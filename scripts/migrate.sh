#!/bin/sh
set -eu

echo "Checking DB connection..."
psql \
  -h "$CATALOG_DB_HOST" \
  -p "$CATALOG_DB_PORT" \
  -U "$CATALOG_DB_USER" \
  -d "$CATALOG_DB_NAME" \
  -v ON_ERROR_STOP=1 \
  -c "select 1;"

echo "Applying migrations in lexicographic order..."
find /migrations -maxdepth 1 -type f -name '*.up.sql' | LC_ALL=C sort | while read -r file; do
  echo "Running $file"
  psql \
    -h "$CATALOG_DB_HOST" \
    -p "$CATALOG_DB_PORT" \
    -U "$CATALOG_DB_USER" \
    -d "$CATALOG_DB_NAME" \
    -v ON_ERROR_STOP=1 \
    -f "$file"
done

echo "Migrations applied successfully."