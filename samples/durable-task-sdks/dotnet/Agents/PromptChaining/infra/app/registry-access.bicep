@description('The name of the Container Registry')
param containerRegistryName string

@description('The Principal ID of the identity that needs access to the registry')
param principalID string

@description('The type of the principal (User, ServicePrincipal, etc.)')
param principalType string = 'ServicePrincipal'

// AcrPull role definition ID: 7f951dda-4ed3-4680-a7ca-43fe172d538d
var acrPullRoleDefinitionId = '7f951dda-4ed3-4680-a7ca-43fe172d538d'

resource containerRegistry 'Microsoft.ContainerRegistry/registries@2023-11-01-preview' existing = {
  name: containerRegistryName
}

resource acrPullRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(containerRegistry.id, principalID, acrPullRoleDefinitionId)
  scope: containerRegistry
  properties: {
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', acrPullRoleDefinitionId)
    principalId: principalID
    principalType: principalType
  }
}

output roleAssignmentId string = acrPullRoleAssignment.id
