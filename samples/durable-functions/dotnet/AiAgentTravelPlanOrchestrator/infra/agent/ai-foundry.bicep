// Creates a Microsoft Foundry resource (AIServices account), a Foundry project, and a model deployment.
// The project endpoint output is the value expected by Azure.AI.Projects 2.x (AIProjectClient).

@description('Azure region of the Foundry resource and model deployment')
param location string

@description('Tags to add to the resources')
param tags object = {}

@description('Name of the Foundry resource (AIServices account). Also used as the custom subdomain.')
param aiFoundryName string

@description('Name of the Foundry project')
param aiProjectName string

@description('Display name of the Foundry project')
param aiProjectFriendlyName string = aiProjectName

@description('Description of the Foundry project')
param aiProjectDescription string = ''

@description('Model name for deployment')
param modelName string

@description('Model format for deployment')
param modelFormat string

@description('Model version for deployment')
param modelVersion string

@description('Model deployment SKU name')
param modelSkuName string

@description('Model deployment capacity')
param modelCapacity int

@description('Whether the Foundry resource accepts traffic from public networks')
@allowed(['Enabled', 'Disabled'])
param publicNetworkAccess string = 'Enabled'

resource aiFoundry 'Microsoft.CognitiveServices/accounts@2025-06-01' = {
  name: aiFoundryName
  location: location
  tags: tags
  kind: 'AIServices'
  sku: {
    name: 'S0'
  }
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    allowProjectManagement: true
    customSubDomainName: toLower(aiFoundryName)
    // The app authenticates with Microsoft Entra ID (managed identity), so keys are not needed.
    disableLocalAuth: true
    publicNetworkAccess: publicNetworkAccess
    networkAcls: {
      defaultAction: publicNetworkAccess == 'Enabled' ? 'Allow' : 'Deny'
    }
  }
}

resource aiProject 'Microsoft.CognitiveServices/accounts/projects@2025-06-01' = {
  parent: aiFoundry
  name: aiProjectName
  location: location
  tags: tags
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    displayName: aiProjectFriendlyName
    description: aiProjectDescription
  }
}

resource modelDeployment 'Microsoft.CognitiveServices/accounts/deployments@2025-06-01' = {
  parent: aiFoundry
  name: modelName
  sku: {
    name: modelSkuName
    capacity: modelCapacity
  }
  properties: {
    model: {
      name: modelName
      format: modelFormat
      version: modelVersion
    }
  }
  dependsOn: [
    aiProject
  ]
}

output aiFoundryName string = aiFoundry.name
output aiFoundryId string = aiFoundry.id
output aiProjectName string = aiProject.name
output aiProjectResourceId string = aiProject.id
output aiProjectPrincipalId string = aiProject.identity.principalId
output modelDeploymentName string = modelDeployment.name
// Format: https://<foundry-resource>.services.ai.azure.com/api/projects/<project>
output projectEndpoint string = aiProject.properties.endpoints['AI Foundry API']
