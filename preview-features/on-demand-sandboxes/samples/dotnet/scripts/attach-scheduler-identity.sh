#!/usr/bin/env bash
# Attaches the sample's user-assigned managed identity to the existing Durable Task
# Scheduler so DTS can use it to pull the sandbox image and let the sandbox worker
# connect back. Runs as an azd postprovision hook. Uses the `durabletask` Azure CLI
# extension's identity command, which adds the identity without removing any that are
# already attached to the scheduler.
#
# NOTE: enabling the On-demand Sandboxes preview *feature* on the scheduler is a
# separate, out-of-band step handled during private-preview onboarding.

set -eu

SUBSCRIPTION_ID="${AZURE_SUBSCRIPTION_ID:?AZURE_SUBSCRIPTION_ID must be set}"
SCHEDULER_NAME="${DTS_SCHEDULER_NAME:?DTS_SCHEDULER_NAME must be set}"
SCHEDULER_RG="${DTS_SCHEDULER_RESOURCE_GROUP:?DTS_SCHEDULER_RESOURCE_GROUP must be set}"
IDENTITY_ID="${AZURE_USER_ASSIGNED_IDENTITY_RESOURCE_ID:?AZURE_USER_ASSIGNED_IDENTITY_RESOURCE_ID must be set}"

echo "==> Ensuring the 'durabletask' Azure CLI extension is installed..."
az extension add --name durabletask --upgrade --only-show-errors >/dev/null

echo "==> Attaching managed identity to scheduler '${SCHEDULER_NAME}'..."
az durabletask scheduler identity assign \
    --subscription "${SUBSCRIPTION_ID}" \
    --resource-group "${SCHEDULER_RG}" \
    --name "${SCHEDULER_NAME}" \
    --user-assigned "${IDENTITY_ID}" >/dev/null

echo "==> Done. Identity ${IDENTITY_ID##*/} is attached to '${SCHEDULER_NAME}'."
