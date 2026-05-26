param($Context)

$cities = @('Tokyo', 'Seattle', 'London', 'Paris', 'Berlin')

# Fan-out: schedule all activities in parallel
$parallelTasks = @()
foreach ($city in $cities) {
    $parallelTasks += Invoke-DurableActivity -FunctionName 'SayHello' -Input $city -NoWait
}

# Fan-in: wait for all to complete
$output = Wait-ActivityFunction -Task $parallelTasks

$output
