metadata description = 'Creates an Azure App Service in an existing Azure App Service plan.'
param name string
param location string = resourceGroup().location
param tags object = {}

// Reference Properties
param appServicePlanId string
param keyVaultName string = ''
param managedIdentity bool = !empty(keyVaultName)

param storageEndpoint string = ''
param cosmosDbEndpoint string = ''
// Runtime Properties
@allowed([
  'dotnet', 'dotnetcore', 'dotnet-isolated', 'node', 'python', 'java', 'powershell', 'custom'
])
param runtimeName string
param runtimeVersion string
param runtimeNameAndVersion string = '${runtimeName}|${runtimeVersion}'

// Microsoft.Web/sites/config
param userAssignedIdentityId string = ''
param userAssignedIdentityClientId string = ''

resource appService 'Microsoft.Web/sites@2022-03-01' = {
  name: name
  location: location
  tags: tags
  properties: {
    serverFarmId: appServicePlanId
    siteConfig: {
      windowsFxVersion: runtimeNameAndVersion
      webSocketsEnabled: true
    }
    httpsOnly: true
  }
  
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '${userAssignedIdentityId}': {}
    }
  }

  resource basicPublishingCredentialsPoliciesFtp 'basicPublishingCredentialsPolicies' = {
    name: 'ftp'
    properties: {
      allow: false
    }
  }

  resource basicPublishingCredentialsPoliciesScm 'basicPublishingCredentialsPolicies' = {
    name: 'scm'
    properties: {
      allow: false
    }
  }
}

// add connectionstring
resource symbolicname 'Microsoft.Web/sites/config@2022-09-01' = {
  name: 'appsettings'
  kind: 'string'
  parent: appService
  properties: {
    AZURE_COSMOS_DB_NOSQL_ENDPOINT: cosmosDbEndpoint
    STORAGE_URL: storageEndpoint
    AZURE_CLIENT_ID: userAssignedIdentityClientId
  }
}

output identityPrincipalId string = managedIdentity ? appService.identity.principalId : ''
output name string = appService.name
output uri string = 'https://${appService.properties.defaultHostName}'
