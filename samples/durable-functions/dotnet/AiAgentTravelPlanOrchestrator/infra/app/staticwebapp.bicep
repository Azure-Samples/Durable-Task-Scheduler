param name string
param location string = resourceGroup().location
param tags object = {}
param userAssignedIdentityId string = ''
param backendResourceId string = ''

resource staticWebApp 'Microsoft.Web/staticSites@2024-04-01' = {
  name: name
  location: location
  tags: tags
  sku: {
    name: 'Standard'
    tier: 'Standard'
  }
  identity: !empty(userAssignedIdentityId) ? {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '${userAssignedIdentityId}': {}
    }
  } : null
  properties: {
    stagingEnvironmentPolicy: 'Enabled'
    allowConfigFileUpdates: true
    enterpriseGradeCdnStatus: 'Disabled'
    provider: 'Custom'
    buildProperties: {
      skipGithubActionWorkflowGeneration: true
    }
  }
}

@description('Link backend to static web app')
resource linkedStaticWebAppBackend 'Microsoft.Web/staticSites/linkedBackends@2024-04-01' = if (!empty(backendResourceId)) {
  parent: staticWebApp
  name: 'linkedBackend'
  properties: {
    backendResourceId: backendResourceId
    region: location
  }
}

output id string = staticWebApp.id
output name string = staticWebApp.name
output uri string = 'https://${staticWebApp.properties.defaultHostname}'
