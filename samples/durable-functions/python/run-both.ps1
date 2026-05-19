# Runs work-item-filtering App A (port 7071) and App B (port 7072) against the
# same DTS task hub. Both apps have manual extensions (extensions.csproj
# referencing Microsoft.Azure.WebJobs.Extensions.DurableTask.AzureManaged
# v1.8.1 from the local NuGet feed) so they pick up the latest filter changes
# instead of the extension bundle.
#
# Usage:
#   .\run-both.ps1                 # set up venv, install extensions, run scenarios
#   .\run-both.ps1 -StartOnly      # set up & start, leave running
#   .\run-both.ps1 -StopOnly       # kill any running func hosts

[CmdletBinding()]
param(
    [switch]$StartOnly,
    [switch]$StopOnly,
    [switch]$SkipSetup
)

$ErrorActionPreference = 'Stop'
$root  = Split-Path -Parent $MyInvocation.MyCommand.Path
$appA  = Join-Path $root 'work-item-filtering'
$appB  = Join-Path $root 'work-item-filtering-app-b'
$venv  = Join-Path $root '.venv-wif'
$logA  = Join-Path $env:TEMP 'wif-py-appA.log'
$logB  = Join-Path $env:TEMP 'wif-py-appB.log'

function Stop-Funcs {
    Get-Process func, Microsoft.Azure.Functions.JobHost, python -ErrorAction SilentlyContinue |
        Where-Object {
            $_.ProcessName -in @('func','Microsoft.Azure.Functions.JobHost') -or
            ($_.ProcessName -eq 'python' -and $_.Path -like "$venv*")
        } |
        Stop-Process -Force -ErrorAction SilentlyContinue
}

Stop-Funcs
if ($StopOnly) { Write-Host 'Stopped func hosts.'; return }

if (-not $SkipSetup) {
    if (-not (Test-Path $venv)) {
        Write-Host '== Creating shared venv =='
        python -m venv $venv
    }
    $py = Join-Path $venv 'Scripts\python.exe'
    Write-Host '== Installing Python deps =='
    & $py -m pip install --quiet --upgrade pip
    & $py -m pip install --quiet -r (Join-Path $appA 'requirements.txt')

    foreach ($app in @($appA, $appB)) {
        Write-Host "== func extensions install in $(Split-Path -Leaf $app) =="
        Push-Location $app
        try {
            # func extensions install shells out to `dotnet build` of extensions.csproj
            # The local NuGet.config feed will be picked up automatically.
            func extensions install --force 2>&1 | Select-Object -Last 4
        } finally { Pop-Location }
    }
}

# Activate venv for this shell so Start-Process inherits the right python on PATH
$env:VIRTUAL_ENV = $venv
$env:PATH = "$venv\Scripts;$env:PATH"

Remove-Item $logA, $logB, "$logA.err", "$logB.err" -ErrorAction SilentlyContinue

Write-Host '== Starting App A on :7071 =='
$pA = Start-Process -FilePath func -ArgumentList 'start','--port','7071' `
    -WorkingDirectory $appA -RedirectStandardOutput $logA `
    -RedirectStandardError "$logA.err" -NoNewWindow -PassThru

Write-Host '== Starting App B on :7072 =='
$pB = Start-Process -FilePath func -ArgumentList 'start','--port','7072' `
    -WorkingDirectory $appB -RedirectStandardOutput $logB `
    -RedirectStandardError "$logB.err" -NoNewWindow -PassThru

Write-Host "App A PID=$($pA.Id)  App B PID=$($pB.Id)"
Write-Host 'Waiting 30s for both hosts to start and register filters...'
Start-Sleep 30

if ($StartOnly) {
    Write-Host 'Both apps running. Stop with: .\run-both.ps1 -StopOnly'
    return
}

function Invoke-Scenario([string]$Name, [string]$Url) {
    Write-Host "`n-- $Name --"
    try {
        $r = Invoke-RestMethod -Method Post -Uri $Url -TimeoutSec 30
        Start-Sleep 8
        $s = Invoke-RestMethod -Uri $r.statusQueryGetUri
        Write-Host ("    status={0}  output={1}" -f $s.runtimeStatus, ($s.output | ConvertTo-Json -Compress))
    } catch {
        Write-Host "    ERROR: $_"
    }
}

Invoke-Scenario 'App A orchestration via App A client (own filter)' `
    'http://localhost:7071/api/orchestrators/greeting'

Invoke-Scenario 'App B orchestration via App B client (own filter)' `
    'http://localhost:7072/api/orchestrators/orders'

Invoke-Scenario 'CROSS-APP: schedule App B orchestration from App A client' `
    'http://localhost:7071/api/start/orders_orchestration'

Invoke-Scenario 'CROSS-APP: schedule App A orchestration from App B client' `
    'http://localhost:7072/api/start/greeting_orchestration'

Invoke-Scenario 'UNKNOWN: orchestration that no app has registered' `
    'http://localhost:7071/api/start/nobody_owns_this'

Write-Host "`n== Filter registration log lines =="
Write-Host '-- App A --'
Select-String -Path $logA -Pattern 'Work item filtering|Registered \d+ orch|AzureManagedProvider' | Select-Object -Last 3
Write-Host '-- App B --'
Select-String -Path $logB -Pattern 'Work item filtering|Registered \d+ orch|AzureManagedProvider' | Select-Object -Last 3

Write-Host "`nLogs: $logA  /  $logB"
Write-Host 'Apps still running. Stop with: .\run-both.ps1 -StopOnly'
