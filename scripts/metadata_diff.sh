#!/usr/bin/env bash
set -euo pipefail

CATEGORY="${1:-}"
CASE_ID="${2:-}"

if [[ -z "${CATEGORY}" || -z "${CASE_ID}" ]]; then
  echo "usage: metadata_diff.sh <files|postgres|api|mixed> <case_id>"
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

if [[ ! -f ./.env ]]; then
  echo ".env not found. Add it to the project root before running metadata-diff" >&2
  exit 1
fi

set -a
source ./.env
set +a

APP_COMPOSE="docker compose -f ./docker-compose.yml"
PG_DEMO_COMPOSE="docker compose -f ./demo/postgres/docker-compose.yml"
API_DEMO_COMPOSE="docker compose -f ./demo/api/docker-compose.yml"

cleanup_pg=0
cleanup_api=0
cleanup_files_case=""

cleanup() {
  if [[ -n "${cleanup_files_case}" ]]; then
    rm -rf "./demo/files/diff/${cleanup_files_case}/current"
  fi
  if [[ "${cleanup_pg}" -eq 1 ]]; then
    ${PG_DEMO_COMPOSE} down -v >/dev/null 2>&1 || true
  fi
  if [[ "${cleanup_api}" -eq 1 ]]; then
    ${API_DEMO_COMPOSE} down -v >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT INT TERM

ensure_app_bin() {
  if [[ -n "${CATALOG_BIN:-}" ]] && command -v "${CATALOG_BIN}" >/dev/null 2>&1; then
    APP_BIN="${CATALOG_BIN}"
    return 0
  fi

  if [[ -n "${CATALOG_BIN:-}" ]] && [[ -x "${CATALOG_BIN}" ]]; then
    APP_BIN="${CATALOG_BIN}"
    return 0
  fi

  if command -v catalog >/dev/null 2>&1; then
    APP_BIN="$(command -v catalog)"
    return 0
  fi

  if [[ -x "./bin/catalog" ]]; then
    APP_BIN="./bin/catalog"
    return 0
  fi

  echo "catalog binary not found. Install release binary or set CATALOG_BIN=./bin/catalog" >&2
  exit 1
}

wait_for_pg() {
  local dsn="$1"
  local retries=30
  local stable_hits=0
  while [[ "${retries}" -gt 0 ]]; do
    if PGPASSWORD="${CATALOG_DB_PASSWORD}" psql "${dsn}" -Atqc "select 1" >/dev/null 2>&1; then
      stable_hits=$((stable_hits + 1))
      if [[ "${stable_hits}" -ge 3 ]]; then
        return 0
      fi
    else
      stable_hits=0
    fi
    sleep 1
    retries=$((retries - 1))
  done

  echo "postgres demo source did not become ready in time for ${dsn}" >&2
  exit 1
}

wait_for_api() {
  local port="$1"
  local retries=30
  while [[ "${retries}" -gt 0 ]]; do
    if curl -fsS "http://localhost:${port}/openapi.json" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
    retries=$((retries - 1))
  done

  echo "api demo source on port ${port} did not become ready in time" >&2
  exit 1
}

run_and_capture_id() {
  local config_path="$1"
  local output
  output="$(${APP_BIN} run --config "${config_path}")"
  printf '%s\n' "${output}" >&2

  local run_id
  run_id="$(printf '%s\n' "${output}" | sed -n 's/.*run_id=\([0-9][0-9]*\).*/\1/p' | tail -n 1)"
  if [[ -z "${run_id}" ]]; then
    echo "failed to parse run_id from run output" >&2
    exit 1
  fi
  printf '%s' "${run_id}"
}

copy_files_state() {
  local case_id="$1"
  local mode="$2"
  local src_dir="./demo/files/diff/${case_id}/${mode}"
  local current_dir="./demo/files/diff/${case_id}/current"

  if [[ ! -d "${src_dir}" ]]; then
    echo "diff files source dir ${src_dir} not found" >&2
    exit 1
  fi

  rm -rf "${current_dir}"
  mkdir -p "${current_dir}"
  cp -R "${src_dir}/." "${current_dir}/"
  cleanup_files_case="${case_id}"
}

start_pg_diff() {
  local case_id="$1"
  local mode="$2"
  export DEMO_PG_CASE="${case_id}"
  export DEMO_PG_MODE="${mode}"
  export DEMO_PG_DSN="postgres://${CATALOG_DB_USER}:${CATALOG_DB_PASSWORD}@localhost:55433/source_case_${case_id}?sslmode=disable"
  ${PG_DEMO_COMPOSE} up -d source_pg >/dev/null
  cleanup_pg=1
  wait_for_pg "${DEMO_PG_DSN}"
}

restart_pg_diff() {
  ${PG_DEMO_COMPOSE} down -v >/dev/null
  cleanup_pg=0
}

start_api_diff() {
  local case_id="$1"
  local mode="$2"
  local service="demo_api_${case_id}"
  local port=$((8080 + case_id))
  export DEMO_API_MODE="${mode}"
  ${API_DEMO_COMPOSE} up -d "${service}" >/dev/null
  cleanup_api=1
  wait_for_api "${port}"
}

restart_api_diff() {
  ${API_DEMO_COMPOSE} down -v >/dev/null
  cleanup_api=0
}

case "${CATEGORY}" in
  files)
    CONFIG_PATH="./testcases/diff_files/${CASE_ID}.yaml"
    copy_files_state "${CASE_ID}" baseline
    ;;
  postgres)
    CONFIG_PATH="./testcases/diff_postgres/${CASE_ID}.yaml"
    start_pg_diff "${CASE_ID}" baseline
    ;;
  api)
    CONFIG_PATH="./testcases/diff_api/${CASE_ID}.yaml"
    start_api_diff "${CASE_ID}" baseline
    ;;
  mixed)
    CONFIG_PATH="./testcases/diff_mixed/${CASE_ID}.yaml"
    copy_files_state "${CASE_ID}" baseline
    start_pg_diff "${CASE_ID}" baseline
    start_api_diff "${CASE_ID}" baseline
    ;;
  *)
    echo "unsupported diff category: ${CATEGORY}" >&2
    exit 1
    ;;
