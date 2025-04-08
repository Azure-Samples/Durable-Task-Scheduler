targetScope = 'subscription'

@minLength(1)
@maxLength(64)
@description('Name of the environment that can be used as part of naming resource convention')
param environmentName string

@minLength(1)
@description('Primary location for all resources')
param location string

@description('Id of the user or app to assign application roles')
param principalId string = ''

var tags = {
  'azd-env-name': environmentName
}

// Role definition ID for "Durable Task Data Contributor"
var durableTaskDataContributorRoleId = '0ad04412-c4d5-4796-b79c-f76d14c8d402'

resource resourceGroup 'Microsoft.Resources/resourceGroups@2021-04-01' = {
  name: '${environmentName}-rg'
  location: location
  tags: tags
}

// Container Apps Environment and required services
module containerAppsEnvironment 'modules/container-apps-environment.bicep' = {
  name: 'container-apps-environment'
  scope: resourceGroup
  params: {
    name: '${environmentName}-containerapps-env'
    location: location
    tags: tags
  }
}

// Durable Task Scheduler
module durableTaskScheduler 'modules/durable-task-scheduler.bicep' = {
  name: 'durable-task-scheduler'
  scope: resourceGroup
  params: {
    name: '${environmentName}-dts'
    location: location
    tags: tags
    ipAllowlist: ['0.0.0.0/0']
    taskhubname: 'default'
    skuName: 'Dedicated'
    skuCapacity: 1
  }
}

// User assigned managed identities for services
module orchestrationServiceIdentity 'modules/managed-identity.bicep' = {
  name: 'orchestration-service-identity'
  scope: resourceGroup
  params: {
    name: '${environmentName}-orchestration-identity'
    location: location
    tags: tags
  }
}

module workerServiceIdentity 'modules/managed-identity.bicep' = {
  name: 'worker-service-identity'
  scope: resourceGroup
  params: {
    name: '${environmentName}-worker-identity'
    location: location
    tags: tags
  }
}

// Role assignments for Durable Task Scheduler
// Orchestration Service
module orchestrationServiceDtsAccess 'modules/dts-access.bicep' = {
  name: 'orchestration-service-dts-access'
  scope: resourceGroup
  params: {
    roleDefinitionID: durableTaskDataContributorRoleId
    principalID: orchestrationServiceIdentity.outputs.principalId
    principalType: 'ServicePrincipal'
    dtsName: durableTaskScheduler.outputs.dts_NAME
  }
}

// Worker Service
module workerServiceDtsAccess 'modules/dts-access.bicep' = {
  name: 'worker-service-dts-access'
  scope: resourceGroup
  params: {
    roleDefinitionID: durableTaskDataContributorRoleId
    principalID: workerServiceIdentity.outputs.principalId
    principalType: 'ServicePrincipal'
    dtsName: durableTaskScheduler.outputs.dts_NAME
  }
}

// Dashboard access for the current user (if principalId is provided)
module dashboardDtsAccess 'modules/dts-access.bicep' = if (!empty(principalId)) {
  name: 'dashboard-dts-access'
  scope: resourceGroup
  params: {
    roleDefinitionID: durableTaskDataContributorRoleId
    principalID: principalId
    principalType: 'User'
    dtsName: durableTaskScheduler.outputs.dts_NAME
  }
}

// Application services
module orchestrationService 'modules/container-app.bicep' = {
  name: 'orchestration-service'
  scope: resourceGroup
  params: {
    name: '${environmentName}-orchestration-service'
    location: location
    tags: tags
    environmentId: containerAppsEnvironment.outputs.id
    containerImage: 'orchestration-service:latest'
    containerPort: 8080
    containerRegistry: ''
    containerRegistryUsername: ''
    containerRegistryPassword: ''
    userAssignedIdentityId: orchestrationServiceIdentity.outputs.id
    env: [
      {
        name: 'ASPNETCORE_ENVIRONMENT'
        value: 'Production'
      }
      {
        name: 'DURABLE_TASK_SCHEDULER_CONNECTION_STRING'
        value: durableTaskScheduler.outputs.connectionString
      }
    ]
    external: true
  }
}

module workerService 'modules/container-app.bicep' = {
  name: 'worker-service'
  scope: resourceGroup
  params: {
    name: '${environmentName}-worker-service'
    location: location
    tags: tags
    environmentId: containerAppsEnvironment.outputs.id
    containerImage: 'worker-service:latest'
    containerPort: 8080
    containerRegistry: ''
    containerRegistryUsername: ''
    containerRegistryPassword: ''
    userAssignedIdentityId: workerServiceIdentity.outputs.id
    env: [
      {
        name: 'ASPNETCORE_ENVIRONMENT'
        value: 'Production'
      }
      {
        name: 'DURABLE_TASK_SCHEDULER_CONNECTION_STRING'
        value: durableTaskScheduler.outputs.connectionString
      }
    ]
    external: false
  }
}

output ORCHESTRATION_SERVICE_URI string = orchestrationService.outputs.uri
output WORKER_SERVICE_URI string = workerService.outputs.uri
output AZURE_LOCATION string = location
