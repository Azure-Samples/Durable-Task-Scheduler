#!/usr/bin/env bash
# Builds the two container images for the On-demand Sandboxes demo server-side using
# ACR Tasks (az acr build) — no local Docker required. Called by azd as a predeploy hook.
#
#   - main-app   : the orchestrator, deployed to Azure Container Apps (azd reads
#                  SERVICE_MAINAPP_IMAGE_NAME and skips its own build/push).
#   - sandbox    : the worker image DTS starts on demand. Not deployed as an azd service;
#                  its image reference is set on the Container App by infra/main.bicep
#                  (DTS_SANDBOX_CONTAINER_IMAGE), so it is pushed to that exact tag here.

set -euo pipefail

REGISTRY="${AZURE_CONTAINER_REGISTRY_NAME:?AZURE_CONTAINER_REGISTRY_NAME must be set}"
REGISTRY_ENDPOINT="${AZURE_CONTAINER_REGISTRY_ENDPOINT:?AZURE_CONTAINER_REGISTRY_ENDPOINT must be set}"
ENV_NAME="${AZURE_ENV_NAME:?AZURE_ENV_NAME must be set}"
TAG="azd-deploy-$(date +%s)"

# The .NET build context is the sample root so Directory.Build.props is available.
build() {
    local image_repo="$1"   # e.g. dts-ondemand-sandboxes/main-app-<env>
    local image_tag="$2"
    local containerfile="$3"
    local full_image="${REGISTRY_ENDPOINT}/${image_repo}:${image_tag}"

    echo "==> Building ${image_repo}:${image_tag} via ACR Tasks (--platform linux/amd64)..." >&2
    az acr build \
        --registry "${REGISTRY}" \
        --image "${image_repo}:${image_tag}" \
        --platform linux/amd64 \
        --file "${containerfile}" \
        . \
        --no-logs \
        --output none >&2

    echo "${full_image}"
}

# main-app uses a unique tag so azd rolls out a new Container App revision each deploy.
MAIN_APP_IMAGE="$(build "dts-ondemand-sandboxes/main-app-${ENV_NAME}" "${TAG}" "main-app/Containerfile")"
# The sandbox image uses a stable 'latest' tag matching DTS_SANDBOX_CONTAINER_IMAGE in
# infra/main.bicep; DTS pulls it fresh each time it starts a sandbox.
SANDBOX_IMAGE="$(build "dts-ondemand-sandboxes/sandbox-worker-${ENV_NAME}" "latest" "sandbox-worker/Containerfile")"

# azd uses SERVICE_<NAME>_IMAGE_NAME to skip its own build and deploy this image instead.
azd env set SERVICE_MAINAPP_IMAGE_NAME "${MAIN_APP_IMAGE}"

echo "==> main-app image  : ${MAIN_APP_IMAGE}"
echo "==> sandbox image   : ${SANDBOX_IMAGE}"
