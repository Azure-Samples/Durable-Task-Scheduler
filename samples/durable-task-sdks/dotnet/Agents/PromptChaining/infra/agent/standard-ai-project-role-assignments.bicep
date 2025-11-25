param aiProjectPrincipalId string
param aiProjectPrincipalType string = 'ServicePrincipal' // Workaround for https://learn.microsoft.com/en-us/azure/role-based-access-control/role-assignments-template#new-service-principal
param userPrincipalId string = ''
param allowUserIdentityPrincipal bool = false // Flag to enable user identity role assignments

param aiServicesName string
param aiSearchName string
param aiCosmosDbName string
param aiStorageAccountName string

param integrationStorageAccountName string

// Parameters for function app managed identity
param functionAppManagedIdentityPrincipalId string = ''
param allowFunctionAppIdentityPrincipal bool = true // Flag to enable function app identity role assignments

// Parameters for DALL-E deployment
param dalleAiServicesId string = ''

// Assignments for AI Services
// ------------------------------------------------------------------

resource aiServices 'Microsoft.CognitiveServices/accounts@2024-06-01-preview' existing = {
  name: aiServicesName
}

// Assign AI Project the Cognitive Services Contributor Role on the AI Services resource

var cognitiveServicesContributorRoleDefinitionId = '25fbc0a9-bd7c-42a3-aa1a-3b75d497ee68'

resource cognitiveServicesContributorAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01'= {
  scope: aiServices
  name: guid(aiServices.id, cognitiveServicesContributorRoleDefinitionId, aiProjectPrincipalId)
  properties: {  
    principalId: aiProjectPrincipalId
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', cognitiveServicesContributorRoleDefinitionId)
    principalType: aiProjectPrincipalType
  }
}

// Assign AI Project the Cognitive Services OpenAI User Role on the AI Services resource

var cognitiveServicesOpenAIUserRoleDefinitionId = '5e0bd9bd-7b93-4f28-af87-19fc36ad61bd'

resource cognitiveServicesOpenAIUserRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  scope: aiServices
  name: guid(aiProjectPrincipalId, cognitiveServicesOpenAIUserRoleDefinitionId, aiServices.id)
  properties: {
    principalId: aiProjectPrincipalId
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', cognitiveServicesOpenAIUserRoleDefinitionId)
    principalType: aiProjectPrincipalType
  }
}

// Assign AI Project the Cognitive Services User Role on the AI Services resource

var cognitiveServicesUserRoleDefinitionId = 'a97b65f3-24c7-4388-baec-2e87135dc908'

resource cognitiveServicesUserRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  scope: aiServices
  name: guid(aiProjectPrincipalId, cognitiveServicesUserRoleDefinitionId, aiServices.id)
  properties: {
    principalId: aiProjectPrincipalId
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', cognitiveServicesUserRoleDefinitionId)
    principalType: aiProjectPrincipalType
  }
}

// Assign AI Project the Cognitive Services Contributor Role on the User Identity Principal on the AI Services resource

resource cognitiveServicesContributorAssignmentUser 'Microsoft.Authorization/roleAssignments@2022-04-01'= if (allowUserIdentityPrincipal && !empty(userPrincipalId)) {
  scope: aiServices
  name: guid(aiServices.id, cognitiveServicesContributorRoleDefinitionId, userPrincipalId)
  properties: {  
    principalId: userPrincipalId
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', cognitiveServicesContributorRoleDefinitionId)
    principalType: 'User'
  }
}

// Assign AI Project the Cognitive Services OpenAI User Role on the User Identity Principal on the AI Services resource

resource cognitiveServicesOpenAIUserRoleAssignmentUser 'Microsoft.Authorization/roleAssignments@2022-04-01' = if (allowUserIdentityPrincipal && !empty(userPrincipalId)){
  scope: aiServices
  name: guid(userPrincipalId, cognitiveServicesOpenAIUserRoleDefinitionId, aiServices.id)
  properties: {
    principalId: userPrincipalId
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', cognitiveServicesOpenAIUserRoleDefinitionId)
    principalType: 'User'
  }
}

