# Runs App A (port 7071) and App B (port 7072) against the same DTS task hub.
# Demonstrates work item filtering: each app only processes the orchestrations
# and activities it has registered. Cross-app scheduling still works because
# DTS routes each work item to the app whose filter matches.
#
# Usage:
#   .\run-both.ps1                 # start both apps, wait, exercise scenarios
#   .\run-both.ps1 -StartOnly      # just start them and leave running
#   .\run-both.ps1 -StopOnly       # kill any running func hosts on 7071/7072

[CmdletBinding()]
param(
    [switch]$StartOnly,
    [switch]$StopOnly
)

$ErrorActionPreference = 'Stop'
$root  = Split-Path -Parent $MyInvocation.MyCommand.Path
$appA  = Join-Path $root 'WorkItemFiltering'
$appB  = Join-Path $root 'WorkItemFiltering.AppB'
$logA  = Join-Path $env:TEMP 'wif-appA.log'
$logB  = Join-Path $env:TEMP 'wif-appB.log'

function Stop-Funcs {
    Get-Process func, Microsoft.Azure.Functions.JobHost -ErrorAction SilentlyContinue |
        Stop-Process -Force -ErrorAction SilentlyContinue
}

Stop-Funcs
if ($StopOnly) { Write-Host 'Stopped func hosts.'; return }

Write-Host '== Building App A and App B =='
dotnet build $appA -c Debug --nologo | Select-Object -Last 3
dotnet build $appB -c Debug --nologo | Select-Object -Last 3

Remove-Item $logA, $logB -ErrorAction SilentlyContinue

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
    'http://localhost:7071/api/start/OrdersOrchestration'

Invoke-Scenario 'CROSS-APP: schedule App A orchestration from App B client' `
    'http://localhost:7072/api/start/GreetingOrchestration'

Invoke-Scenario 'UNKNOWN: orchestration that no app has registered' `
    'http://localhost:7071/api/start/NobodyOwnsThis'

Write-Host "`n== Filter registration log lines =="
Write-Host '-- App A --'
Select-String -Path $logA -Pattern 'Work item filtering|Registered \d+ orch' | Select-Object -Last 2
Write-Host '-- App B --'
Select-String -Path $logB -Pattern 'Work item filtering|Registered \d+ orch' | Select-Object -Last 2

Write-Host "`nLogs: $logA  /  $logB"
Write-Host 'Apps still running. Stop with: .\run-both.ps1 -StopOnly'
