// storage-Access.bicep
// Assigns role to a principal on a storage account

@description('The name of the storage account')
param storageAccountName string

@description('The principal ID to assign the role to')
param principalID string

@description('The role definition ID to assign')
param roleDefinitionID string

@description('The type of the principal (User, Group, ServicePrincipal)')
param principalType string

resource storageAccount 'Microsoft.Storage/storageAccounts@2023-01-01' existing = {
  name: storageAccountName
}

resource roleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(storageAccount.id, principalID, roleDefinitionID)
  scope: storageAccount
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', roleDefinitionID)
    principalId: principalID
    principalType: principalType
  }
}

output roleAssignmentId string = roleAssignment.id