// Assign AI Project the Cognitive Services User Role on the User Identity Principal on the AI Services resource

resource cognitiveServicesUserRoleAssignmentUser 'Microsoft.Authorization/roleAssignments@2022-04-01' = if (allowUserIdentityPrincipal && !empty(userPrincipalId)){
  scope: aiServices
  name: guid(userPrincipalId, cognitiveServicesUserRoleDefinitionId, aiServices.id)
  properties: {
    principalId: userPrincipalId
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', cognitiveServicesUserRoleDefinitionId)
    principalType: 'User'
  }
}

// Assignments for AI Search Service
// ------------------------------------------------------------------

resource aiSearchService 'Microsoft.Search/searchServices@2024-06-01-preview' existing = {
  name: aiSearchName
}

// Assign AI Project the Search Index Data Contributor Role on the AI Search Service resource

var searchIndexDataContributorRoleDefinitionId = '8ebe5a00-799e-43f5-93ac-243d3dce84a7'

resource searchIndexDataContributorAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  scope: aiSearchService
  name: guid(aiProjectPrincipalId, searchIndexDataContributorRoleDefinitionId, aiSearchService.id)
  properties: {
    principalId: aiProjectPrincipalId
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', searchIndexDataContributorRoleDefinitionId)
    principalType: aiProjectPrincipalType
  }
}

// Assign AI Project the Search Index Data Contributor Role on the AI Search Service resource

var searchServiceContributorRoleDefinitionId = '7ca78c08-252a-4471-8644-bb5ff32d4ba0'

resource searchServiceContributorRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  scope: aiSearchService
  name: guid(aiProjectPrincipalId, searchServiceContributorRoleDefinitionId, aiSearchService.id)
  properties: {
    principalId: aiProjectPrincipalId
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', searchServiceContributorRoleDefinitionId)
    principalType: aiProjectPrincipalType
  }
}

// Assignments for Storage Account
// ------------------------------------------------------------------

resource aiStorageAccount 'Microsoft.Storage/storageAccounts@2024-01-01' existing = {
  name: aiStorageAccountName 
}

// Assign AI Project the Storage Blob Data Contributor Role on the Storage Account resource

var storageBlobDataContributorRoleDefinitionId = 'ba92f5b4-2d11-453d-a403-e96b0029c9fe'

resource storageBlobDataContributorRoleAssignmentProject 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  scope: aiStorageAccount
  name: guid(aiProjectPrincipalId, storageBlobDataContributorRoleDefinitionId, aiStorageAccount.id)
  properties: {
    principalId: aiProjectPrincipalId
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', storageBlobDataContributorRoleDefinitionId)
    principalType: aiProjectPrincipalType
  }
}

// Assignments for Cosmos DB
// ------------------------------------------------------------------

resource cosmosDbAccount 'Microsoft.DocumentDB/databaseAccounts@2024-12-01-preview' existing = {
  name: aiCosmosDbName
}

// Assign AI Project the Cosmos DB Operator Role on the Cosmos DB Account resource

var cosmosDbOperatorRoleDefinitionId = '230815da-be43-4aae-9cb4-875f7bd000aa'

resource cosmosDbOperatorRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  scope: cosmosDbAccount
  name: guid(aiProjectPrincipalId, cosmosDbOperatorRoleDefinitionId, cosmosDbAccount.id)
  properties: {
    principalId: aiProjectPrincipalId
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', cosmosDbOperatorRoleDefinitionId)
    principalType: aiProjectPrincipalType
  }
}

// Assignments for Storage Account
// ------------------------------------------------------------------

resource integrationStorageAccount 'Microsoft.Storage/storageAccounts@2024-01-01' existing = {
  name: integrationStorageAccountName 
}

// Assign AI Project Storage Queue Data Contributor Role on the integration Storage Account resource
// between the agent and azure function

var storageQueueDataContributorRoleDefinitionId  = '974c5e8b-45b9-4653-ba55-5f855dd0fb88'

