// ai-role-assignments.bicep
// Assigns the necessary roles for AI resources

@description('Principal ID to assign roles to')
param principalId string

@description('Name of the OpenAI resource')
param openAiResourceName string = ''

@description('Name of the AI Hub')
param aiHubName string = ''

@description('Name of the AI Project')
param aiProjectName string = ''

// If OpenAI resource name is provided, assign roles
resource openAi 'Microsoft.CognitiveServices/accounts@2023-05-01' existing = if (!empty(openAiResourceName)) {
  name: openAiResourceName
}

// Assign Cognitive Services OpenAI User role
resource openAiUserRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = if (!empty(openAiResourceName)) {
  name: guid(openAi.id, principalId, '5e0bd9bd-7b93-4f28-af87-19fc36ad61bd')
  scope: openAi
  properties: {
    principalId: principalId
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '5e0bd9bd-7b93-4f28-af87-19fc36ad61bd') // Cognitive Services OpenAI User
    principalType: 'ServicePrincipal'
  }
}

// Assign Cognitive Services Contributor role
resource cognitiveServicesContributorRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = if (!empty(openAiResourceName)) {
  name: guid(openAi.id, principalId, '25fbc0a9-bd7c-42a3-aa1a-3b75d497ee68')
  scope: openAi
  properties: {
    principalId: principalId
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '25fbc0a9-bd7c-42a3-aa1a-3b75d497ee68') // Cognitive Services Contributor
    principalType: 'ServicePrincipal'
  }
}

// Add AI Hub roles if hub name is provided
resource aiHub 'Microsoft.MachineLearningServices/workspaces@2024-01-01-preview' existing = if (!empty(aiHubName)) {
  name: aiHubName
}

resource aiHubContributorRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = if (!empty(aiHubName)) {
  name: guid(aiHub.id, principalId, 'b24988ac-6180-42a0-ab88-20f7382dd24c')
  scope: aiHub
  properties: {
    principalId: principalId
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', 'b24988ac-6180-42a0-ab88-20f7382dd24c') // Contributor
    principalType: 'ServicePrincipal'
  }
}

// Add AI Project roles if project name is provided
resource aiProject 'Microsoft.MachineLearningServices/workspaces@2023-08-01-preview' existing = if (!empty(aiProjectName)) {
  name: aiProjectName
}

resource aiProjectContributorRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = if (!empty(aiProjectName)) {
  name: guid(aiProject.id, principalId, 'b24988ac-6180-42a0-ab88-20f7382dd24c')
  scope: aiProject
  properties: {
    principalId: principalId
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', 'b24988ac-6180-42a0-ab88-20f7382dd24c') // Contributor
    principalType: 'ServicePrincipal'
  }
}

// Add Reader role - required for the managed identity to use AI Project services
resource aiProjectReaderRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = if (!empty(aiProjectName)) {
  name: guid(aiProject.id, principalId, 'acdd72a7-3385-48ef-bd42-f606fba81ae7')
  scope: aiProject
  properties: {
    principalId: principalId
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', 'acdd72a7-3385-48ef-bd42-f606fba81ae7') // Reader role
    principalType: 'ServicePrincipal'
  }
}

output openAiUserRoleAssignmentId string = !empty(openAiResourceName) ? openAiUserRoleAssignment.id : ''
output cognitiveServicesContributorRoleAssignmentId string = !empty(openAiResourceName) ? cognitiveServicesContributorRoleAssignment.id : ''
output aiProjectReaderRoleAssignmentId string = !empty(aiProjectName) ? aiProjectReaderRoleAssignment.id : ''
