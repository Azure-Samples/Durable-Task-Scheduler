// Assigns a role on the Foundry resource (AIServices account) to a principal.

@description('Name of the Foundry resource (AIServices account)')
param aiFoundryName string

@description('Role definition ID to assign')
param roleDefinitionID string

@description('Principal ID to assign the role to')
param principalID string

@description('Type of the principal')
@allowed(['User', 'ServicePrincipal', 'Group'])
param principalType string = 'ServicePrincipal'

resource aiFoundry 'Microsoft.CognitiveServices/accounts@2025-06-01' existing = {
  name: aiFoundryName
}

resource roleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(aiFoundry.id, principalID, roleDefinitionID)
  scope: aiFoundry
  properties: {
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', roleDefinitionID)
    principalId: principalID
    principalType: principalType
  }
}
