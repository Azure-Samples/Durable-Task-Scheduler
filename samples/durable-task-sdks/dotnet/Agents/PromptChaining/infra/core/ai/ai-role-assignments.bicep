// ai-role-assignments.bicep - Role assignments for AI services

@description('The principal ID to assign roles to')
param principalId string

@description('The ID of the Azure AI Project')
param aiProjectId string

@description('The type of the principal (ServicePrincipal, User, Group, etc)')
param principalType string = 'ServicePrincipal'

// AI Project Contributor role definition ID
var aiProjectContributorRoleId = '1affc506-2bb4-4bbd-86a8-804c7c9d7a99' // Azure AI Project Contributor

// Create role assignment for AI Project Contributor
resource aiProjectContributorRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(aiProjectId, principalId, aiProjectContributorRoleId)
  scope: resourceGroup()
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', aiProjectContributorRoleId)
    principalId: principalId
    principalType: principalType
  }
}

output aiProjectRoleAssignmentId string = aiProjectContributorRoleAssignment.id
