targetScope = 'subscription'

// The main bicep module to provision Azure resources.
// For a more complete walkthrough to understand how this file works with azd,
// see https://learn.microsoft.com/en-us/azure/developer/azure-developer-cli/make-azd-compatible?pivots=azd-create

@minLength(1)
@maxLength(64)
@description('Name of the the environment which is used to generate a short unique hash used in all resources.')
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

param clientsServiceName string = 'client'
param workerServiceName string = 'worker'

// Optional parameters to override the default azd resource naming conventions.
// Add the following to main.parameters.json to provide values:
// "resourceGroupName": {
//      "value": "myGroupName"
// }
param resourceGroupName string = ''

var abbrs = loadJsonContent('./abbreviations.json')

// tags that should be applied to all resources.
var tags = {
  // Tag all resources with the environment name.
  'azd-env-name': environmentName
}

// Generate a unique token to be used in naming resources.
// Remove linter suppression after using.
#disable-next-line no-unused-vars
var resourceToken = toLower(uniqueString(subscription().id, environmentName, location))

// Organize resources in a resource group
resource rg 'Microsoft.Resources/resourceGroups@2021-04-01' = {
  name: !empty(resourceGroupName) ? resourceGroupName : '${abbrs.resourcesResourceGroups}${environmentName}'
  location: location
  tags: tags
}

// Add resources to be provisioned below.
// A full example that leverages azd bicep modules can be seen in the todo-python-mongo template:
// https://github.com/Azure-Samples/todo-python-mongo/tree/main/infra

// Create a user assigned identity
module identity './app/user-assigned-identity.bicep' = {
  name: 'identity'
  scope: rg
  params: {
    name: 'dts-ca-identity'
  }
}

module identityAssignDTS './core/security/role.bicep' = {
  name: 'identityAssignDTS'
  scope: rg
  params: {
    principalId: identity.outputs.principalId
    roleDefinitionId: '0ad04412-c4d5-4796-b79c-f76d14c8d402'
    principalType: 'ServicePrincipal'
  }
}

module identityAssignDTSDash './core/security/role.bicep' = {
  name: 'identityAssignDTSDash'
  scope: rg
  params: {
    principalId: principalId
    roleDefinitionId: '0ad04412-c4d5-4796-b79c-f76d14c8d402'
    principalType: 'User'
  }
}

// Create virtual network with subnets for Container Apps
module vnet './core/networking/vnet.bicep' = {
  name: 'vnet'
  scope: rg
  params: {
    name: '${abbrs.networkVirtualNetworks}${resourceToken}'
    location: location
    tags: tags
  }
}

// Container apps env and registry
module containerAppsEnv './core/host/container-apps.bicep' = {
  name: 'container-apps'
  scope: rg
  params: {
    name: 'app'
    containerAppsEnvironmentName: !empty(containerAppsEnvName) ? containerAppsEnvName : '${abbrs.appManagedEnvironments}${resourceToken}'
    containerRegistryName: !empty(containerRegistryName) ? containerRegistryName : '${abbrs.containerRegistryRegistries}${resourceToken}'
    location: location
    // Add subnet configuration
    subnetResourceId: vnet.outputs.infrastructureSubnetId
    loadBalancerType: 'External' // Can be changed to 'Internal' if needed
  }
}

module dts './app/dts.bicep' = {
  scope: rg
  name: 'dtsResource'
  params: {
    name: !empty(dtsName) ? dtsName : '${abbrs.dts}${resourceToken}'
    taskhubname: !empty(taskHubName) ? taskHubName : '${abbrs.taskhub}${resourceToken}'
    location: dtsLocation
    tags: tags
    ipAllowlist: [
      '0.0.0.0/0'
    ]
    skuName: dtsSkuName
    skuCapacity: dtsCapacity
  }
}


// Container app
module client 'app/app.bicep' = {
  name: clientsServiceName
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
    exists: false
    identityName: identity.outputs.name
    dtsEndpoint: dts.outputs.dts_URL
    taskHubName: dts.outputs.TASKHUB_NAME
  }
}

// Container app
module worker 'app/app.bicep' = {
  name: workerServiceName
  scope: rg
  params: {
    appName: !empty(containerAppsAppName) ? '${containerAppsAppName}-worker' : '${abbrs.appContainerApps}${resourceToken}-worker'
    containerAppsEnvironmentName: containerAppsEnv.outputs.environmentName
    containerRegistryName: containerAppsEnv.outputs.registryName
    userAssignedManagedIdentity: {
      resourceId: identity.outputs.resourceId
      clientId: identity.outputs.clientId
    }
    location: location
    tags: tags
    serviceName: 'worker'
    exists: false
    identityName: identity.outputs.name
    dtsEndpoint: dts.outputs.dts_URL
    taskHubName: dts.outputs.TASKHUB_NAME
  }
}

// Add outputs from the deployment here, if needed.
//
// This allows the outputs to be referenced by other bicep deployments in the deployment pipeline,
// or by the local machine as a way to reference created resources in Azure for local development.
// Secrets should not be added here.
//
// Outputs are automatically saved in the local azd environment .env file.
// To see these outputs, run `azd env get-values`,  or `azd env get-values --output json` for json output.
output AZURE_LOCATION string = location
output AZURE_TENANT_ID string = tenant().tenantId
// Container outputs
output AZURE_CONTAINER_REGISTRY_ENDPOINT string = containerAppsEnv.outputs.registryLoginServer
output AZURE_CONTAINER_REGISTRY_NAME string = containerAppsEnv.outputs.registryName

// // Application outputs
// output AZURE_CONTAINER_APP_ENDPOINT string = web.outputs.endpoint
// output AZURE_CONTAINER_ENVIRONMENT_NAME string = web.outputs.envName

// Identity outputs
output AZURE_USER_ASSIGNED_IDENTITY_NAME string = identity.outputs.name
