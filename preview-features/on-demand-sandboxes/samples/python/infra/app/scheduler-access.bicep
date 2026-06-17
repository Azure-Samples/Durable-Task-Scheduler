metadata description = 'Wires up an existing Durable Task Scheduler: ensures the task hub, grants data-plane access, and surfaces the endpoint. Deployed into the scheduler\'s resource group.'

@description('Name of the existing Durable Task Scheduler')
param schedulerName string

@description('Name of the task hub to use (created if it does not already exist)')
param taskHubName string = 'default'

@description('Principal id of the workload identity that connects to DTS')
param workloadPrincipalId string

@description('Principal id of the deploying user, for dashboard access (optional)')
param userPrincipalId string = ''

// The scheduler is created and patched out of band (preview feature enablement +
// managed-identity attach), so it is referenced as an existing resource here.
resource scheduler 'Microsoft.DurableTask/schedulers@2025-11-01' existing = {
  name: schedulerName
}

// Ensure the task hub the app uses exists on the scheduler.
resource taskHub 'Microsoft.DurableTask/schedulers/taskhubs@2025-11-01' = {
  parent: scheduler
  name: taskHubName
}

// Durable Task Data Contributor (0ad04412-c4d5-4796-b79c-f76d14c8d402) — data-plane
// access used by the orchestrator app and the sandbox worker to connect to DTS.
var dtsDataRole = '0ad04412-c4d5-4796-b79c-f76d14c8d402'

resource workloadDtsAccess 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  scope: scheduler
  name: guid(scheduler.id, workloadPrincipalId, dtsDataRole)
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', dtsDataRole)
    principalId: workloadPrincipalId
    principalType: 'ServicePrincipal'
  }
}

resource userDtsAccess 'Microsoft.Authorization/roleAssignments@2022-04-01' = if (!empty(userPrincipalId)) {
  scope: scheduler
  name: guid(scheduler.id, userPrincipalId, dtsDataRole)
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', dtsDataRole)
    principalId: userPrincipalId
    principalType: 'User'
  }
}

output endpoint string = scheduler.properties.endpoint
output taskHubName string = taskHub.name
output schedulerName string = scheduler.name