resource storageQueueDataContributorRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(integrationStorageAccount.id, aiProjectPrincipalId, storageQueueDataContributorRoleDefinitionId)
  scope: integrationStorageAccount
  properties: {
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', storageQueueDataContributorRoleDefinitionId)
    principalId: aiProjectPrincipalId
    principalType: aiProjectPrincipalType
  }
}

// assignments for User Identity Principal

resource storageQueueDataContributorRoleAssignmentUser 'Microsoft.Authorization/roleAssignments@2022-04-01' = if (allowUserIdentityPrincipal && !empty(userPrincipalId)) {
  name: guid(integrationStorageAccount.id, userPrincipalId, storageQueueDataContributorRoleDefinitionId)
  scope: integrationStorageAccount
  properties: {
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', storageQueueDataContributorRoleDefinitionId)
    principalId: userPrincipalId
    principalType: 'User'
  }
}

// Assign Function App Managed Identity the Cognitive Services Contributor Role on the AI Services resource

resource cognitiveServicesContributorAssignmentFunctionApp 'Microsoft.Authorization/roleAssignments@2022-04-01'= if (allowFunctionAppIdentityPrincipal && !empty(functionAppManagedIdentityPrincipalId)) {
  scope: aiServices
  name: guid(aiServices.id, cognitiveServicesContributorRoleDefinitionId, functionAppManagedIdentityPrincipalId)
  properties: {  
    principalId: functionAppManagedIdentityPrincipalId
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', cognitiveServicesContributorRoleDefinitionId)
    principalType: 'ServicePrincipal'
  }
}

// Assign Function App Managed Identity the Cognitive Services OpenAI User Role on the AI Services resource

resource cognitiveServicesOpenAIUserRoleAssignmentFunctionApp 'Microsoft.Authorization/roleAssignments@2022-04-01' = if (allowFunctionAppIdentityPrincipal && !empty(functionAppManagedIdentityPrincipalId)) {
  scope: aiServices
  name: guid(functionAppManagedIdentityPrincipalId, cognitiveServicesOpenAIUserRoleDefinitionId, aiServices.id)
  properties: {
    principalId: functionAppManagedIdentityPrincipalId
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', cognitiveServicesOpenAIUserRoleDefinitionId)
    principalType: 'ServicePrincipal'
  }
}

// Assign Function App Managed Identity the Cognitive Services User Role on the AI Services resource

resource cognitiveServicesUserRoleAssignmentFunctionApp 'Microsoft.Authorization/roleAssignments@2022-04-01' = if (allowFunctionAppIdentityPrincipal && !empty(functionAppManagedIdentityPrincipalId)) {
  scope: aiServices
  name: guid(functionAppManagedIdentityPrincipalId, cognitiveServicesUserRoleDefinitionId, aiServices.id)
  properties: {
    principalId: functionAppManagedIdentityPrincipalId
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', cognitiveServicesUserRoleDefinitionId)
    principalType: 'ServicePrincipal'
  }
}

// DALL-E Role Assignments
// ------------------------------------------------------------------

// Reference to the Image Generation AI Services resource
resource imageGenAiServices 'Microsoft.CognitiveServices/accounts@2024-06-01-preview' existing = {
  name: last(split(dalleAiServicesId, '/'))
}

// Assign Function App Managed Identity the Cognitive Services OpenAI User Role on the Image Generation AI Services resource
resource imageGenOpenAIUserRoleAssignmentFunctionApp 'Microsoft.Authorization/roleAssignments@2022-04-01' = if (allowFunctionAppIdentityPrincipal && !empty(functionAppManagedIdentityPrincipalId)) {
  scope: imageGenAiServices
  name: guid(functionAppManagedIdentityPrincipalId, cognitiveServicesOpenAIUserRoleDefinitionId, dalleAiServicesId)
  properties: {
    principalId: functionAppManagedIdentityPrincipalId
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', cognitiveServicesOpenAIUserRoleDefinitionId)
    principalType: 'ServicePrincipal'
  }
}
