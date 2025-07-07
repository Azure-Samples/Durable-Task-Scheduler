param principalID string
param principalType string = 'ServicePrincipal' // Workaround for https://learn.microsoft.com/en-us/azure/role-based-access-control/role-assignments-template#new-service-principal
param roleDefinitionID string
param storageAccountName string

resource storageAccount 'Microsoft.Storage/storageAccounts@2024-01-01' existing = {
  name: storageAccountName
}

// Allow access from API to storage account using a managed identity and least priv Storage roles
resource storageRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  scope: storageAccount
  name: guid(storageAccount.id, principalID, roleDefinitionID) // the guid function creates a unique name for the role assignment
  properties: {
    roleDefinitionId: resourceId('Microsoft.Authorization/roleDefinitions', roleDefinitionID)
    principalId: principalID
    principalType: principalType
  }
}
output FullyQualifiedRoleDefinitionID string = resourceId('Microsoft.Authorization/roleDefinitions', roleDefinitionID)
output ROLE_ASSIGNMENT_NAME string = storageRoleAssignment.name
