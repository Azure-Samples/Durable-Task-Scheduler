@description('The name of the Container Registry')
param containerRegistryName string

@description('The user principal ID that needs push access to the registry')
param userPrincipalID string

@description('The type of the principal (usually User)')
param principalType string = 'User'

// AcrPush role definition ID: 8311e382-0749-4cb8-b61a-304f252e45ec
var acrPushRoleDefinitionId = '8311e382-0749-4cb8-b61a-304f252e45ec'

resource containerRegistry 'Microsoft.ContainerRegistry/registries@2023-11-01-preview' existing = {
  name: containerRegistryName
}

// Only deploy if skip parameter is false
resource acrPushRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(containerRegistry.id, userPrincipalID, acrPushRoleDefinitionId)
  scope: containerRegistry
  properties: {
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', acrPushRoleDefinitionId)
    principalId: userPrincipalID
    principalType: principalType
  }
}
