param($Request, $starter, $TriggerMetadata)

$instanceId = Start-DurableOrchestration -FunctionName 'FanOutFanInOrchestration' -DurableClient $starter
Write-Host "Started fan-out/fan-in orchestration with ID = '$instanceId'."

$response = New-DurableOrchestrationCheckStatusResponse -Request $Request -InstanceId $instanceId -DurableClient $starter
Push-OutputBinding -Name Response -Value $response
