# Builds the sandbox worker image server-side using ACR Tasks (az acr build) - no local
# Docker required. Called by azd as a predeploy hook on Windows (the POSIX equivalent is
# acr-build.sh).
#
# The main-app (orchestrator, main_app.py) image is built by azd itself from
# Containerfile.mainapp (see azure.yaml docker.remoteBuild), so it is intentionally NOT
# built here.
#
# The sandbox image (remote_worker.py) is the worker DTS starts on demand. It is not
# deployed as an azd service; its image reference is set on the Container App by
# infra/main.bicep (DTS_SANDBOX_CONTAINER_IMAGE), so it is pushed to that exact tag here.

$ErrorActionPreference = 'Stop'

function Get-RequiredEnv([string]$name) {
    $value = [Environment]::GetEnvironmentVariable($name)
    if ([string]::IsNullOrEmpty($value)) {
        throw "$name must be set"
    }
    return $value
}

$Registry         = Get-RequiredEnv 'AZURE_CONTAINER_REGISTRY_NAME'
$RegistryEndpoint = Get-RequiredEnv 'AZURE_CONTAINER_REGISTRY_ENDPOINT'
$EnvName          = Get-RequiredEnv 'AZURE_ENV_NAME'

function Build-Image([string]$imageRepo, [string]$imageTag, [string]$containerfile) {
    $fullImage = "$RegistryEndpoint/${imageRepo}:$imageTag"

    Write-Host "==> Building ${imageRepo}:$imageTag via ACR Tasks (--platform linux/amd64)..."
    az acr build `
        --registry $Registry `
        --image "${imageRepo}:$imageTag" `
        --platform linux/amd64 `
        --file $containerfile `
        . `
        --no-logs `
        --output none
    if ($LASTEXITCODE -ne 0) { throw "az acr build failed for $imageRepo" }

    return $fullImage
}

# The sandbox image uses a stable 'latest' tag matching DTS_SANDBOX_CONTAINER_IMAGE in
# infra/main.bicep; DTS pulls it fresh each time it starts a sandbox.
$SandboxImage = Build-Image "dts-ondemand-sandboxes/sandbox-worker-$EnvName" "latest" "Containerfile"

Write-Host "==> sandbox image   : $SandboxImage"
