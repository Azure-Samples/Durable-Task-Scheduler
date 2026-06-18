#!/usr/bin/env bash
# Builds the two container images for the On-demand Sandboxes demo server-side using
# ACR Tasks (az acr build) — no local Docker required. Called by azd as a predeploy hook.
#
#   - main-app  : the orchestrator (main_app.py), deployed to AKS (azd reads
#                 SERVICE_MAINAPP_IMAGE_NAME and skips its own build/push).
#   - sandbox   : the worker image (remote_worker.py) DTS starts on demand. Not deployed
#                 to AKS; its full image reference is handed to the app via
#                 DTS_SANDBOX_CONTAINER_IMAGE.

set -euo pipefail

REGISTRY="${AZURE_CONTAINER_REGISTRY_NAME:?AZURE_CONTAINER_REGISTRY_NAME must be set}"
REGISTRY_ENDPOINT="${AZURE_CONTAINER_REGISTRY_ENDPOINT:?AZURE_CONTAINER_REGISTRY_ENDPOINT must be set}"
ENV_NAME="${AZURE_ENV_NAME:?AZURE_ENV_NAME must be set}"
TAG="azd-deploy-$(date +%s)"

build() {
    local image_repo="$1"   # e.g. dts-ondemand-sandboxes/main-app-<env>
    local containerfile="$2"
    local full_image="${REGISTRY_ENDPOINT}/${image_repo}:${TAG}"

    echo "==> Building ${image_repo}:${TAG} via ACR Tasks (--platform linux/amd64)..." >&2
    az acr build \
        --registry "${REGISTRY}" \
        --image "${image_repo}:${TAG}" \
        --platform linux/amd64 \
        --file "${containerfile}" \
        . \
        --no-logs \
        --output none >&2

    echo "${full_image}"
}

MAIN_APP_IMAGE="$(build "dts-ondemand-sandboxes/main-app-${ENV_NAME}" "Containerfile.mainapp")"
SANDBOX_IMAGE="$(build "dts-ondemand-sandboxes/sandbox-worker-${ENV_NAME}" "Containerfile")"

# azd uses SERVICE_<NAME>_IMAGE_NAME to skip its own build and deploy this image instead.
azd env set SERVICE_MAINAPP_IMAGE_NAME "${MAIN_APP_IMAGE}"
# The app declares the sandbox worker profile using this image reference.
azd env set DTS_SANDBOX_CONTAINER_IMAGE "${SANDBOX_IMAGE}"

echo "==> main-app image  : ${MAIN_APP_IMAGE}"
echo "==> sandbox image   : ${SANDBOX_IMAGE}"
