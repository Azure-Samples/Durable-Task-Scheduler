targetScope = 'subscription'

@minLength(1)
@maxLength(64)
@description('Name of the the environment which is used to generate a short unique hash used in all resources.')
param environmentName string

@minLength(1)
@description('Primary location for all resources')
@allowed(['australiaeast', 'eastasia', 'northcentralus', 'northeurope', 'southeastasia', 'swedencentral', 'uksouth', 'westus2', 'centralus'])
@metadata({
  azd: {
    type: 'location'
  }
})
param location string

@allowed([
  'EP1'
  'EP2'
  'EP3'
  'P1v2'
  'P2v2'
  'P3v2'
  'P1v3'
  'P2v3'
  'P3v3'
  'P1mv3'
  'P2mv3'
  'P3mv3'
  'P4mv3'
  'P5mv3'
  'P0v3'
  'S1'
  'B1'
  'B2'
  'B3'
])
param functionSkuName string = 'EP1' // Uses main.parameters.json first

@allowed([
  'ElasticPremium'
  'PremiumV3'
  'Premium0V3'
  'Standard'
  'Basic'
])

param functionSkuTier string = 'ElasticPremium' // Uses main.parameters.json first
param functionReservedPlan bool = true // Set to false to get a Windows OS plan

param dtsLocation string = location
param dtsSkuName string = 'Consumption'
param dtsCapacity int = 1

param durableFunctionServiceName string = ''
param durableFunctionUserAssignedIdentityName string = ''
param applicationInsightsName string = ''
param appServicePlanName string = ''
param logAnalyticsName string = ''
param resourceGroupName string = ''
param storageAccountName string = ''
param dtsName string = ''
param taskHubName string = ''
param vNetName string = ''
param disableLocalAuth bool = true

var abbrs = loadJsonContent('./abbreviations.json')
var resourceToken = toLower(uniqueString(subscription().id, environmentName, location))
var tags = { 'azd-env-name': environmentName }
var functionAppName = !empty(durableFunctionServiceName) ? durableFunctionServiceName : '${abbrs.webSitesFunctions}${resourceToken}'
var deploymentStorageContainerName = 'app-package-${take(functionAppName, 32)}-${take(toLower(uniqueString(functionAppName, resourceToken)), 7)}'
var desktopIpAddress = ''  //keep empty to set later, or set now

@description('Id of the user or app to assign application roles')
param principalId string = ''

// Organize resources in a resource group
resource rg 'Microsoft.Resources/resourceGroups@2021-04-01' = {
  name: !empty(resourceGroupName) ? resourceGroupName : '${abbrs.resourcesResourceGroups}${environmentName}'
  location: location
  tags: tags
}

module durableFunctionUserAssignedIdentity './core/identity/userAssignedIdentity.bicep' = {
  name: 'DurableFunctionUserAssignedIdentity'
  scope: rg
  params: {
    location: location
    tags: tags
    identityName: !empty(durableFunctionUserAssignedIdentityName) ? durableFunctionUserAssignedIdentityName : '${abbrs.managedIdentityUserAssignedIdentities}durable-function-${resourceToken}'
  }
}

// The application backend is a function app
module appServicePlan './core/host/appserviceplan.bicep' = {
  name: 'appserviceplan'
  scope: rg
  params: {
    name: !empty(appServicePlanName) ? appServicePlanName : '${abbrs.webServerFarms}${resourceToken}'
    location: location
    tags: tags
    sku: {
      name: functionSkuName // Change this to the desired Elastic Premium SKU
      tier: functionSkuTier
    }
    reserved: functionReservedPlan // Set to false to get a Windows OS plan
  }
}

