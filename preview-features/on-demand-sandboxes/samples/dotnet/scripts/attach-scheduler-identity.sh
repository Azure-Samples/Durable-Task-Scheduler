#!/usr/bin/env bash
# Attaches the sample's user-assigned managed identity to the existing Durable Task
# Scheduler so DTS can use it to pull the sandbox image and let the sandbox worker
# connect back. Runs as an azd postprovision hook. The PATCH is merge-safe: it keeps
# any identities already attached to the scheduler.
#
# NOTE: enabling the On-demand Sandboxes preview *feature* on the scheduler is a
# separate, out-of-band step handled during private-preview onboarding.

set -euo pipefail

SUBSCRIPTION_ID="${AZURE_SUBSCRIPTION_ID:?AZURE_SUBSCRIPTION_ID must be set}"
SCHEDULER_NAME="${DTS_SCHEDULER_NAME:?DTS_SCHEDULER_NAME must be set}"
SCHEDULER_RG="${DTS_SCHEDULER_RESOURCE_GROUP:?DTS_SCHEDULER_RESOURCE_GROUP must be set}"
IDENTITY_ID="${AZURE_USER_ASSIGNED_IDENTITY_RESOURCE_ID:?AZURE_USER_ASSIGNED_IDENTITY_RESOURCE_ID must be set}"
API_VERSION="2026-05-01-preview"

if ! command -v python3 >/dev/null 2>&1; then
    echo "ERROR: python3 is required to merge the scheduler identity block." >&2
    exit 1
fi

URI="https://management.azure.com/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${SCHEDULER_RG}/providers/Microsoft.DurableTask/schedulers/${SCHEDULER_NAME}?api-version=${API_VERSION}"

echo "==> Reading current identity on scheduler '${SCHEDULER_NAME}'..."
CURRENT="$(az rest --method get --uri "${URI}")"

BODY="$(IDENTITY_ID="${IDENTITY_ID}" python3 - "${CURRENT}" <<'PY'
import json, os, sys

current = json.loads(sys.argv[1])
identity_id = os.environ["IDENTITY_ID"]

identity = current.get("identity") or {}
user_assigned = identity.get("userAssignedIdentities") or {}
user_assigned[identity_id] = {}

current_type = identity.get("type", "") or ""
new_type = "SystemAssigned, UserAssigned" if "SystemAssigned" in current_type else "UserAssigned"

print(json.dumps({"identity": {"type": new_type, "userAssignedIdentities": user_assigned}}))
PY
)"

TMP="$(mktemp)"
trap 'rm -f "${TMP}"' EXIT
printf '%s' "${BODY}" > "${TMP}"

echo "==> Attaching managed identity to scheduler..."
az rest --method patch --uri "${URI}" --body "@${TMP}" >/dev/null
echo "==> Done. Identity ${IDENTITY_ID##*/} is attached to '${SCHEDULER_NAME}'."
