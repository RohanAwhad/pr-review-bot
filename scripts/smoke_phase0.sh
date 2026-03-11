#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_ID="$(date +%Y%m%d_%H%M%S)"
RUN_DIR="${ROOT_DIR}/logs/smoke/run-${RUN_ID}"
KEEP_SMOKE_LOGS="${KEEP_SMOKE_LOGS:-0}"

CASE_NAMES=(
  "reward_hub_64"
  "new_math_mnist_9"
  "invalid_issue_url"
  "first_pass_new_math_mnist_9"
  "first_pass_agentic_rl_1"
)
CASE_COMMANDS=(
  "classify-pr"
  "classify-pr"
  "classify-pr"
  "first-pass"
  "first-pass"
)
CASE_URLS=(
  "https://github.com/Red-Hat-AI-Innovation-Team/reward_hub/pull/64"
  "https://github.com/RohanAwhad/new-math-mnist/pull/9"
  "https://github.com/RohanAwhad/new-math-mnist/issues/9"
  "https://github.com/RohanAwhad/new-math-mnist/pull/9"
  "https://github.com/RohanAwhad/agentic-rl/pull/1"
)
CASE_EXPECTED=("no_human" "human_required" "human_required" "ready_to_merge" "wip")
CASE_JSON_FIELDS=(
  "classification"
  "classification"
  "classification"
  "merge_readiness"
  "merge_readiness"
)

cleanup() {
  if [[ "${KEEP_SMOKE_LOGS}" != "1" ]]; then
    rm -rf "${RUN_DIR}"
  else
    echo "kept smoke logs at ${RUN_DIR}"
  fi
}
trap cleanup EXIT

mkdir -p "${RUN_DIR}"
cd "${ROOT_DIR}"
PIDS=()

for i in "${!CASE_NAMES[@]}"; do
  stdout_file="${RUN_DIR}/${CASE_NAMES[$i]}.stdout"
  stderr_file="${RUN_DIR}/${CASE_NAMES[$i]}.stderr"
  status_file="${RUN_DIR}/${CASE_NAMES[$i]}.exit"
  command_name="${CASE_COMMANDS[$i]}"
  (
    go run "./cmd/${command_name}" "${CASE_URLS[$i]}" >"${stdout_file}" 2>"${stderr_file}"
    echo "$?" >"${status_file}"
    exit 0
  ) &
  PIDS+=("$!")
done

for pid in "${PIDS[@]}"; do
  if ! wait "${pid}"; then
    :
  fi
done

pass_count=0
mismatch_count=0
error_count=0

for i in "${!CASE_NAMES[@]}"; do
  name="${CASE_NAMES[$i]}"
  expected="${CASE_EXPECTED[$i]}"
  json_field="${CASE_JSON_FIELDS[$i]}"
  stdout_file="${RUN_DIR}/${name}.stdout"
  stderr_file="${RUN_DIR}/${name}.stderr"
  status_file="${RUN_DIR}/${name}.exit"
  actual=""
  status=""

  if [[ ! -f "${status_file}" ]]; then
    status="ERROR"
    actual="missing-exit-code"
    ((error_count+=1))
  elif [[ "$(cat "${status_file}")" != "0" ]]; then
    status="ERROR"
    actual="command-failed"
    ((error_count+=1))
  else
    actual="$(jq -r --arg field "${json_field}" '.[$field] // empty' "${stdout_file}" 2>/dev/null || true)"
    if [[ -z "${actual}" ]]; then
      status="ERROR"
      actual="invalid-json"
      ((error_count+=1))
    elif [[ "${actual}" == "${expected}" ]]; then
      status="PASS"
      ((pass_count+=1))
    else
      status="MISMATCH"
      ((mismatch_count+=1))
    fi
  fi

  echo "===== ${name} ====="
  echo "Command: ${CASE_COMMANDS[$i]}"
  echo "URL: ${CASE_URLS[$i]}"
  echo "JSON field: ${json_field}"
  echo "Expected: ${expected}"
  echo "Actual: ${actual}"
  echo "Status: ${status}"
  echo "--- Captured STDOUT ---"
  cat "${stdout_file}"
  echo "--- Captured STDERR ---"
  cat "${stderr_file}"
  echo
done

  echo "Summary: pass=${pass_count} mismatch=${mismatch_count} error=${error_count}"

if [[ ${error_count} -gt 0 ]]; then exit 1; fi
