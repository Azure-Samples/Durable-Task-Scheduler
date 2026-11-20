#!/usr/bin/env bash
set -euo pipefail

baseUrl="${BASE_URL:-http://localhost:7071}"
headers=(-H 'Accept: application/json')
if [[ -n "${FUNCTION_KEY:-}" ]]; then
  headers+=(-H "x-functions-key: $FUNCTION_KEY")
fi
response=$(curl --fail --silent --show-error --max-time 15 \
  "${headers[@]}" -H 'Content-Type: text/plain' -d 'order-1' "$baseUrl/api/orders")
instanceId=$(printf '%s' "$response" | jq -er '.id')
echo "Started $instanceId with DTS_TaskQueue=orders"
deadline=$((SECONDS + 180))
while (( SECONDS < deadline )); do
  response=$(curl --silent --show-error --max-time 10 \
    "${headers[@]}" "$baseUrl/api/orders/$instanceId")
  if ! status=$(printf '%s' "$response" | jq -er '.runtimeStatus'); then
    echo "Invalid status response: $response" >&2
    exit 1
  fi
  case "$status" in
    COMPLETED)
      if ! output=$(printf '%s' "$response" | jq -er '.output'); then
        echo "Missing output in completed response: $response" >&2
        exit 1
      fi
      if [[ "$output" != "Shipped order-1" ]]; then
        echo "Unexpected output: $output" >&2
        exit 1
      fi
      printf '%s' "$response" | jq .
      exit 0
      ;;
    FAILED|TERMINATED|CANCELED)
      printf '%s\n' "$response" >&2
      exit 1
      ;;
    PENDING|RUNNING|SUSPENDED)
      sleep 2
      ;;
    *)
      echo "Unexpected status: $response" >&2
      exit 1
      ;;
  esac
done
echo "Timed out waiting for $instanceId. Inspect its persisted state in the dashboard." >&2
exit 1
