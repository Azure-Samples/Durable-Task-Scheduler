# Azure Functions profile.ps1
#
# This profile.ps1 will get executed every "cold start" of your Function App.
# "cold start" occurs when:
#
# * A Function App starts up for the very first time
# * A Function App starts up after being de-allocated due to inactivity
#
# You can define helper functions, run commands, or specify environment variables.
# NOTE: variables defined outside a function are stored in the script scope and
#       are not automatically available inside your function scripts unless you
#       explicitly pass them in or define them in a shared module.

# Register HttpStatusCode type accelerator (required by Durable Functions module)
$accelerator = [PowerShell].Assembly.GetType("System.Management.Automation.TypeAccelerators")
$accelerator::Add('HttpStatusCode', [System.Net.HttpStatusCode])

# Authenticate with Azure PowerShell using MSI (if deployed to Azure)
if ($env:MSI_SECRET) {
    Disable-AzContextAutosave -Scope Process | Out-Null
    Connect-AzAccount -Identity
}
