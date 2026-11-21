targetScope = 'subscription'
@minLength(1)
@maxLength(64)
@description('Name of the the environment which is used to generate a short unique hash used in all resources.')
param environmentName string

@minLength(1)
@description('Primary location for all resources. Must support both Durable Task Scheduler and the agent model deployment.')
@allowed(['eastus2'])
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

@description('Name of the Microsoft Foundry resource (AIServices account). Leave empty to generate a unique name.')
param aiFoundryName string = ''

@description('Name of the Microsoft Foundry project.')
param aiProjectName string = 'travel-planner'

@description('Display name of the Microsoft Foundry project')
param aiProjectFriendlyName string = 'Travel Planner Agents'

@description('Description of the Microsoft Foundry project')
param aiProjectDescription string = 'Foundry project hosting the travel planner agents.'

@description('Whether the Microsoft Foundry resource accepts traffic from public networks')
@allowed(['Enabled', 'Disabled'])
param aiPublicNetworkAccess string = 'Enabled'

@description('Model name for deployment')
param modelName string = 'gpt-5-mini'

@description('Model format for deployment')
param modelFormat string = 'OpenAI'

@description('Model version for deployment')
param modelVersion string = '2025-08-07'

@description('Model deployment SKU name')
param modelSkuName string = 'GlobalStandard'

@description('Model deployment capacity')
param modelCapacity int = 50

@description('Location of the Microsoft Foundry resource and model deployment, if it should differ from the rest of the resources.')
param modelLocation string = location

@description('The agent ID of the destination recommender agent')
param destinationRecommenderAgentId string = 'agent-destination-recommender-agent'

@description('The agent ID of the itinerary planner agent')
param itineraryPlannerAgentId string = 'agent-itinerary-planner-agent'

@description('The agent ID of the local recommendations agent')
param localRecommendationsAgentId string = 'agent-local-recommendations-agent'

// Variables
var abbrs = loadJsonContent('./abbreviations.json')
var resourceToken = toLower(uniqueString(subscription().id, rg.id, environmentName, location))
var tags = { 'azd-env-name': environmentName }
var functionAppName = !empty(apiServiceName) ? apiServiceName : '${abbrs.webSitesFunctions}api-${resourceToken}'
var deploymentStorageContainerName = 'app-package-${take(functionAppName, 32)}-${take(toLower(uniqueString(functionAppName, resourceToken)), 7)}'
var foundryName = !empty(aiFoundryName) ? aiFoundryName : 'aif-${resourceToken}'
param dtsSkuName string = 'Consumption'
param dtsCapacity int = 1
param dtsName string = ''
param taskHubName string = ''
// Define the web app name first so we can construct the URL
var webAppName = !empty(webServiceName) ? webServiceName : '${abbrs.webStaticSites}web-${resourceToken}'
// Pre-compute the expected web URI for CORS settings
var webUri = 'https://${webAppName}.azurestaticapps.net'
param webServiceName string = ''

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
    location: location
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
    location: location
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
      // These would be needed if running outside of cloud
      AzureWebJobsStorage__blobServiceUri: storage.outputs.primaryEndpoints.blob
      AzureWebJobsStorage__queueServiceUri: storage.outputs.primaryEndpoints.queue
      AzureWebJobsStorage__tableServiceUri: storage.outputs.primaryEndpoints.table
      AZURE_TENANT_ID: tenant().tenantId
      AZURE_CLIENT_ID: apiUserAssignedIdentity.outputs.identityClientId
      DURABLE_TASK_SCHEDULER_CONNECTION_STRING: 'Endpoint=${dts.outputs.dts_URL};Authentication=ManagedIdentity;ClientID=${apiUserAssignedIdentity.outputs.identityClientId}'
      TASKHUB_NAME: dts.outputs.TASKHUB_NAME
      DESTINATION_RECOMMENDER_CONNECTION: aiFoundry.outputs.projectEndpoint
      ITINERARY_PLANNER_CONNECTION: aiFoundry.outputs.projectEndpoint
      LOCAL_RECOMMENDATIONS_CONNECTION: aiFoundry.outputs.projectEndpoint
      DESTINATION_RECOMMENDER_AGENT_ID: destinationRecommenderAgentId
      ITINERARY_PLANNER_AGENT_ID: itineraryPlannerAgentId
      LOCAL_RECOMMENDATIONS_AGENT_ID: localRecommendationsAgentId
    }
    virtualNetworkSubnetId: skipVnet ? '' : serviceVirtualNetwork!.outputs.appSubnetID
  }
}

