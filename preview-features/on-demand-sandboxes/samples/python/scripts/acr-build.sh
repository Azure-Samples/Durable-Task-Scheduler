#!/usr/bin/env bash
# Builds the sandbox worker image server-side using ACR Tasks (az acr build) — no local
# Docker required. Called by azd as a predeploy hook.
#
# The main-app (orchestrator, main_app.py) image is built by azd itself from
# Containerfile.mainapp (see azure.yaml docker.remoteBuild), so it is intentionally NOT
# built here.
#
# The sandbox image (remote_worker.py) is the worker DTS starts on demand. It is not
# deployed as an azd service; its image reference is set on the Container App by
# infra/main.bicep (DTS_SANDBOX_CONTAINER_IMAGE), so it is pushed to that exact tag here.

set -euo pipefail

REGISTRY="${AZURE_CONTAINER_REGISTRY_NAME:?AZURE_CONTAINER_REGISTRY_NAME must be set}"
REGISTRY_ENDPOINT="${AZURE_CONTAINER_REGISTRY_ENDPOINT:?AZURE_CONTAINER_REGISTRY_ENDPOINT must be set}"
ENV_NAME="${AZURE_ENV_NAME:?AZURE_ENV_NAME must be set}"

build() {
    local image_repo="$1"   # e.g. dts-ondemand-sandboxes/sandbox-worker-<env>
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

# The sandbox image uses a stable 'latest' tag matching DTS_SANDBOX_CONTAINER_IMAGE in
# infra/main.bicep; DTS pulls it fresh each time it starts a sandbox.
SANDBOX_IMAGE="$(build "dts-ondemand-sandboxes/sandbox-worker-${ENV_NAME}" "latest" "Containerfile")"

echo "==> sandbox image   : ${SANDBOX_IMAGE}"
