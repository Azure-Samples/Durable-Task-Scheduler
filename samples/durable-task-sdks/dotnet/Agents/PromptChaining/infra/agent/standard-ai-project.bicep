// Creates an Azure AI resource with proxied endpoints for the Azure AI services provider

@description('Azure region of the deployment')
param location string

@description('Tags to add to the resources')
param tags object

@description('AI Services Foundry account under which the project will be created')
param aiServicesAccountName string

@description('AI Project name')
param aiProjectName string

@description('AI Project display name')
param aiProjectFriendlyName string = aiProjectName

@description('AI Project description')
param aiProjectDescription string

@description('Cosmos DB Account for agent thread storage')
param cosmosDbAccountName string
param cosmosDbAccountSubscriptionId string
param cosmosDbAccountResourceGroupName string

@description('Storage Account for agent artifacts')
param storageAccountName string
param storageAccountSubscriptionId string
param storageAccountResourceGroupName string

@description('AI Search Service for vector store and search')
param aiSearchName string
param aiSearchSubscriptionId string
param aiSearchResourceGroupName string

resource cosmosDbAccount 'Microsoft.DocumentDB/databaseAccounts@2024-11-15' existing = {
  name: cosmosDbAccountName
  scope: resourceGroup(cosmosDbAccountSubscriptionId, cosmosDbAccountResourceGroupName)
}

resource storageAccount 'Microsoft.Storage/storageAccounts@2024-01-01' existing = {
  name: storageAccountName
  scope: resourceGroup(storageAccountSubscriptionId, storageAccountResourceGroupName)
}

resource aiSearchService 'Microsoft.Search/searchServices@2024-06-01-preview' existing = {
  name: aiSearchName
  scope: resourceGroup(aiSearchSubscriptionId, aiSearchResourceGroupName)
}

resource aiServicesAccount 'Microsoft.CognitiveServices/accounts@2025-04-01-preview' existing = {
  name: aiServicesAccountName
}

resource aiProject 'Microsoft.CognitiveServices/accounts/projects@2025-04-01-preview' = {
  parent: aiServicesAccount
  name: aiProjectName
  location: location
  tags: tags
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    description: aiProjectDescription
    displayName: aiProjectFriendlyName
  }

  resource project_connection_cosmosdb_account 'connections@2025-04-01-preview' = {
    name: cosmosDbAccountName
    properties: {
      category: 'CosmosDB'
      target: cosmosDbAccount.properties.documentEndpoint
      authType: 'AAD'
      metadata: {
        ApiType: 'Azure'
        ResourceId: cosmosDbAccount.id
        location: cosmosDbAccount.location
      }
    }
  }

  resource project_connection_azure_storage 'connections@2025-04-01-preview' = {
    name: storageAccountName
    properties: {
      category: 'AzureStorageAccount'
      target: storageAccount.properties.primaryEndpoints.blob
      authType: 'AAD'
      metadata: {
        ApiType: 'Azure'
        ResourceId: storageAccount.id
        location: storageAccount.location
      }
    }
  }

  resource project_connection_azureai_search 'connections@2025-04-01-preview' = {
    name: aiSearchName
    properties: {
      category: 'CognitiveSearch'
      target: 'https://${aiSearchName}.search.windows.net'
      authType: 'AAD'
      metadata: {
        ApiType: 'Azure'
        ResourceId: aiSearchService.id
        location: aiSearchService.location
      }
    }
  }
}

// Outputs

output aiProjectName string = aiProject.name
output aiProjectResourceId string = aiProject.id
output aiProjectPrincipalId string = aiProject.identity.principalId

output aiSearchConnection string = aiSearchName
output azureStorageConnection string = storageAccountName
output cosmosDbConnection string = cosmosDbAccountName

// This is used for storage naming conventions and is needed to help
// create the right fine-grained role assignments. The naming
// convention also uses dashes injected into the value, so we're
// handling that here.
// This will likely change or be made available via a different property.
#disable-next-line BCP053
var internalId = aiProject.properties.internalId
output projectWorkspaceId string = '${substring(internalId, 0, 8)}-${substring(internalId, 8, 4)}-${substring(internalId, 12, 4)}-${substring(internalId, 16, 4)}-${substring(internalId, 20, 12)}'

// This endpoint is also built by convention at this time but will
// hopefully be available as a different property at some point.
output projectEndpoint string = 'https://${aiServicesAccountName}.services.ai.azure.com/api/projects/${aiProjectName}'