module durableFunction './app/durable-function.bicep' = {
  name: 'order-processor'
  scope: rg
  params: {
    name: functionAppName
    location: location
    tags: tags
    applicationInsightsName: monitoring.outputs.applicationInsightsName
    appServicePlanId: appServicePlan.outputs.id
    runtimeName: 'dotnet-isolated'
    runtimeVersion: '8.0'
    storageAccountName: storage.outputs.name
    identityId: durableFunctionUserAssignedIdentity.outputs.identityId
    identityClientId: durableFunctionUserAssignedIdentity.outputs.identityClientId
    deploymentStorageContainerName: deploymentStorageContainerName
    dtsURL: dts.outputs.dts_URL
    taskHubName: dts.outputs.TASKHUB_NAME
    appSettings: {
    }
    virtualNetworkSubnetId: serviceVirtualNetwork.outputs.appSubnetID
  }
}

// Backing storage for Azure functions durable function
module storage './core/storage/storage-account.bicep' = {
  name: 'storage'
  scope: rg
  params: {
    name: !empty(storageAccountName) ? storageAccountName : '${abbrs.storageStorageAccounts}${resourceToken}'
    location: location
    tags: tags
    containers: [{
      name: deploymentStorageContainerName
      publicAccess: 'None'
    },{
      name: 'input'
      publicAccess: 'None'

    },{
      name: 'output'
      publicAccess: 'None'
    }]
    publicNetworkAccess: 'Enabled' // Pinning this to enabled, and we use defaultAction Deny or Allow
    networkAcls: {
      defaultAction: 'Deny'
      bypass: 'None'
      // Conditionally add the desktop IP to ipRules
      ipRules: !empty(desktopIpAddress) ? [
        {
          action: 'Allow'
          value: desktopIpAddress 
        }
      ] : [] // If desktopIpAddress is not provided, ipRules will be an empty array
    }
    allowBlobPublicAccess: false
  }
}

// Allow access from durable function to storage account using a user assigned managed identity
var blobDataOwnerRoleDefinitionId  = 'b7e6dc6d-f1e8-4753-8033-0f276bb0955b' //Storage Blob Data Owner role
var tableDataContributorDefinitionId = '0a9a7e1f-b9d0-4cc4-a60d-0319b160aaa3' // Storage Table Data Contributor role
var queueDataContributorDefinitionId = '974c5e8b-45b9-4653-ba55-5f855dd0fb88' // Storage Queue Data Contributor role
var blobDataContributorDefinitionId  = 'ba92f5b4-2d11-453d-a403-e96b0029c9fe' //Storage Blob Data Contributor role

module blobOwnerAssignmentUAMI 'app/storage-Access.bicep' = {
  name:'blobDataOwnerPocessorUAMI'
  scope: rg
  params: {
    storageAccountName: storage.outputs.name
    roleDefinitionID: blobDataOwnerRoleDefinitionId
    principalID: durableFunctionUserAssignedIdentity.outputs.identityPrincipalId
    principalType: 'ServicePrincipal'
  }
}

module tableDataContributorUAMI 'app/storage-Access.bicep' = {
  name: 'tableDataContributorPocessorUAMI'
  scope: rg
  params: {
    storageAccountName: storage.outputs.name
    roleDefinitionID: tableDataContributorDefinitionId
    principalID: durableFunctionUserAssignedIdentity.outputs.identityPrincipalId
    principalType: 'ServicePrincipal'
  }
}

module queueDataContributorUAMI 'app/storage-Access.bicep' = {
  name: 'queueDataContributorPocessorUAMI'
  scope: rg
  params: {
    storageAccountName: storage.outputs.name
    roleDefinitionID: queueDataContributorDefinitionId
    principalID: durableFunctionUserAssignedIdentity.outputs.identityPrincipalId
    principalType: 'ServicePrincipal'
  }
}

module blobDataContributorUAMI 'app/storage-Access.bicep' = {
  name: 'blobDataContributorPocessorUAMI'
  scope: rg
  params: {
    storageAccountName: storage.outputs.name
    roleDefinitionID: blobDataContributorDefinitionId
    principalID: durableFunctionUserAssignedIdentity.outputs.identityPrincipalId
    principalType: 'ServicePrincipal'
  }
}

