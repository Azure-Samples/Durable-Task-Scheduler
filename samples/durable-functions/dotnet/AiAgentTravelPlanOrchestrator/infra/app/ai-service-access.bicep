// Assigns the necessary roles for a Function App to access AI Services using managed identity

param aiServicesName string
param principalId string
param principalType string = 'ServicePrincipal'

resource aiServices 'Microsoft.CognitiveServices/accounts@2024-06-01-preview' existing = {
  name: aiServicesName
  scope: resourceGroup()
}

// Cognitive Services OpenAI User - allows calling the API
var cognitiveServicesOpenAIUserRoleId = '5e0bd9bd-7b93-4f28-af87-19fc36ad61bd'
resource cognitiveServicesOpenAIUserRole 'Microsoft.Authorization/roleDefinitions@2022-04-01' existing = {
  name: cognitiveServicesOpenAIUserRoleId
  scope: subscription()
}

resource cognitiveServicesOpenAIUserRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  scope: aiServices
  name: guid(aiServices.id, cognitiveServicesOpenAIUserRole.id, principalId)
  properties: {
    principalId: principalId
    roleDefinitionId: cognitiveServicesOpenAIUserRole.id
    principalType: principalType
  }
}
