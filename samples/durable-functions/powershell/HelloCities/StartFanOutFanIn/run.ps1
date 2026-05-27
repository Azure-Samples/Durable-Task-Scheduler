param($Request, $TriggerMetadata)

$instanceId = Start-DurableOrchestration -FunctionName 'FanOutFanInOrchestration'
Write-Host "Started fan-out/fan-in orchestration with ID = '$instanceId'."

$response = New-DurableOrchestrationCheckStatusResponse -Request $Request -InstanceId $instanceId
Push-OutputBinding -Name Response -Value $response