// Allow access to storage account using the Login identity of this bicep (usually AZD CLI)
module storageRoleAssignmentApi 'app/storage-Access.bicep' = {
  name: 'blobDataOwnerLoginIdentity'
  scope: rg
  params: {
    storageAccountName: storage.outputs.name
    roleDefinitionID: blobDataOwnerRoleDefinitionId
    principalID: principalId
    principalType: 'User'
  }
}

// Virtual Network & private endpoint to blob storage
module serviceVirtualNetwork 'app/vnet.bicep' = {
  name: 'serviceVirtualNetwork'
  scope: rg
  params: {
    location: location
    tags: tags
    vNetName: !empty(vNetName) ? vNetName : '${abbrs.networkVirtualNetworks}${resourceToken}'
  }
}

module storagePrivateEndpoint 'app/storage-PrivateEndpoint.bicep' = {
  name: 'servicePrivateEndpoint'
  scope: rg
  params: {
    location: location
    tags: tags
    virtualNetworkName: !empty(vNetName) ? vNetName : '${abbrs.networkVirtualNetworks}${resourceToken}'
    subnetName: serviceVirtualNetwork.outputs.peSubnetName
    resourceName: storage.outputs.name
  }
}

// Monitor application with Azure Monitor
module monitoring './core/monitor/monitoring.bicep' = {
  name: 'monitoring'
  scope: rg
  params: {
    location: location
    tags: tags
    logAnalyticsName: !empty(logAnalyticsName) ? logAnalyticsName : '${abbrs.operationalInsightsWorkspaces}${resourceToken}'
    applicationInsightsName: !empty(applicationInsightsName) ? applicationInsightsName : '${abbrs.insightsComponents}${resourceToken}'
    disableLocalAuth: disableLocalAuth  
  }
}

var monitoringRoleDefinitionId = '3913510d-42f4-4e42-8a64-420c390055eb' // Monitoring Metrics Publisher role ID

// Allow access from durable function` to application insights using a managed identity
module appInsightsRoleAssignmentApi './core/monitor/appinsights-access.bicep' = {
  name: 'appInsightsRoleAssignmentDurableFunction'
  scope: rg
  params: {
    appInsightsName: monitoring.outputs.applicationInsightsName
    roleDefinitionID: monitoringRoleDefinitionId
    principalID: durableFunctionUserAssignedIdentity.outputs.identityPrincipalId
  }
}

// Allow access from durable function to storage account using a user assigned managed identity
module dtsRoleAssignment 'app/dts-Access.bicep' = {
  name: 'dtsRoleAssignment'
  scope: rg
  params: {
   roleDefinitionID: '0ad04412-c4d5-4796-b79c-f76d14c8d402'
   principalID: durableFunctionUserAssignedIdentity.outputs.identityPrincipalId
   principalType: 'ServicePrincipal'
   dtsName: dts.outputs.dts_NAME
  }
}

module dtsDashboardRoleAssignment 'app/dts-Access.bicep' = {
  name: 'dtsDashboardRoleAssignment'
  scope: rg
  params: {
   roleDefinitionID: '0ad04412-c4d5-4796-b79c-f76d14c8d402'
   principalID: principalId
   principalType: 'User'
   dtsName: dts.outputs.dts_NAME
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

// App outputs
output APPLICATIONINSIGHTS_CONNECTION_STRING string = monitoring.outputs.applicationInsightsConnectionString
output AZURE_LOCATION string = location
output AZURE_TENANT_ID string = tenant().tenantId
output SERVICE_DURABLE_FUNCTION_NAME string = durableFunction.outputs.SERVICE_DURABLE_FUNCTION_NAME
output AZURE_FUNCTION_NAME string = durableFunction.outputs.SERVICE_DURABLE_FUNCTION_NAME
output AZURE_RESOURCE_GROUP string = rg.name
output AZURE_STORAGE_ACCOUNT_NAME string = storage.outputs.name
output AZURE_STORAGE_CONTAINER_NAME string = deploymentStorageContainerName
