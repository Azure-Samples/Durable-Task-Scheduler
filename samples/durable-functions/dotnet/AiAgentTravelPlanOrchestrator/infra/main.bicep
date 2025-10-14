targetScope = 'subscription'
@minLength(1)
@maxLength(64)
@description('Name of the the environment which is used to generate a short unique hash used in all resources.')
param environmentName string

@minLength(1)
@description('Primary location for all resources')
@allowed(['westus2', 'eastus', 'eastus2', 'westus'])
@metadata({
  azd: {
    type: 'location'
  }
})
param location string

@description('Skip the creation of the virtual network and private endpoint')
param skipVnet bool = true

@description('Name of the API service')
param apiServiceName string = ''

@description('Name of the user assigned identity')
param apiUserAssignedIdentityName string = ''

@description('Name of the application insights resource')
param applicationInsightsName string = ''

@description('Name of the app service plan')
param appServicePlanName string = ''

@description('Name of the log analytics workspace')
param logAnalyticsName string = ''

@description('Name of the resource group')
param resourceGroupName string = ''

@description('Name of the storage account')
param storageAccountName string = ''

@description('Name of the virtual network')
param vNetName string = ''

@description('Disable local authentication for Azure Monitor')
param disableLocalAuth bool = true

@description('Id of the user or app to assign application roles')
param principalId string = ''

@description('Name for the AI resource and used to derive name of dependent resources.')
param aiHubName string = 'hub-demo'

@description('Friendly name for your Hub resource')
param aiHubFriendlyName string = 'Agents Hub resource'

@description('Description of your Azure AI resource displayed in AI studio')
param aiHubDescription string = 'This is an example AI resource for use in Azure AI Studio.'

@description('Name for the AI project resources.')
param aiProjectName string = 'project-demo'

@description('Friendly name for your Azure AI resource')
param aiProjectFriendlyName string = 'Agents Project resource'

@description('Description of your Azure AI resource displayed in AI studio')
param aiProjectDescription string = 'This is an example AI Project resource for use in Azure AI Studio.'

@description('Name of the Azure AI Search account')
param aiSearchName string = 'agentaisearch'

@description('Name for capabilityHost.')
param capabilityHostName string = 'caphost1'

@description('Name of the Azure AI Services account')
param aiServicesName string = 'agentaiservices'

@description('Model name for deployment')
param modelName string = 'gpt-4o-mini'

@description('Model format for deployment')
param modelFormat string = 'OpenAI'

@description('Model version for deployment')
param modelVersion string = '2024-07-18'

@description('Model deployment SKU name')
param modelSkuName string = 'Standard'

@description('Model deployment capacity')
param modelCapacity int = 10

@description('Model deployment location. If you want to deploy an Azure AI resource/model in different location than the rest of the resources created.')
param modelLocation string = location

@description('The AI Service Account full ARM Resource ID. This is an optional field, and if not provided, the resource will be created.')
param aiServiceAccountResourceId string = ''

@description('The Ai Search Service full ARM Resource ID. This is an optional field, and if not provided, the resource will be created.')
param aiSearchServiceResourceId string = ''

@description('The Ai Storage Account full ARM Resource ID. This is an optional field, and if not provided, the resource will be created.')
param aiStorageAccountResourceId string = ''

// Variables
var abbrs = loadJsonContent('./abbreviations.json')
var resourceToken = toLower(uniqueString(subscription().id, rg.id, environmentName, location))
var tags = { 'azd-env-name': environmentName }
var functionAppName = !empty(apiServiceName) ? apiServiceName : '${abbrs.webSitesFunctions}api-${resourceToken}'
var deploymentStorageContainerName = 'app-package-${take(functionAppName, 32)}-${take(toLower(uniqueString(functionAppName, resourceToken)), 7)}'
var name = toLower('${aiHubName}')
var projectName = toLower('${aiProjectName}')
param dtsSkuName string = 'Dedicated'
param dtsCapacity int = 1
param dtsName string = ''
param taskHubName string = ''
// Define the web app name first so we can construct the URL
var webAppName = !empty(webServiceName) ? webServiceName : '${abbrs.webStaticSites}web-${resourceToken}'
// Pre-compute the expected web URI for CORS settings
var webUri = 'https://${webAppName}.azurestaticapps.net'
param webServiceName string = ''

