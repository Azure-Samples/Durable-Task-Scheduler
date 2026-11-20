param schedulerName string
param taskHubName string

@minValue(1)
param ordersActionsPerSecond int = 20

@minValue(1)
param partnerActionsPerSecond int = 5

resource scheduler 'Microsoft.DurableTask/schedulers@2026-02-01' existing = {
  name: schedulerName
}

resource taskHub 'Microsoft.DurableTask/schedulers/taskHubs@2026-02-01' existing = {
  parent: scheduler
  name: taskHubName
}

resource orders 'Microsoft.DurableTask/schedulers/taskHubs/taskQueues@2026-07-01-preview' = {
  parent: taskHub
  name: 'orders'
  properties: {
    throughput: {
      actionsPerSecond: ordersActionsPerSecond
    }
  }
}

resource partnerApi 'Microsoft.DurableTask/schedulers/taskHubs/taskQueues@2026-07-01-preview' = {
  parent: taskHub
  name: 'partner-api'
  properties: {
    throughput: {
      actionsPerSecond: partnerActionsPerSecond
    }
  }
}
