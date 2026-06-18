# Builds the two container images for the On-demand Sandboxes demo server-side using
# ACR Tasks (az acr build) - no local Docker required. Called by azd as a predeploy hook
# on Windows (the POSIX equivalent is acr-build.sh).
#
#   - main-app   : the orchestrator, deployed to AKS (azd reads SERVICE_MAINAPP_IMAGE_NAME
#                  and skips its own build/push).
#   - sandbox    : the worker image DTS starts on demand. Not deployed to AKS; its full
#                  image reference is handed to the app via DTS_SANDBOX_CONTAINER_IMAGE.

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
$Tag = "azd-deploy-$([DateTimeOffset]::UtcNow.ToUnixTimeSeconds())"

# The .NET build context is the sample root so Directory.Build.props is available.
function Build-Image([string]$imageRepo, [string]$containerfile) {
    $fullImage = "$RegistryEndpoint/${imageRepo}:$Tag"

    Write-Host "==> Building ${imageRepo}:$Tag via ACR Tasks (--platform linux/amd64)..."
    # The classic ACR builder does not auto-populate the BuildKit TARGETARCH arg, so we
    # pass it explicitly. We always build linux/amd64 here, so amd64 is correct.
    az acr build `
        --registry $Registry `
        --image "${imageRepo}:$Tag" `
        --platform linux/amd64 `
        --build-arg TARGETARCH=amd64 `
        --file $containerfile `
        . `
        --no-logs `
        --output none
    if ($LASTEXITCODE -ne 0) { throw "az acr build failed for $imageRepo" }

    return $fullImage
}

$MainAppImage = Build-Image "dts-ondemand-sandboxes/main-app-$EnvName" "main-app/Containerfile"
$SandboxImage = Build-Image "dts-ondemand-sandboxes/sandbox-worker-$EnvName" "sandbox-worker/Containerfile"

# azd uses SERVICE_<NAME>_IMAGE_NAME to skip its own build and deploy this image instead.
azd env set SERVICE_MAINAPP_IMAGE_NAME $MainAppImage
# The app declares the sandbox worker profile using this image reference.
azd env set DTS_SANDBOX_CONTAINER_IMAGE $SandboxImage

Write-Host "==> main-app image  : $MainAppImage"
Write-Host "==> sandbox image   : $SandboxImage"