esac

if [[ ! -f "${CONFIG_PATH}" ]]; then
  echo "diff config ${CONFIG_PATH} not found" >&2
  exit 1
fi

ensure_app_bin

baseline_run_id="$(run_and_capture_id "${CONFIG_PATH}")"
printf '\nBaseline run_id=%s\n' "${baseline_run_id}"

case "${CATEGORY}" in
  files)
    copy_files_state "${CASE_ID}" changed
    ;;
  postgres)
    restart_pg_diff
    start_pg_diff "${CASE_ID}" changed
    ;;
  api)
    restart_api_diff
    start_api_diff "${CASE_ID}" changed
    ;;
  mixed)
    restart_pg_diff
    restart_api_diff
    copy_files_state "${CASE_ID}" changed
    start_pg_diff "${CASE_ID}" changed
    start_api_diff "${CASE_ID}" changed
    ;;
esac

changed_run_id="$(run_and_capture_id "${CONFIG_PATH}")"
printf '\nChanged run_id=%s\n' "${changed_run_id}"

case "${CATEGORY}" in
  postgres)
    restart_pg_diff
    ;;
  api)
    restart_api_diff
    ;;
  mixed)
    restart_pg_diff
    restart_api_diff
    ;;
esac

if [[ "${CATEGORY}" == "files" || "${CATEGORY}" == "mixed" ]]; then
  rm -rf "./demo/files/diff/${CASE_ID}/current"
  cleanup_files_case=""
fi

printf '\n'
${APP_BIN} diff --config "${CONFIG_PATH}" --from-run-id "${baseline_run_id}" --to-run-id "${changed_run_id}"
