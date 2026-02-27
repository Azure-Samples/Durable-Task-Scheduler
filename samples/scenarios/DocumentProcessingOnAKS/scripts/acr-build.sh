#!/usr/bin/env bash
# Builds Docker images server-side using ACR Tasks (az acr build).
# This avoids local Docker networking issues and doesn't require Docker Desktop.
# Called by azd as a predeploy hook.

set -euo pipefail

REGISTRY="${AZURE_CONTAINER_REGISTRY_NAME:?AZURE_CONTAINER_REGISTRY_NAME must be set}"
REGISTRY_ENDPOINT="${AZURE_CONTAINER_REGISTRY_ENDPOINT:?AZURE_CONTAINER_REGISTRY_ENDPOINT must be set}"
TAG="azd-deploy-$(date +%s)"

build_and_set() {
    local service_name="$1"      # e.g. client, worker
    local project_dir="$2"       # e.g. ./Client
    local image_repo="durable-task-on-aks/${service_name}-${AZURE_ENV_NAME}"
    local full_image="${REGISTRY_ENDPOINT}/${image_repo}:${TAG}"

    echo "==> Building ${service_name} via ACR Tasks (${image_repo}:${TAG})..."
    az acr build \
        --registry "${REGISTRY}" \
        --image "${image_repo}:${TAG}" \
        "${project_dir}" \
        --no-logs

    # Persist the image name so the .tmpl.yaml manifest can reference it
    local env_var="SERVICE_$(echo "${service_name}" | tr '[:lower:]' '[:upper:]')_IMAGE_NAME"
    azd env set "${env_var}" "${full_image}"
    echo "==> Set ${env_var}=${full_image}"
}

build_and_set "client" "./Client"
build_and_set "worker" "./Worker"

echo "==> All images built and pushed to ${REGISTRY_ENDPOINT}"
