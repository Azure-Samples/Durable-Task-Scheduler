// This file defines an Azure AI Project resource and capability host for agents

@description('Name of the AI Project')
param name string

@description('Azure region for the project')
param location string = resourceGroup().location

@description('Project SKU')
param sku object = {
  name: 'Free'
  tier: 'Free'
}

@description('Tags for the resource')
param tags object = {}

@description('The Azure OpenAI resource ID to connect to this project')
param openAiResourceId string

@description('The Azure OpenAI resource name for connections')
param openAiName string

@description('The Hub resource ID to associate with this project')
param hubResourceId string

@description('If a Managed Service Identity for this AI Project should be created')
param managedIdentity bool = true

// For creating proper connection strings
var subscriptionId = subscription().subscriptionId
var resourceGroupName = resourceGroup().name
// Original semicolon format - kept for reference
var standardConnectionString = '${location}.api.azureml.ms;${subscriptionId};${resourceGroupName};${name}'
// URL format connection string that matches the SDK requirements
// Using standard Azure AI Project endpoint format
var projectConnectionString = 'https://${location}.aiprojects.azure.com/api/projects/${resourceGroupName}/${name}'

var capabilityHostName = 'agent-host'

// Define the necessary connections for the capability host
var storageConnections = ['${name}/workspaceblobstore']
var aiServiceConnections = ['${openAiName}-connection'] 
var vectorStoreConnections = ['${name}/default-vectorstore'] // Required non-empty value

// Create the AI Project with the proper hub association and capability host
resource aiProject 'Microsoft.MachineLearningServices/workspaces@2023-08-01-preview' = {
  name: name
  location: location
  tags: union(tags, {
    ProjectConnectionString: projectConnectionString
    StandardConnectionString: standardConnectionString
    OpenAiResourceId: openAiResourceId
  })
  identity: {
    type: managedIdentity ? 'SystemAssigned' : 'None'
  }
  sku: sku
  kind: 'project'
  properties: {
    friendlyName: 'Agent Chaining AI Project'
    description: 'AI Project for the agent chaining sample'
    hubResourceId: hubResourceId
    publicNetworkAccess: 'Enabled'
    // Do not reference storage account directly for projects
    // Storage is managed by the Hub
  }

  // Create the capability host for agents - follows the pattern from Travel Plan Orchestrator example
  resource capabilityHost 'capabilityHosts@2024-10-01-preview' = {
    name: '${name}-${capabilityHostName}'
    properties: {
      capabilityHostKind: 'Agents'
      aiServicesConnections: aiServiceConnections
      vectorStoreConnections: vectorStoreConnections // Using non-empty vector store connection
      storageConnections: storageConnections
    }
  }
}

// End of aiProject resource

// Add role assignment for the AI Project to access storage
resource storageRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = if (managedIdentity) {
  name: guid(resourceGroup().id, aiProject.id, 'ba92f5b4-2d11-453d-a403-e96b0029c9fe')
  scope: resourceGroup()
  properties: {
    principalId: aiProject.identity.principalId
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', 'ba92f5b4-2d11-453d-a403-e96b0029c9fe') // Storage Blob Data Contributor
    principalType: 'ServicePrincipal'
  }
}

// Output the project ID and information
output id string = aiProject.id
output name string = aiProject.name
output projectConnectionString string = projectConnectionString
output principalId string = managedIdentity ? aiProject.identity.principalId : ''
output projectResourceId string = aiProject.id
output capabilityHostName string = '${name}-${capabilityHostName}'
