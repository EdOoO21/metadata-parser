#!/bin/sh
set -eu

echo "Preparing demo postgres source databases..."

escaped_password="$(printf "%s" "${POSTGRES_PASSWORD}" | sed "s/'/''/g")"

echo "Synchronizing password for role ${POSTGRES_USER}"
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<SQL
ALTER ROLE "${POSTGRES_USER}" WITH PASSWORD '${escaped_password}';
SQL

if [ -f "${PGDATA}/pg_hba.conf" ]; then
  echo "Enabling trust auth for external demo-source connections"
  sed -i '1ihost all all all trust' "${PGDATA}/pg_hba.conf"
fi

if [ -n "${DEMO_PG_CASE:-}" ] && [ -n "${DEMO_PG_MODE:-}" ]; then
  case_id="${DEMO_PG_CASE}"
  mode="${DEMO_PG_MODE}"
  db_name="source_case_${case_id}"
  sql_file="/demo-postgres/diff/${case_id}/${mode}/init.sql"

  if [ ! -f "$sql_file" ]; then
    echo "Diff fixture $sql_file not found"
    exit 1
  fi

  echo "Creating database $db_name for diff mode $mode"
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<SQL
SELECT 'CREATE DATABASE ${db_name}'
WHERE NOT EXISTS (
  SELECT 1
  FROM pg_database
  WHERE datname = '${db_name}'
)\gexec
SQL

  echo "Applying diff fixture $sql_file to $db_name"
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$db_name" -f "$sql_file"

  echo "Demo postgres diff database is ready."
  exit 0
fi

for dir in /demo-postgres/test_*; do
  [ -d "$dir" ] || continue

  case_name="$(basename "$dir")"
  case_id="${case_name#test_}"
  db_name="source_case_${case_id}"
  sql_file="$dir/init.sql"

  if [ ! -f "$sql_file" ]; then
    echo "Skipping $case_name: init.sql not found"
    continue
  fi

  echo "Creating database $db_name"
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<SQL
SELECT 'CREATE DATABASE ${db_name}'
WHERE NOT EXISTS (
  SELECT 1
  FROM pg_database
  WHERE datname = '${db_name}'
)\gexec
SQL

  echo "Applying fixture $sql_file to $db_name"
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$db_name" -f "$sql_file"
done

echo "Demo postgres source databases are ready."
