# Attaches the sample's user-assigned managed identity to the existing Durable Task
# Scheduler so DTS can use it to pull the sandbox image and let the sandbox worker
# connect back. Runs as an azd postprovision hook on Windows (the POSIX equivalent is
# attach-scheduler-identity.sh). Uses the `durabletask` Azure CLI extension's identity
# command, which adds the identity without removing any that are already attached to
# the scheduler.
#
# NOTE: enabling the On-demand Sandboxes preview *feature* on the scheduler is a
# separate, out-of-band step handled during private-preview onboarding.

$ErrorActionPreference = 'Stop'

function Get-RequiredEnv([string]$name) {
    $value = [Environment]::GetEnvironmentVariable($name)
    if ([string]::IsNullOrEmpty($value)) {
        throw "$name must be set"
    }
    return $value
}

$SubscriptionId = Get-RequiredEnv 'AZURE_SUBSCRIPTION_ID'
$SchedulerName  = Get-RequiredEnv 'DTS_SCHEDULER_NAME'
$SchedulerRg    = Get-RequiredEnv 'DTS_SCHEDULER_RESOURCE_GROUP'
$IdentityId     = Get-RequiredEnv 'AZURE_USER_ASSIGNED_IDENTITY_RESOURCE_ID'

Write-Host "==> Ensuring the 'durabletask' Azure CLI extension is installed..."
az extension add --name durabletask --upgrade --only-show-errors | Out-Null
if ($LASTEXITCODE -ne 0) { throw "Failed to add the 'durabletask' az extension" }

Write-Host "==> Attaching managed identity to scheduler '$SchedulerName'..."
az durabletask scheduler identity assign `
    --subscription $SubscriptionId `
    --resource-group $SchedulerRg `
    --name $SchedulerName `
    --user-assigned $IdentityId | Out-Null
if ($LASTEXITCODE -ne 0) { throw "Failed to assign identity to scheduler '$SchedulerName'" }

$IdentityName = $IdentityId.Substring($IdentityId.LastIndexOf('/') + 1)
Write-Host "==> Done. Identity $IdentityName is attached to '$SchedulerName'."