// Microsoft Foundry resource, project, and model deployment used by the agents
module aiFoundry './agent/ai-foundry.bicep' = {
  scope: rg
  name: 'aiFoundry-${resourceToken}'
  params: {
    location: modelLocation
    tags: tags
    aiFoundryName: foundryName
    aiProjectName: toLower(aiProjectName)
    aiProjectFriendlyName: aiProjectFriendlyName
    aiProjectDescription: aiProjectDescription
    modelName: modelName
    modelFormat: modelFormat
    modelVersion: modelVersion
    modelSkuName: modelSkuName
    modelCapacity: modelCapacity
    publicNetworkAccess: aiPublicNetworkAccess
  }
}

var azureAiUserRoleId = '53ca6127-db72-4b80-b1b0-d745d6d5456d' // Foundry User (formerly Azure AI User)

// Allow the function app identity to call the Foundry Agent Service
module aiFoundryRoleAssignmentApi './agent/ai-foundry-access.bicep' = {
  scope: rg
  name: 'aiFoundryRoleAssignmentApi-${resourceToken}'
  params: {
    aiFoundryName: aiFoundry.outputs.aiFoundryName
    roleDefinitionID: azureAiUserRoleId
    principalID: apiUserAssignedIdentity.outputs.identityPrincipalId
    principalType: 'ServicePrincipal'
  }
}

// Allow the deploying user to create and test agents in the Foundry project
module aiFoundryRoleAssignmentUser './agent/ai-foundry-access.bicep' = if (!empty(principalId)) {
  scope: rg
  name: 'aiFoundryRoleAssignmentUser-${resourceToken}'
  params: {
    aiFoundryName: aiFoundry.outputs.aiFoundryName
    roleDefinitionID: azureAiUserRoleId
    principalID: principalId
    principalType: 'User'
  }
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
    subnetName: skipVnet ? '' : serviceVirtualNetwork!.outputs.peSubnetName
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
}

var durableTaskDataContributorRoleDefinitionId = '0ad04412-c4d5-4796-b79c-f76d14c8d402'

// Allow access from durable function to storage account using a user assigned managed identity
module dtsRoleAssignment 'app/dts-Access.bicep' = {
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
  ]
}

module dtsDashboardRoleAssignment 'app/dts-Access.bicep' = {
  name: 'dtsDashboardRoleAssignment-${resourceToken}'
  scope: rg
  params: {
   roleDefinitionID: durableTaskDataContributorRoleDefinitionId
   principalID: principalId
   principalType: 'User'
   dtsName: dts.outputs.dts_NAME
  }
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
output AZURE_TENANT_ID string = tenant().tenantId
output AZURE_CLIENT_ID string = apiUserAssignedIdentity.outputs.identityClientId
output SERVICE_API_NAME string = api.outputs.SERVICE_API_NAME
output SERVICE_API_URI string = api.outputs.SERVICE_API_URI
output AZURE_FUNCTION_APP_NAME string = api.outputs.SERVICE_API_NAME
output STATIC_WEB_APP_NAME string = webapp.outputs.name  
output STATIC_WEB_APP_URI string = webapp.outputs.uri
output PRE_STATIC_WEB_APP_URI string = webAppName
output RESOURCE_GROUP string = rg.name
output AI_FOUNDRY_NAME string = aiFoundry.outputs.aiFoundryName
output AI_PROJECT_NAME string = aiFoundry.outputs.aiProjectName
output AI_MODEL_DEPLOYMENT_NAME string = aiFoundry.outputs.modelDeploymentName
output PROJECT_ENDPOINT string = aiFoundry.outputs.projectEndpoint
output STORAGE_CONNECTION__queueServiceUri string = 'https://${storage.outputs.name}.queue.${environment().suffixes.storage}'

// Agent outputs
output DESTINATION_RECOMMENDER_AGENT_ID string = destinationRecommenderAgentId
output ITINERARY_PLANNER_AGENT_ID string = itineraryPlannerAgentId
output LOCAL_RECOMMENDATIONS_AGENT_ID string = localRecommendationsAgentId
output AGENT_CONNECTION_STRING string = aiFoundry.outputs.projectEndpoint
