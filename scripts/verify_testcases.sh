#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

if [[ ! -f ./.env ]]; then
  echo ".env not found. Add it to the project root before running verify_testcases.sh" >&2
  exit 1
fi

set -a
source ./.env
set +a

LOG_DIR="${ROOT_DIR}/tmp/e2e-verify"
SUMMARY_FILE="${LOG_DIR}/summary.txt"
LOCK_DIR="${LOG_DIR}/.lock"

mkdir -p "${LOG_DIR}"
: > "${SUMMARY_FILE}"

cleanup_app_db() {
  make app-down-v >/dev/null 2>&1 || true
}

cleanup_demo_sources() {
  docker compose -f ./demo/postgres/docker-compose.yml down -v >/dev/null 2>&1 || true
  docker compose -f ./demo/api/docker-compose.yml down -v >/dev/null 2>&1 || true
}

cleanup_all() {
  cleanup_demo_sources
  cleanup_app_db
}

if ! mkdir "${LOCK_DIR}" 2>/dev/null; then
  echo "verify_testcases.sh is already running: ${LOCK_DIR}" >&2
  exit 1
fi

trap 'cleanup_all; rmdir "${LOCK_DIR}" >/dev/null 2>&1 || true' EXIT INT TERM

declare -A EXPECTED_EXIT
declare -A EXPECTED_STATUS

EXPECTED_EXIT["files/10"]=2
EXPECTED_EXIT["files/11"]=2
EXPECTED_EXIT["files/17"]=2
EXPECTED_EXIT["files/19"]=2
EXPECTED_EXIT["mixed/7"]=2

EXPECTED_STATUS["files/10"]="failed"
EXPECTED_STATUS["files/11"]="failed"
EXPECTED_STATUS["files/17"]="partial"
EXPECTED_STATUS["files/19"]="partial"
EXPECTED_STATUS["mixed/7"]="partial"

sql() {
  psql "${CATALOG_DSN}" -v ON_ERROR_STOP=1 -Atqc "$1" < /dev/null
}

assert_clean_catalog() {
  local failures_ref="$1"
  local existing_runs
  local existing_run_sources
  local existing_datasets

  existing_runs="$(sql 'select count(*) from runs')"
  existing_run_sources="$(sql 'select count(*) from run_sources')"
  existing_datasets="$(sql 'select count(*) from datasets')"

  check_zero "pre-run runs" "${existing_runs}" "${failures_ref}"
  check_zero "pre-run run_sources" "${existing_run_sources}" "${failures_ref}"
  check_zero "pre-run datasets" "${existing_datasets}" "${failures_ref}"
}

source_count() {
  grep -c '^  - name:' "$1"
}

metadata_command() {
  local category="$1"
  local case_id="$2"

  case "${category}" in
    diff_files)
      printf 'make metadata-diff CATEGORY=files CASE=%s' "${case_id}"
      ;;
    diff_postgres)
      printf 'make metadata-diff CATEGORY=postgres CASE=%s' "${case_id}"
      ;;
    diff_api)
      printf 'make metadata-diff CATEGORY=api CASE=%s' "${case_id}"
      ;;
    diff_mixed)
      printf 'make metadata-diff CATEGORY=mixed CASE=%s' "${case_id}"
      ;;
    *)
      printf 'make metadata CATEGORY=%s CASE=%s' "${category}" "${case_id}"
      ;;
  esac
}

config_path() {
  local category="$1"
  local case_id="$2"
  printf './testcases/%s/%s.yaml' "${category}" "${case_id}"
}

expected_runs() {
  local category="$1"
  case "${category}" in
    diff_files|diff_postgres|diff_api|diff_mixed)
      printf '2'
      ;;
    *)
      printf '1'
      ;;
  esac
}

expected_exit_code() {
  local category="$1"
  local case_id="$2"
  local key="${category}/${case_id}"
  printf '%s' "${EXPECTED_EXIT[${key}]:-0}"
}

expected_run_status() {
  local category="$1"
  local case_id="$2"
  local key="${category}/${case_id}"
  printf '%s' "${EXPECTED_STATUS[${key}]:-success}"
}

check_eq() {
  local label="$1"
  local actual="$2"
  local expected="$3"
  local failures_ref="$4"

  if [[ "${actual}" != "${expected}" ]]; then
    printf -v "${failures_ref}" '%s\n- %s: expected=%s actual=%s' "${!failures_ref}" "${label}" "${expected}" "${actual}"
  fi
}

check_zero() {
  local label="$1"
  local actual="$2"
  local failures_ref="$3"

  if [[ "${actual}" != "0" ]]; then
    printf -v "${failures_ref}" '%s\n- %s: expected=0 actual=%s' "${!failures_ref}" "${label}" "${actual}"
  fi
}

