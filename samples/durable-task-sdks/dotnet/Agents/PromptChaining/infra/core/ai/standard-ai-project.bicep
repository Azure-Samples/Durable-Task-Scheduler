// standard-ai-project.bicep
// This file defines an Azure AI Project resource and capability host for agents
// Based on the Travel Plan Orchestrator example

@description('Azure region of the deployment')
param location string

@description('Tags to add to the resources')
param tags object = {}

@description('AI Project name')
param aiProjectName string

@description('AI Project display name')
param aiProjectFriendlyName string = aiProjectName

@description('AI Project description')
param aiProjectDescription string = 'AI Project for agent chaining functionality'

@description('Resource ID of the AI Hub resource')
param aiHubId string

@description('Name for capabilityHost')
param capabilityHostName string = 'agent-host'

@description('Name for OpenAI connection')
param aoaiConnectionName string = 'openai-connection'

// For constructing endpoint
var subscriptionId = subscription().subscriptionId
var resourceGroupName = resourceGroup().name
// Original semicolon format - kept for reference/compatibility
var standardConnectionString = '${location}.api.azureml.ms;${subscriptionId};${resourceGroupName};${aiProjectName}'
// URL format connection string that matches the SDK requirements
// Using standard Azure AI Project endpoint format
var projectConnectionString = 'https://${location}.aiprojects.azure.com/api/projects/${resourceGroupName}/${aiProjectName}'

// Define storage connections exactly as in the example
var storageConnections = ['${aiProjectName}/workspaceblobstore']

// Define a dummy vector store connection as required by the API
var vectorStoreConnections = ['dummy-vectorstore']

// Define AI services connections - must match the connection name from the hub
// This is critical - must match exactly the connection name defined in the AI Hub
var aiServiceConnections = [aoaiConnectionName]

resource aiProject 'Microsoft.MachineLearningServices/workspaces@2023-08-01-preview' = {
  name: aiProjectName
  location: location
  tags: union(tags, {
    ProjectConnectionString: projectConnectionString
    StandardConnectionString: standardConnectionString
  })
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    // organization
    friendlyName: aiProjectFriendlyName
    description: aiProjectDescription

    // dependent resources
    hubResourceId: aiHubId
  }
  kind: 'project'

  // Resource definition for the capability host
  resource capabilityHost 'capabilityHosts@2024-10-01-preview' = {
    name: '${aiProjectName}-${capabilityHostName}'
    properties: {
      capabilityHostKind: 'Agents'
      aiServicesConnections: aiServiceConnections
      vectorStoreConnections: vectorStoreConnections
      storageConnections: storageConnections
    }
  }
}

output aiProjectName string = aiProject.name
output aiProjectResourceId string = aiProject.id
output aiProjectPrincipalId string = aiProject.identity.principalId
output projectConnectionString string = projectConnectionString