// Create a short, unique suffix, that will be unique to each resource group
var uniqueSuffix = toLower(uniqueString(subscription().id, rg.id, location))

// Organize resources in a resource group
resource rg 'Microsoft.Resources/resourceGroups@2025-04-01' = {
  name: !empty(resourceGroupName) ? resourceGroupName : '${abbrs.resourcesResourceGroups}${environmentName}'
  location: location
  tags: tags
}

// The application frontend webapp
module webapp './app/staticwebapp.bicep' = {
  name: 'webapp-${resourceToken}'
  scope: rg
  params: {
    name: !empty(webAppName) ? webAppName : '${abbrs.webStaticSites}web-${resourceToken}'
    location: 'westus2'
    tags: union(tags, { 'azd-service-name': 'web' })
    backendResourceId: api.outputs.Service_API_ID
    userAssignedIdentityId: apiUserAssignedIdentity.outputs.identityId
  }
}

// User assigned managed identity to be used by the function app to reach storage and service bus
module apiUserAssignedIdentity './core/identity/user-assigned-identity.bicep' = {
  name: 'apiUserAssignedIdentity-${resourceToken}'
  scope: rg
  params: {
    location: location
    tags: tags
    identityName: !empty(apiUserAssignedIdentityName) ? apiUserAssignedIdentityName : '${abbrs.managedIdentityUserAssignedIdentities}api-${resourceToken}'
  }
}

//  Backing storage for Azure functions backend processor
module storage 'core/storage/storage-account.bicep' = {
  scope: rg
  name: 'storage-${resourceToken}'
  params: {
    name: !empty(storageAccountName) ? storageAccountName : '${abbrs.storageStorageAccounts}${resourceToken}'
    location: location
    tags: tags
    containers: [
      {name: deploymentStorageContainerName}
     ]
     networkAcls: skipVnet ? {} : {
        defaultAction: 'Deny'
      }
  }
}

// The application backend is a function app
module appServicePlan './core/host/appservice-plan.bicep' = {
  name: 'appserviceplan-${resourceToken}'
  scope: rg
  params: {
    name: !empty(appServicePlanName) ? appServicePlanName : '${abbrs.webServerFarms}${resourceToken}'
    location: 'westus'
    tags: tags
    sku: {
      tier: 'Basic'
      name: 'B1'
    }
  }
}

module api './app/api.bicep' = {
  name: 'api-${resourceToken}'
  scope: rg
  params: {
    name: functionAppName
    location: location
    tags: union(tags, { 'azd-service-name': 'api' })
    serviceName: 'api'
    applicationInsightsName: monitoring.outputs.applicationInsightsName
    appServicePlanId: appServicePlan.outputs.id
    storageAccountName: storage.outputs.name
    identityId: apiUserAssignedIdentity.outputs.identityId
    identityClientId: apiUserAssignedIdentity.outputs.identityClientId
    allowedOrigins: [ webUri ]
    appSettings: {
      // https://learn.microsoft.com/en-us/azure/azure-functions/durable/durable-functions-configure-managed-identity
      // Storage account connection using managed identity
      AzureWebJobsStorage__blobServiceUri: 'https://${storage.outputs.name}.blob.core.windows.net'
      AzureWebJobsStorage__queueServiceUri: 'https://${storage.outputs.name}.queue.core.windows.net'
      AzureWebJobsStorage__tableServiceUri: 'https://${storage.outputs.name}.blob.core.windows.net'
      // Note: AZURE_TENANT_ID and AZURE_CLIENT_ID are NOT needed - DefaultAzureCredential automatically discovers the managed identity
      DURABLE_TASK_SCHEDULER_CONNECTION_STRING: 'Endpoint=${dts.outputs.dts_URL};Authentication=ManagedIdentity;ClientID=${apiUserAssignedIdentity.outputs.identityClientId}'
      TASKHUB_NAME: dts.outputs.TASKHUB_NAME
      // Azure OpenAI configuration for the application
      AZURE_OPENAI_ENDPOINT: aiDependencies.outputs.aiservicesTarget
      AZURE_OPENAI_DEPLOYMENT_NAME: modelName
    }
    virtualNetworkSubnetId: skipVnet ? '' : serviceVirtualNetwork.outputs.appSubnetID
  }
  dependsOn: [
    storage
    appServicePlan
    apiUserAssignedIdentity
  ]
}

