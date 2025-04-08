param principalId string
param roleDefinitionId string
param principalType string = 'ServicePrincipal'
param resourceId string

// This is an extensible resource type that can be applied at different scopes
resource role_assignment 'Microsoft.Authorization/roleAssignments@2020-04-01-preview' = {
  name: guid(resourceId, principalId, roleDefinitionId)
  properties: {
    principalId: principalId
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', roleDefinitionId)
    principalType: principalType
  }
}
