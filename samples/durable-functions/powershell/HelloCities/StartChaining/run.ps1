param($Request, $starter, $TriggerMetadata)

$instanceId = Start-DurableOrchestration -FunctionName 'ChainingOrchestration' -DurableClient $starter
Write-Host "Started chaining orchestration with ID = '$instanceId'."

$response = New-DurableOrchestrationCheckStatusResponse -Request $Request -InstanceId $instanceId -DurableClient $starter
Push-OutputBinding -Name Response -Value $response