// Dependent resources for the Azure Machine Learning workspace
module aiDependencies './agent/standard-dependent-resources.bicep' = {
  scope: rg
  name: 'dependencies-${name}-${resourceToken}'
  params: {
    location: modelLocation  // Use modelLocation for AI resources
    storageName: 'st${resourceToken}'
    keyvaultName: 'kv-${name}${resourceToken}'
    aiServicesName: '${aiServicesName}${resourceToken}'
    aiSearchName: '${aiSearchName}${resourceToken}'
    tags: tags

    // Model deployment parameters
    modelName: modelName
    modelFormat: modelFormat
    modelVersion: modelVersion
    modelSkuName: modelSkuName
    modelCapacity: modelCapacity  
    modelLocation: modelLocation

    aiServiceAccountResourceId: aiServiceAccountResourceId
    aiSearchServiceResourceId: aiSearchServiceResourceId
    aiStorageAccountResourceId: aiStorageAccountResourceId
    }

  dependsOn: [
    storage
  ]
}

module aiHub './agent/standard-ai-hub.bicep' = {
  scope: rg
  name: '${name}-${resourceToken}'
  params: {
    // workspace organization
    aiHubName: '${name}${uniqueSuffix}'
    aiHubFriendlyName: aiHubFriendlyName
    aiHubDescription: aiHubDescription
    location: modelLocation  // Use modelLocation for AI resources
    tags: tags
    capabilityHostName: '${name}${uniqueSuffix}${capabilityHostName}'

    aiSearchName: aiDependencies.outputs.aiSearchName
    aiSearchId: aiDependencies.outputs.aisearchID

    aiServicesName: aiDependencies.outputs.aiServicesName
    aiServicesId: aiDependencies.outputs.aiservicesID
    aiServicesTarget: aiDependencies.outputs.aiservicesTarget
    
    keyVaultId: aiDependencies.outputs.keyvaultId
    storageAccountId: aiDependencies.outputs.storageId
  }
  dependsOn: [
    storage
    aiDependencies
  ]
}

module aiProject './agent/standard-ai-project.bicep' = {
  scope: rg
  name: '${projectName}-${resourceToken}-deployment'
  params: {
    // workspace organization
    aiProjectName: '${projectName}${uniqueSuffix}'
    aiProjectFriendlyName: aiProjectFriendlyName
    aiProjectDescription: aiProjectDescription
    location: location
    tags: tags
    
    // dependent resources
    capabilityHostName: '${projectName}${uniqueSuffix}${capabilityHostName}'

    aiHubId: aiHub.outputs.aiHubID
    acsConnectionName: aiHub.outputs.acsConnectionName
    aoaiConnectionName: aiHub.outputs.aoaiConnectionName
  }
  dependsOn:[
    storage
  ]
}

module aiServiceRoleAssignments './agent/ai-service-role-assignments.bicep' = {
  scope: rg
  name: 'aiserviceroleassignments${projectName}-${resourceToken}'
  params: {
    aiServicesName: aiDependencies.outputs.aiServicesName
    aiProjectPrincipalId: aiProject.outputs.aiProjectPrincipalId
    aiProjectId: aiProject.outputs.aiProjectResourceId
  }
}

module aiSearchRoleAssignments './agent/ai-search-role-assignments.bicep' = {
  scope: rg
  name: 'aisearchroleassignments${projectName}-${resourceToken}'
  params: {
    aiSearchName: aiDependencies.outputs.aiSearchName
    aiProjectPrincipalId: aiProject.outputs.aiProjectPrincipalId
    aiProjectId: aiProject.outputs.aiProjectResourceId
  }
}

// Allow the Function App's managed identity to access AI Services
module aiServiceRoleAssignmentFunctionApp './app/ai-service-access.bicep' = {
  scope: rg
  name: 'aiServiceRoleAssignmentFunctionApp-${resourceToken}'
  params: {
    aiServicesName: aiDependencies.outputs.aiServicesName
    principalId: apiUserAssignedIdentity.outputs.identityPrincipalId
    principalType: 'ServicePrincipal'
  }
  dependsOn: [
    aiDependencies
    apiUserAssignedIdentity
  ]
}

var storageBlobDataContributorRole  = 'ba92f5b4-2d11-453d-a403-e96b0029c9fe' // Storage Blob Data Owner role

