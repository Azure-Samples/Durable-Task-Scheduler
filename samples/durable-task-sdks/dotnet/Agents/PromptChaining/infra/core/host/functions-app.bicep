param name string
param location string = resourceGroup().location
param tags object = {}

// Reference Properties
param applicationInsightsName string = ''
param appServicePlanId string
param storageAccountName string
param virtualNetworkSubnetId string = ''

@allowed(['SystemAssigned', 'UserAssigned'])
param identityType string

@description('User assigned identity name')
param identityId string

param kind string = 'functionapp,linux'

// Microsoft.Web/sites/config
param appSettings object = {}
param allowedOrigins array = []

var linuxFxVersion string = 'DOTNET-ISOLATED|8.0'

resource storageAccount 'Microsoft.Storage/storageAccounts@2024-01-01' existing = {
  name: storageAccountName
}

resource applicationInsights 'Microsoft.Insights/components@2020-02-02' existing = if (!empty(applicationInsightsName)) {
  name: applicationInsightsName
}

module functions 'br/public:avm/res/web/site:0.16.0' = {
  name: 'siteDeployment'
  params: {
    name: name
    kind: kind
    location: location
    tags: tags
    serverFarmResourceId: appServicePlanId
    managedIdentities: {
      userAssignedResourceIds: [identityId]
    }
    siteConfig: {
      netFrameworkVersion: 'v8.0'
      linuxFxVersion: linuxFxVersion
      cors: {
        allowedOrigins: union([ 'https://portal.azure.com', 'https://ms.portal.azure.com' ], allowedOrigins)
      }
      alwaysOn: true
      use32BitWorkerProcess: false
      ftpsState: 'FtpsOnly'
    }
    virtualNetworkSubnetId: !empty(virtualNetworkSubnetId) ? virtualNetworkSubnetId : null
    configs: [
      {
        name: 'appsettings'
        properties: union(appSettings,
        {
          FUNCTIONS_EXTENSION_VERSION: '~4'
          FUNCTIONS_WORKER_RUNTIME: 'dotnet-isolated'
          AzureWebJobsStorage__accountName: storageAccount.name
          AzureWebJobsStorage__credential : 'managedidentity'
          APPLICATIONINSIGHTS_CONNECTION_STRING: applicationInsights.properties.ConnectionString
          WEBSITE_USE_PLACEHOLDER_DOTNETISOLATED: '1'
        })
      }
    ]
  }
}

output name string = functions.outputs.name
output uri string = 'https://${functions.outputs.defaultHostname}'
output identityPrincipalId string = identityType == 'SystemAssigned' ? functions.outputs.systemAssignedMIPrincipalId : ''
output id string = functions.outputs.resourceId
