param($Request, $TriggerMetadata)

$instanceId = Start-DurableOrchestration -FunctionName 'ChainingOrchestration'
Write-Host "Started chaining orchestration with ID = '$instanceId'."

$response = New-DurableOrchestrationCheckStatusResponse -Request $Request -InstanceId $instanceId
Push-OutputBinding -Name Response -Value $response