// Allow access from api to storage account using a managed identity
module storageRoleAssignmentApi 'app/storage-Access.bicep' = {
  scope: rg
  name: 'storageRoleAssignmentApi-${resourceToken}'
  params: {
    storageAccountName: storage.outputs.name
    roleDefinitionID: storageBlobDataContributorRole
    principalID: apiUserAssignedIdentity.outputs.identityPrincipalId
    principalType: 'ServicePrincipal'
  }
}

// Allow access from user to storage account
module storageRoleAssignmentUser 'app/storage-Access.bicep' = {
  scope: rg
  name: 'storageRoleAssignmentUser-${resourceToken}'
  params: {
    storageAccountName: storage.outputs.name
    roleDefinitionID: storageBlobDataContributorRole
    principalID: principalId
    principalType: 'User'
  }
}

var storageQueueDataContributorRoleDefinitionId  = '974c5e8b-45b9-4653-ba55-5f855dd0fb88' // Storage Queue Data Contributor

module storageQueueDataContributorRoleAssignmentprocessor 'app/storage-Access.bicep' = {
  scope: rg
  name: 'storageQueueDataContributorRoleAssignmentprocessor-${resourceToken}'
  params: {
    storageAccountName: storage.outputs.name
    roleDefinitionID: storageQueueDataContributorRoleDefinitionId
    principalID: apiUserAssignedIdentity.outputs.identityPrincipalId
    principalType: 'ServicePrincipal'
  }
}

// Allow access from AI project to storage account using a managed identity
module storageQueueDataContributorRoleAssignmentAIProject 'app/storage-Access.bicep' = {
  scope: rg
  name: 'storageQueueDataContributorRoleAssignmentAIProject-${resourceToken}'
  params: {
    storageAccountName: storage.outputs.name
    roleDefinitionID: storageQueueDataContributorRoleDefinitionId
    principalID: aiProject.outputs.aiProjectPrincipalId
    principalType: 'ServicePrincipal'
  }
}

module storageQueueDataContributorRoleAssignmentUserIdentityprocessor 'app/storage-Access.bicep' = {
  scope: rg
  name: 'storageQueueDataContributorRole-${resourceToken}'
  params: {
    storageAccountName: storage.outputs.name
    roleDefinitionID: storageQueueDataContributorRoleDefinitionId
    principalID: principalId
    principalType: 'User'
  }
}

var storageTableDataContributorRoleDefinitionId  = '0a9a7e1f-b9d0-4cc4-a60d-0319b160aaa3' // Storage Table Data Contributor

module storageTableDataContributorRoleAssignmentprocessor 'app/storage-Access.bicep' = {
  scope: rg
  name: 'storageTableDataContributorRole-${resourceToken}'
  params: {
    storageAccountName: storage.outputs.name
    roleDefinitionID: storageTableDataContributorRoleDefinitionId
    principalID: apiUserAssignedIdentity.outputs.identityPrincipalId
    principalType: 'ServicePrincipal'
  }
}

// Virtual Network & private endpoint to blob storage
module serviceVirtualNetwork 'app/vnet.bicep' =  if (!skipVnet) {
  scope: rg
  name: 'serviceVirtualNetwork-${resourceToken}'
  params: {
    location: location
    tags: tags
    vNetName: !empty(vNetName) ? vNetName : '${abbrs.networkVirtualNetworks}${resourceToken}'
  }
}

module storagePrivateEndpoint 'app/storage-private-endpoint.bicep' = if (!skipVnet) {
  scope: rg
  name: 'servicePrivateEndpoint-${resourceToken}'
  params: {
    location: location
    tags: tags
    virtualNetworkName: !empty(vNetName) ? vNetName : '${abbrs.networkVirtualNetworks}${resourceToken}'
    subnetName: skipVnet ? '' : serviceVirtualNetwork.outputs.peSubnetName
    resourceName: storage.outputs.name
  }
}

// Monitor application with Azure Monitor
module monitoring './core/monitor/monitoring.bicep' = {
  scope: rg
  name: 'monitoring-${resourceToken}'
  params: {
    location: location
    tags: tags
    logAnalyticsName: !empty(logAnalyticsName) ? logAnalyticsName : '${abbrs.operationalInsightsWorkspaces}${resourceToken}'
    applicationInsightsName: !empty(applicationInsightsName) ? applicationInsightsName : '${abbrs.insightsComponents}${resourceToken}'
    disableLocalAuth: disableLocalAuth  
  }
}

