metadata description = 'Creates an Azure Function in an existing Azure App Service plan.'
param name string
param location string = resourceGroup().location
param tags object = {}

// Reference Properties
param applicationInsightsName string = ''
param appServicePlanId string
param keyVaultName string = ''
param managedIdentity bool = !empty(keyVaultName) || storageManagedIdentity
param storageAccountName string
param storageManagedIdentity bool = false
param virtualNetworkSubnetId string = ''
@allowed(['SystemAssigned', 'UserAssigned'])
param identityType string
@description('User assigned identity name')
param identityId string

// Runtime Properties
@allowed([
  'dotnet', 'dotnetcore', 'dotnet-isolated', 'node', 'python', 'java', 'powershell', 'custom'
])
param runtimeName string
param runtimeNameAndVersion string = '${runtimeName}|${runtimeVersion}'
param runtimeVersion string

// Function Settings
@allowed([
  '~4', '~3', '~2', '~1'
])
param extensionVersion string = '~4'

// Microsoft.Web/sites Properties
param kind string = 'functionapp,linux'

// Microsoft.Web/sites/config
param allowedOrigins array = []
param alwaysOn bool = true
param appCommandLine string = ''
@secure()
param appSettings object = {}
param clientAffinityEnabled bool = false
param enableOryxBuild bool = contains(kind, 'linux')
param functionAppScaleLimit int = -1
param linuxFxVersion string = runtimeNameAndVersion
param minimumElasticInstanceCount int = -1
param numberOfWorkers int = -1
param scmDoBuildDuringDeployment bool = true
param use32BitWorkerProcess bool = false
param healthCheckPath string = ''
param deploymentStorageContainerName string
param deploymentStorageFileName string = 'functions' // Default to functions.zip name in WEBSITE_RUN_FROM_PACKAGE=<url> value 

module functions 'appservice.bicep' = {
  name: '${name}-functions'
  params: {
    name: name
    location: location
    tags: tags
    allowedOrigins: allowedOrigins
    alwaysOn: alwaysOn
    appCommandLine: appCommandLine
    applicationInsightsName: applicationInsightsName
    appServicePlanId: appServicePlanId
    appSettings: union(appSettings, {
        AzureWebJobsStorage__accountName: storageManagedIdentity ? storage.name : null
        FUNCTIONS_EXTENSION_VERSION: extensionVersion
        FUNCTIONS_WORKER_RUNTIME: runtimeName
        WEBSITE_RUN_FROM_PACKAGE: 'https://${storage.name}.blob.core.windows.net/${deploymentStorageContainerName}/${deploymentStorageFileName}.zip'
        WEBSITE_RUN_FROM_PACKAGE_BLOB_MI_RESOURCE_ID: storageManagedIdentity ? identityId : null
      })
    clientAffinityEnabled: clientAffinityEnabled
    enableOryxBuild: enableOryxBuild
    functionAppScaleLimit: functionAppScaleLimit
    healthCheckPath: healthCheckPath
    keyVaultName: keyVaultName
    kind: kind
    linuxFxVersion: linuxFxVersion
    managedIdentity: managedIdentity
    identityId: identityId
    identityType: identityType
    minimumElasticInstanceCount: minimumElasticInstanceCount
    numberOfWorkers: numberOfWorkers
    runtimeName: runtimeName
    runtimeVersion: runtimeVersion
    runtimeNameAndVersion: runtimeNameAndVersion
    scmDoBuildDuringDeployment: scmDoBuildDuringDeployment
    use32BitWorkerProcess: use32BitWorkerProcess
    virtualNetworkSubnetId: virtualNetworkSubnetId
  }
}

module storageOwnerRole '../security/role.bicep' = if (storageManagedIdentity) {
  name: 'search-index-contrib-role-api'
  params: {
    principalId: functions.outputs.identityPrincipalId
    // Search Index Data Contributor
    roleDefinitionId: '8ebe5a00-799e-43f5-93ac-243d3dce84a7'
    principalType: 'ServicePrincipal'
  }
}

resource storage 'Microsoft.Storage/storageAccounts@2021-09-01' existing = {
  name: storageAccountName
}

output identityPrincipalId string = managedIdentity ? functions.outputs.identityPrincipalId : ''
output name string = functions.outputs.name
output uri string = functions.outputs.uri
