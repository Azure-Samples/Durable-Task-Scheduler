targetScope = 'subscription'

@minLength(1)
@maxLength(64)
@description('Name of the environment which is used to generate a short unique hash used in all resources.')
param environmentName string

@minLength(1)
@description('Primary location for all resources')
param location string

@description('Id of the user or app to assign application roles')
param principalId string = ''

param containerAppsEnvName string = ''
param containerAppsAppName string = ''
param containerRegistryName string = ''
param dtsLocation string = 'centralus'
param dtsSkuName string = 'Consumption'
param dtsCapacity int = 1
param dtsName string = ''
param taskHubName string = ''

param clientServiceName string = 'client'
param orchestratorWorkerServiceName string = 'orchestrator-worker'
param validatorWorkerServiceName string = 'validator-worker'
param shipperWorkerServiceName string = 'shipper-worker'

param resourceGroupName string = ''

var abbrs = loadJsonContent('./abbreviations.json')

var tags = {
  'azd-env-name': environmentName
}

#disable-next-line no-unused-vars
var resourceToken = toLower(uniqueString(subscription().id, environmentName, location))

// Resource Group
resource rg 'Microsoft.Resources/resourceGroups@2021-04-01' = {
  name: !empty(resourceGroupName) ? resourceGroupName : '${abbrs.resourcesResourceGroups}${environmentName}'
  location: location
  tags: tags
}

// User-assigned managed identity for all container apps
module identity './app/user-assigned-identity.bicep' = {
  name: 'identity'
  scope: rg
  params: {
    name: 'dts-ca-identity'
  }
}

// Assign DTS Worker/Client role to the managed identity
module identityAssignDTS './core/security/role.bicep' = {
  name: 'identityAssignDTS'
  scope: rg
  params: {
    principalId: identity.outputs.principalId
    roleDefinitionId: '0ad04412-c4d5-4796-b79c-f76d14c8d402'
    principalType: 'ServicePrincipal'
  }
}

// Assign DTS role to the deploying user (for dashboard access)
module identityAssignDTSDash './core/security/role.bicep' = {
  name: 'identityAssignDTSDash'
  scope: rg
  params: {
    principalId: principalId
    roleDefinitionId: '0ad04412-c4d5-4796-b79c-f76d14c8d402'
    principalType: 'User'
  }
}

// Virtual network
module vnet './core/networking/vnet.bicep' = {
  name: 'vnet'
  scope: rg
  params: {
    name: '${abbrs.networkVirtualNetworks}${resourceToken}'
    location: location
    tags: tags
  }
}

// Container Apps Environment + Container Registry
module containerAppsEnv './core/host/container-apps.bicep' = {
  name: 'container-apps'
  scope: rg
  params: {
    name: 'app'
    containerAppsEnvironmentName: !empty(containerAppsEnvName) ? containerAppsEnvName : '${abbrs.appManagedEnvironments}${resourceToken}'
    containerRegistryName: !empty(containerRegistryName) ? containerRegistryName : '${abbrs.containerRegistryRegistries}${resourceToken}'
    location: location
    subnetResourceId: vnet.outputs.infrastructureSubnetId
    loadBalancerType: 'External'
  }
}

// Durable Task Scheduler + Task Hub
module dts './app/dts.bicep' = {
  scope: rg
  name: 'dtsResource'
  params: {
    name: !empty(dtsName) ? dtsName : '${abbrs.dts}${resourceToken}'
    taskhubname: !empty(taskHubName) ? taskHubName : '${abbrs.taskhub}${resourceToken}'
    location: dtsLocation
    tags: tags
    ipAllowlist: ['0.0.0.0/0']
    skuName: dtsSkuName
    skuCapacity: dtsCapacity
  }
}

// Client — schedules orchestrations and polls for results
module client 'app/app.bicep' = {
  name: clientServiceName
  scope: rg
  params: {
    appName: !empty(containerAppsAppName) ? '${containerAppsAppName}-client' : '${abbrs.appContainerApps}${resourceToken}-client'
    containerAppsEnvironmentName: containerAppsEnv.outputs.environmentName
    containerRegistryName: containerAppsEnv.outputs.registryName
    userAssignedManagedIdentity: {
      resourceId: identity.outputs.resourceId
      clientId: identity.outputs.clientId
    }
    location: location
    tags: tags
    serviceName: 'client'
    identityName: identity.outputs.name
    dtsEndpoint: dts.outputs.dts_URL
    taskHubName: dts.outputs.TASKHUB_NAME
  }
}

// Orchestrator Worker — handles orchestrations only
module orchestratorWorker 'app/app.bicep' = {
  name: orchestratorWorkerServiceName
  scope: rg
  params: {
    appName: !empty(containerAppsAppName) ? '${containerAppsAppName}-orchestrator' : '${abbrs.appContainerApps}${resourceToken}-orchestrator'
    containerAppsEnvironmentName: containerAppsEnv.outputs.environmentName
    containerRegistryName: containerAppsEnv.outputs.registryName
    userAssignedManagedIdentity: {
      resourceId: identity.outputs.resourceId
      clientId: identity.outputs.clientId
    }
    location: location
    tags: tags
    serviceName: 'orchestrator-worker'
    identityName: identity.outputs.name
    dtsEndpoint: dts.outputs.dts_URL
    taskHubName: dts.outputs.TASKHUB_NAME
    workItemType: 'Orchestration'
  }
}

// Validator Worker — handles ValidateOrder activity only
module validatorWorker 'app/app.bicep' = {
  name: validatorWorkerServiceName
  scope: rg
  params: {
    appName: !empty(containerAppsAppName) ? '${containerAppsAppName}-validator' : '${abbrs.appContainerApps}${resourceToken}-validator'
    containerAppsEnvironmentName: containerAppsEnv.outputs.environmentName
    containerRegistryName: containerAppsEnv.outputs.registryName
    userAssignedManagedIdentity: {
      resourceId: identity.outputs.resourceId
      clientId: identity.outputs.clientId
    }
    location: location
    tags: tags
    serviceName: 'validator-worker'
    identityName: identity.outputs.name
    dtsEndpoint: dts.outputs.dts_URL
    taskHubName: dts.outputs.TASKHUB_NAME
    workItemType: 'Activity'
  }
}

// Shipper Worker — handles ShipOrder activity only
module shipperWorker 'app/app.bicep' = {
  name: shipperWorkerServiceName
  scope: rg
  params: {
    appName: !empty(containerAppsAppName) ? '${containerAppsAppName}-shipper' : '${abbrs.appContainerApps}${resourceToken}-shipper'
    containerAppsEnvironmentName: containerAppsEnv.outputs.environmentName
    containerRegistryName: containerAppsEnv.outputs.registryName
    userAssignedManagedIdentity: {
      resourceId: identity.outputs.resourceId
      clientId: identity.outputs.clientId
    }
    location: location
    tags: tags
    serviceName: 'shipper-worker'
    identityName: identity.outputs.name
    dtsEndpoint: dts.outputs.dts_URL
    taskHubName: dts.outputs.TASKHUB_NAME
    workItemType: 'Activity'
  }
}

output AZURE_LOCATION string = location
output AZURE_TENANT_ID string = tenant().tenantId
output AZURE_CONTAINER_REGISTRY_ENDPOINT string = containerAppsEnv.outputs.registryLoginServer
output AZURE_CONTAINER_REGISTRY_NAME string = containerAppsEnv.outputs.registryName
output AZURE_USER_ASSIGNED_IDENTITY_NAME string = identity.outputs.name