verify_case() {
  local category="$1"
  local case_id="$2"
  local config
  local case_log
  local cmd
  local expected_sources
  local expected_run_count
  local expected_exit
  local expected_status
  local cmd_exit=0
  local failures=""

  config="$(config_path "${category}" "${case_id}")"
  case_log="${LOG_DIR}/${category}_${case_id}.log"
  cmd="$(metadata_command "${category}" "${case_id}")"
  expected_sources="$(source_count "${config}")"
  expected_run_count="$(expected_runs "${category}")"
  expected_exit="$(expected_exit_code "${category}" "${case_id}")"
  expected_status="$(expected_run_status "${category}" "${case_id}")"

  printf '==> %s/%s\n' "${category}" "${case_id}" | tee -a "${SUMMARY_FILE}"
  printf 'config=%s\ncommand=%s\n' "${config}" "${cmd}" > "${case_log}"

  cleanup_all
  make app-up >> "${case_log}" 2>&1 < /dev/null
  assert_clean_catalog failures

  set +e
  bash -lc "${cmd}" >> "${case_log}" 2>&1 < /dev/null
  cmd_exit=$?
  set -e

  local run_count
  local run_statuses
  local run_source_count
  local run_source_running
  local run_running
  local datasets_count
  local columns_count
  local stats_count
  local top_values_count
  local dataset_without_columns
  local success_sources_without_datasets
  local invalid_dataset_kinds
  local invalid_profile_statuses
  local invalid_column_types
  local invalid_row_counts
  local profiled_missing_stats
  local unprofiled_with_stats
  local stat_totals_mismatch
  local invalid_distinct_counts
  local invalid_top_values
  local duplicate_top_ranks
  local top_values_exceed_non_null

  run_count="$(sql 'select count(*) from runs')"
  run_statuses="$(sql "select coalesce(string_agg(status, ',' order by id), '') from runs")"
  run_source_count="$(sql 'select count(*) from run_sources')"
  run_source_running="$(sql "select count(*) from run_sources where status = 'running'")"
  run_running="$(sql "select count(*) from runs where status = 'running'")"
  datasets_count="$(sql 'select count(*) from datasets')"
  columns_count="$(sql 'select count(*) from columns')"
  stats_count="$(sql 'select count(*) from column_stats')"
  top_values_count="$(sql 'select count(*) from column_top_values')"
  dataset_without_columns="$(sql "with per_dataset as (select d.id, count(c.id) as cnt from datasets d left join columns c on c.dataset_id = d.id group by d.id) select count(*) from per_dataset where cnt = 0")"
  success_sources_without_datasets="$(sql "with per_source as (select rs.id, rs.status, count(d.id) as cnt from run_sources rs left join datasets d on d.run_source_id = rs.id group by rs.id, rs.status) select count(*) from per_source where status = 'success' and cnt = 0")"
  invalid_dataset_kinds="$(sql "select count(*) from datasets where kind not in ('table', 'view', 'file', 'endpoint')")"
  invalid_profile_statuses="$(sql "select count(*) from datasets where profile_status not in ('profiled', 'discovered_only', 'skipped_requires_params', 'failed')")"
  invalid_column_types="$(sql "select count(*) from columns where normalized_type not in ('STRING', 'NUMBER', 'BOOLEAN', 'TIMESTAMP', 'ARRAY')")"
  invalid_row_counts="$(sql 'select count(*) from datasets where row_count is not null and row_count < 0')"
  profiled_missing_stats="$(sql "select count(*) from columns c join datasets d on d.id = c.dataset_id left join column_stats s on s.column_id = c.id where d.profile_status = 'profiled' and s.id is null")"
  unprofiled_with_stats="$(sql "select count(*) from columns c join datasets d on d.id = c.dataset_id join column_stats s on s.column_id = c.id where d.profile_status <> 'profiled'")"
  stat_totals_mismatch="$(sql "select count(*) from column_stats s join columns c on c.id = s.column_id join datasets d on d.id = c.dataset_id where d.profile_status = 'profiled' and d.row_count is not null and (s.non_null_count + s.null_count) <> d.row_count")"
  invalid_distinct_counts="$(sql 'select count(*) from column_stats where distinct_count > non_null_count')"
  invalid_top_values="$(sql 'select count(*) from column_top_values where rank <= 0 or occurrence_count <= 0')"
  duplicate_top_ranks="$(sql 'select count(*) from (select column_stat_id, rank, count(*) as cnt from column_top_values group by column_stat_id, rank having count(*) > 1) t')"
  top_values_exceed_non_null="$(sql 'select count(*) from (select s.id, s.non_null_count, coalesce(sum(tv.occurrence_count), 0) as top_total from column_stats s left join column_top_values tv on tv.column_stat_id = s.id group by s.id, s.non_null_count having coalesce(sum(tv.occurrence_count), 0) > s.non_null_count) t')"

  check_eq "command exit code" "${cmd_exit}" "${expected_exit}" failures
  check_eq "run count" "${run_count}" "${expected_run_count}" failures
  check_eq "run_source count" "${run_source_count}" "$(( expected_run_count * expected_sources ))" failures
  check_zero "runs still running" "${run_running}" failures
  check_zero "run_sources still running" "${run_source_running}" failures
  check_zero "datasets without columns" "${dataset_without_columns}" failures
  check_zero "successful run_sources without datasets" "${success_sources_without_datasets}" failures
  check_zero "invalid dataset kinds" "${invalid_dataset_kinds}" failures
  check_zero "invalid profile statuses" "${invalid_profile_statuses}" failures
  check_zero "invalid normalized types" "${invalid_column_types}" failures
  check_zero "negative row_count" "${invalid_row_counts}" failures
  check_zero "profiled columns without stats" "${profiled_missing_stats}" failures
  check_zero "stats on unprofiled datasets" "${unprofiled_with_stats}" failures
  check_zero "column stat totals mismatch row_count" "${stat_totals_mismatch}" failures
  check_zero "distinct_count greater than non_null_count" "${invalid_distinct_counts}" failures
  check_zero "invalid top values" "${invalid_top_values}" failures
  check_zero "duplicate top-value ranks" "${duplicate_top_ranks}" failures
  check_zero "top values exceed non_null_count" "${top_values_exceed_non_null}" failures

  case "${expected_status}" in
    success)
      if [[ "${expected_run_count}" == "1" && "${run_statuses}" != "success" ]]; then
        printf -v failures '%s\n- run statuses: expected=success actual=%s' "${failures}" "${run_statuses}"
      fi
      if [[ "${expected_run_count}" == "2" && "${run_statuses}" != "success,success" ]]; then
        printf -v failures '%s\n- run statuses: expected=success,success actual=%s' "${failures}" "${run_statuses}"
      fi
      ;;
    failed)
      if [[ "${run_statuses}" != "failed" ]]; then
        printf -v failures '%s\n- run statuses: expected=failed actual=%s' "${failures}" "${run_statuses}"
      fi
      ;;
    partial)
      if [[ "${run_statuses}" != "partial" ]]; then
        printf -v failures '%s\n- run statuses: expected=partial actual=%s' "${failures}" "${run_statuses}"
      fi
      ;;
  esac

  if [[ "${expected_status}" == "success" && "${datasets_count}" == "0" ]]; then
    printf -v failures '%s\n- datasets count: expected more than 0 for successful case' "${failures}"
  fi

  if [[ "${expected_status}" == "failed" && "${datasets_count}" != "0" ]]; then
    printf -v failures '%s\n- datasets count: expected=0 for failed case actual=%s' "${failures}" "${datasets_count}"
  fi

  {
    printf 'exit_code=%s\n' "${cmd_exit}"
    printf 'run_count=%s\n' "${run_count}"
    printf 'run_statuses=%s\n' "${run_statuses}"
    printf 'run_sources=%s\n' "${run_source_count}"
    printf 'datasets=%s\n' "${datasets_count}"
    printf 'columns=%s\n' "${columns_count}"
    printf 'column_stats=%s\n' "${stats_count}"
    printf 'column_top_values=%s\n' "${top_values_count}"
  } >> "${case_log}"

  cleanup_all >> "${case_log}" 2>&1 < /dev/null || true

  if [[ -n "${failures}" ]]; then
    printf 'FAIL %s/%s%s\n\n' "${category}" "${case_id}" "${failures}" | tee -a "${SUMMARY_FILE}"
    return 1
  fi

  printf 'PASS %s/%s runs=%s datasets=%s columns=%s stats=%s top_values=%s\n\n' \
    "${category}" "${case_id}" "${run_count}" "${datasets_count}" "${columns_count}" "${stats_count}" "${top_values_count}" \
    | tee -a "${SUMMARY_FILE}"
}

main() {
  local failures=0
  local category
  local case_file
  local case_id
  local -a categories

  if [[ "$#" -gt 0 ]]; then
    categories=("$@")
  else
    categories=(files postgres api mixed diff_files diff_postgres diff_api diff_mixed)
  fi

  cleanup_all

  for category in "${categories[@]}"; do
    while IFS= read -r case_file; do
      case_id="$(basename "${case_file}" .yaml)"
      if ! verify_case "${category}" "${case_id}"; then
        failures=$((failures + 1))
      fi
    done < <(find "./testcases/${category}" -maxdepth 1 -type f -name '*.yaml' | sort -V)
  done

  if [[ "${failures}" -gt 0 ]]; then
    printf 'DONE with %s failing cases. Summary: %s\n' "${failures}" "${SUMMARY_FILE}"
    exit 1
  fi

  printf 'DONE all cases passed. Summary: %s\n' "${SUMMARY_FILE}"
}

main "$@"