var monitoringRoleDefinitionId = '3913510d-42f4-4e42-8a64-420c390055eb' // Monitoring Metrics Publisher role ID

// Allow access from api to application insights using a managed identity
module appInsightsRoleAssignmentApi './core/monitor/appinsights-access.bicep' = {
  scope: rg
  name: 'appInsightsRoleAssignmentApi-${resourceToken}'
  params: {
    appInsightsName: monitoring.outputs.applicationInsightsName
    roleDefinitionID: monitoringRoleDefinitionId
    principalID: apiUserAssignedIdentity.outputs.identityPrincipalId
  }
  dependsOn: [
    apiUserAssignedIdentity
  ]
}

var azureAiDeveloperRoleId = '64702f94-c441-49e6-a78b-ef80e0188fee' // Azure AI Developer role ID
// Enable access to AI Project from the Azure Function user assigned identity
resource AIProjectRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(azureAiDeveloperRoleId, aiProjectName, resourceId('Microsoft.MachineLearningServices/workspaces', aiProjectName), resourceToken)
  properties: {
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', azureAiDeveloperRoleId)
    principalId: apiUserAssignedIdentity.outputs.identityPrincipalId
    principalType: 'ServicePrincipal'
  }
  dependsOn: [
    apiUserAssignedIdentity
  ]
}

var durableTaskDataContributorRoleDefinitionId = '0ad04412-c4d5-4796-b79c-f76d14c8d402'

// Allow access from durable function to storage account using a user assigned managed identity
module dtsRoleAssignment 'app/dts-access.bicep' = {
  name: 'dtsRoleAssignment-${resourceToken}'
  scope: rg
  params: {
   roleDefinitionID: durableTaskDataContributorRoleDefinitionId
   principalID: apiUserAssignedIdentity.outputs.identityPrincipalId
   principalType: 'ServicePrincipal'
   dtsName: dts.outputs.dts_NAME
  }
  dependsOn: [
    storage
    apiUserAssignedIdentity
    dts
  ]
}

module dtsDashboardRoleAssignment 'app/dts-access.bicep' = {
  name: 'dtsDashboardRoleAssignment-${resourceToken}'
  scope: rg
  params: {
   roleDefinitionID: durableTaskDataContributorRoleDefinitionId
   principalID: principalId
   principalType: 'User'
   dtsName: dts.outputs.dts_NAME
  }
  dependsOn: [
    dts
  ]
}

module dts './app/dts.bicep' = {
  scope: rg 
  name: 'dtsResource-${resourceToken}'
  params: {
    name: !empty(dtsName) ? dtsName : '${abbrs.dts}${resourceToken}'
    taskhubname: !empty(taskHubName) ? taskHubName : '${abbrs.taskhub}${resourceToken}'
    location: location
    tags: tags
    ipAllowlist: [
      '0.0.0.0/0'
    ]
    skuName: dtsSkuName
    skuCapacity: dtsCapacity
  }
  dependsOn: [
    storage
  ]
}

// App outputs
output APPLICATIONINSIGHTS_CONNECTION_STRING string = monitoring.outputs.applicationInsightsConnectionString
output AZURE_LOCATION string = location
// Note: AZURE_TENANT_ID and AZURE_CLIENT_ID outputs removed - not needed with DefaultAzureCredential
output SERVICE_API_NAME string = api.outputs.SERVICE_API_NAME
output SERVICE_API_URI string = api.outputs.SERVICE_API_URI
output AZURE_FUNCTION_APP_NAME string = api.outputs.SERVICE_API_NAME
output STATIC_WEB_APP_NAME string = webapp.outputs.name  
output STATIC_WEB_APP_URI string = webapp.outputs.uri
output PRE_STATIC_WEB_APP_URI string = webAppName
output RESOURCE_GROUP string = rg.name
output PROJECT_CONNECTION_STRING string = aiProject.outputs.projectConnectionString
output STORAGE_CONNECTION__queueServiceUri string = 'https://${storage.outputs.name}.queue.${environment().suffixes.storage}'
output AZURE_OPENAI_ENDPOINT string = aiDependencies.outputs.aiservicesTarget
output AZURE_OPENAI_DEPLOYMENT_NAME string = modelName
