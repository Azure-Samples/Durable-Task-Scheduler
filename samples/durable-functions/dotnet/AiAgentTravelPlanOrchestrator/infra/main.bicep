targetScope = 'subscription'
@minLength(1)
@maxLength(64)
@description('Name of the the environment which is used to generate a short unique hash used in all resources.')
param environmentName string

@minLength(1)
@description('Primary location for all resources')
@metadata({
  azd: {
    type: 'location'
  }
})
param location string

@description('Skip the creation of the virtual network and private endpoint')
param skipVnet bool = true

@description('Name of the API service')
param apiServiceName string = ''

@description('Name of the user assigned identity')
param apiUserAssignedIdentityName string = ''

@description('Name of the application insights resource')
param applicationInsightsName string = ''

@description('Name of the app service plan')
param appServicePlanName string = ''

@description('Name of the log analytics workspace')
param logAnalyticsName string = ''

@description('Name of the resource group')
param resourceGroupName string = ''

@description('Name of the storage account')
param storageAccountName string = ''

@description('Name of the virtual network')
param vNetName string = ''

@description('Disable local authentication for Azure Monitor')
param disableLocalAuth bool = true

@description('Name of the Durable Task Scheduler')
param dtsName string = ''

@description('Name of the task hub')
param taskHubName string = ''

@description('Durable Task Scheduler SKU name')
param dtsSkuName string = 'Consumption'

@description('Durable Task Scheduler SKU capacity')
param dtsCapacity int = 1

@description('Name of the web service')
param webServiceName string = ''

@description('Id of the user or app to assign application roles')
param principalId string = deployer().objectId

@description('Friendly name for your Hub resource')
param aiHubFriendlyName string = 'Agents Hub resource'

@description('Description of your Azure AI resource displayed in AI studio')
param aiHubDescription string = 'This is an example AI resource for use in Azure AI Studio.'

@description('Friendly name for your Azure AI resource')
param aiProjectFriendlyName string = 'Agents Project resource'

@description('Description of your Azure AI resource displayed in AI studio')
param aiProjectDescription string = 'This is an example AI Project resource for use in Azure AI Studio.'

@description('Name of the Azure AI Search account')
param aiSearchName string = 'agentaisearch'

@description('Name of the Azure AI Services account')
param aiServicesName string = 'agentaiservices'

@description('Model name for deployment')
param modelName string = 'gpt-5-mini'

@description('Model format for deployment')
param modelFormat string = 'OpenAI'

@description('Model version for deployment')
param modelVersion string = '2025-08-07'

@description('Model deployment SKU name')
param modelSkuName string = 'S0'

@description('Model deployment capacity')
param modelCapacity int = 10

@description('Model deployment location. If you want to deploy an Azure AI resource/model in different location than the rest of the resources created.')
param modelLocation string = location

@description('The AI Service Account full ARM Resource ID. This is an optional field, and if not provided, the resource will be created.')
param aiServiceAccountResourceId string = ''

@description('The Ai Search Service full ARM Resource ID. This is an optional field, and if not provided, the resource will be created.')
param aiSearchServiceResourceId string = ''

@description('The Ai Storage Account full ARM Resource ID. This is an optional field, and if not provided, the resource will be created.')
param aiStorageAccountResourceId string = ''

// Variables
var abbrs = loadJsonContent('./abbreviations.json')
var resourceToken = toLower(uniqueString(subscription().id, rg.id, environmentName, location))
var aiResourceToken = toLower(uniqueString(subscription().id, rg.id, environmentName, modelLocation))
var tags = { 'azd-env-name': environmentName }
var functionAppName = !empty(apiServiceName) ? apiServiceName : '${abbrs.webSitesFunctions}api-${resourceToken}'
var deploymentStorageContainerName = 'app-package-${take(functionAppName, 32)}-${take(toLower(uniqueString(functionAppName, resourceToken)), 7)}'
var name = 'hub'
var projectName = 'proj'
// Define the web app name first so we can construct the URL
var webAppName = !empty(webServiceName) ? webServiceName : '${abbrs.webStaticSites}web-${resourceToken}'
// Pre-compute the expected web URI for CORS settings
var webUri = 'https://${webAppName}.azurestaticapps.net'

// Create a short, unique suffix, that will be unique to each resource group
var aiUniqueSuffix = toLower(uniqueString(subscription().id, rg.id, modelLocation))

// Organize resources in a resource group
resource rg 'Microsoft.Resources/resourceGroups@2025-04-01' = {
  name: !empty(resourceGroupName) ? resourceGroupName : '${abbrs.resourcesResourceGroups}${environmentName}'
  location: location
  tags: tags
}

// The application frontend webapp using AVM
module webapp 'br/public:avm/res/web/static-site:0.9.3' = {
  name: 'webapp-${resourceToken}'
  scope: rg
  params: {
    name: webAppName
    location: location
    tags: union(tags, { 'azd-service-name': 'web' })
    sku: 'Standard'
    managedIdentities: {
      userAssignedResourceIds: [apiUserAssignedIdentity.outputs.resourceId]
    }
  }
}

// User assigned managed identity using AVM
module apiUserAssignedIdentity 'br/public:avm/res/managed-identity/user-assigned-identity:0.4.0' = {
  name: 'apiUserAssignedIdentity-${resourceToken}'
  scope: rg
  params: {
    name: !empty(apiUserAssignedIdentityName) ? apiUserAssignedIdentityName : '${abbrs.managedIdentityUserAssignedIdentities}api-${resourceToken}'
    location: location
    tags: tags
  }
}

// Backing storage for Azure functions using AVM
module storage 'br/public:avm/res/storage/storage-account:0.29.0' = {
  scope: rg
  name: 'storage-${resourceToken}'
  params: {
    name: !empty(storageAccountName) ? storageAccountName : '${abbrs.storageStorageAccounts}${resourceToken}'
    location: location
    tags: tags
    kind: 'StorageV2'
    skuName: 'Standard_LRS'
    allowSharedKeyAccess: false
    blobServices: {
      containers: [
        { name: deploymentStorageContainerName }
      ]
    }
    publicNetworkAccess: skipVnet ? 'Enabled' : 'Disabled'
    networkAcls: skipVnet ? {
        defaultAction: 'Allow'
        bypass: 'AzureServices'
      } : {
      defaultAction: 'Deny'
      bypass: 'AzureServices'
    }
    roleAssignments: [
      {
        principalId: apiUserAssignedIdentity.outputs.principalId
        roleDefinitionIdOrName: 'b7e6dc6d-f1e8-4753-8033-0f276bb0955b' // Storage Blob Data Owner
        principalType: 'ServicePrincipal'
      }
      {
        principalId: apiUserAssignedIdentity.outputs.principalId
        roleDefinitionIdOrName: 'ba92f5b4-2d11-453d-a403-e96b0029c9fe' // Storage Blob Data Contributor
        principalType: 'ServicePrincipal'
      }
      {
        principalId: apiUserAssignedIdentity.outputs.principalId
        roleDefinitionIdOrName: '0a9a7e1f-b9d0-4cc4-a60d-0319b160aaa3' // Storage Table Data Contributor
        principalType: 'ServicePrincipal'
      }
      {
        principalId: apiUserAssignedIdentity.outputs.principalId
        roleDefinitionIdOrName: '974c5e8b-45b9-4653-ba55-5f855dd0fb88' // Storage Queue Data Contributor
        principalType: 'ServicePrincipal'
      }
      {
        principalId: principalId
        roleDefinitionIdOrName: 'b7e6dc6d-f1e8-4753-8033-0f276bb0955b' // Storage Blob Data Owner
        principalType: 'User'
      }
      {
        principalId: principalId
        roleDefinitionIdOrName: 'ba92f5b4-2d11-453d-a403-e96b0029c9fe' // Storage Blob Data Contributor
        principalType: 'User'
      }
      {
        principalId: principalId
        roleDefinitionIdOrName: '0a9a7e1f-b9d0-4cc4-a60d-0319b160aaa3' // Storage Table Data Contributor
        principalType: 'User'
      }
      {
        principalId: principalId
        roleDefinitionIdOrName: '974c5e8b-45b9-4653-ba55-5f855dd0fb88' // Storage Queue Data Contributor
        principalType: 'User'
      }
    ]
  }
}

// App Service Plan using AVM - Flex Consumption
module appServicePlan 'br/public:avm/res/web/serverfarm:0.5.0' = {
  name: 'appserviceplan-${resourceToken}'
  scope: rg
  params: {
    name: !empty(appServicePlanName) ? appServicePlanName : '${abbrs.webServerFarms}${resourceToken}'
    location: location
    tags: tags
    skuName: 'FC1' // Flex Consumption
    reserved: true
  }
}

// Function App using AVM - Flex Consumption
module api 'br/public:avm/res/web/site:0.19.3' = {
  name: 'api-${resourceToken}'
  scope: rg
  params: {
    name: functionAppName
    location: location
    tags: union(tags, { 'azd-service-name': 'api' })
    kind: 'functionapp,linux'
    serverFarmResourceId: appServicePlan.outputs.resourceId
    managedIdentities: {
      userAssignedResourceIds: [apiUserAssignedIdentity.outputs.resourceId]
    }
    functionAppConfig: {
      deployment: {
        storage: {
          type: 'blobContainer'
          value: 'https://${storage.outputs.name}.blob.${environment().suffixes.storage}/${deploymentStorageContainerName}'
          authentication: {
            type: 'UserAssignedIdentity'
            userAssignedIdentityResourceId: apiUserAssignedIdentity.outputs.resourceId
          }
        }
      }
      scaleAndConcurrency: {
        instanceMemoryMB: 2048
        maximumInstanceCount: 100
      }
      runtime: {
        name: 'dotnet-isolated'
        version: '9.0'
      }
    }
    virtualNetworkSubnetResourceId: skipVnet ? '' : '${serviceVirtualNetwork!.outputs.resourceId}/subnets/app-subnet'
    diagnosticSettings: [
      {
        workspaceResourceId: logAnalytics.outputs.resourceId
      }
    ]
    siteConfig: {
      alwaysOn: false
      cors: {
        allowedOrigins: [ webUri, 'https://${webapp.outputs.defaultHostname}' ]
      }
      appSettings: [
        { name: 'AzureWebJobsStorage__credential', value: 'managedidentity' }
        { name: 'AzureWebJobsStorage__clientId', value: apiUserAssignedIdentity.outputs.clientId }
        { name: 'AzureWebJobsStorage__blobServiceUri', value: 'https://${storage.outputs.name}.blob.${environment().suffixes.storage}' }
        { name: 'AzureWebJobsStorage__queueServiceUri', value: 'https://${storage.outputs.name}.queue.${environment().suffixes.storage}' }
        { name: 'AzureWebJobsStorage__tableServiceUri', value: 'https://${storage.outputs.name}.table.${environment().suffixes.storage}' }
        { name: 'AzureWebJobsStorage__accountName', value: storage.outputs.name }
        { name: 'DURABLE_TASK_SCHEDULER_CONNECTION_STRING', value: 'Endpoint=${dts.outputs.dts_URL};Authentication=ManagedIdentity;ClientID=${apiUserAssignedIdentity.outputs.clientId}' }
        { name: 'TASKHUB_NAME', value: dts.outputs.TASKHUB_NAME }
        { name: 'AZURE_OPENAI_ENDPOINT', value: aiServiceExists ? reference(aiServiceAccountResourceId, '2023-05-01').endpoint : aiServices!.outputs.endpoint }
        { name: 'AZURE_OPENAI_DEPLOYMENT_NAME', value: modelName }
        { name: 'AZURE_CLIENT_ID', value: apiUserAssignedIdentity.outputs.clientId }
        { name: 'APPLICATIONINSIGHTS_AUTHENTICATION_STRING', value: 'ClientId=${apiUserAssignedIdentity.outputs.clientId};Authorization=AAD' }
        { name: 'APPLICATIONINSIGHTS_CONNECTION_STRING', value: monitoring.outputs.connectionString }
      ]
    }
  }
  dependsOn: [
    storageQueueRoleApi  // Waits for queue role which includes AI Project reference
  ]
}

// AI-specific dependent resources using AVM modules
var aiStorageExists = aiStorageAccountResourceId != ''
var aiServiceExists = aiServiceAccountResourceId != ''
var acsExists = aiSearchServiceResourceId != ''
var aiStorageName = 'stai${aiResourceToken}'
var aiServiceName = '${aiServicesName}${aiResourceToken}'
var aiSearchNameFinal = '${aiSearchName}${aiResourceToken}'

// AI Storage Account using AVM (if not provided)
module aiStorage 'br/public:avm/res/storage/storage-account:0.29.0' = if (!aiStorageExists) {
  scope: rg
  name: 'aiStorage-${aiResourceToken}'
  params: {
    name: aiStorageName
    location: modelLocation
    tags: tags
    kind: 'StorageV2'
    skuName: 'Standard_LRS'
    minimumTlsVersion: 'TLS1_2'
    allowBlobPublicAccess: false
    publicNetworkAccess: 'Enabled'
    allowSharedKeyAccess: false
    networkAcls: {
      bypass: 'AzureServices'
      defaultAction: 'Allow'
    }
  }
}

// Key Vault using AVM
module aiKeyVault 'br/public:avm/res/key-vault/vault:0.9.0' = {
  scope: rg
  name: 'aiKeyVault-${aiResourceToken}'
  params: {
    name: 'kv-${name}${resourceToken}'
    location: modelLocation
    tags: tags
    sku: 'standard'
    enableVaultForDeployment: false
    enableVaultForDiskEncryption: false
    enableVaultForTemplateDeployment: false
    enableSoftDelete: true
    enableRbacAuthorization: true
    softDeleteRetentionInDays: 7
    networkAcls: {
      bypass: 'AzureServices'
      defaultAction: 'Deny'
    }
  }
}

// AI Services (Cognitive Services) using AVM with model deployment
module aiServices 'br/public:avm/res/cognitive-services/account:0.9.2' = if (!aiServiceExists) {
  scope: rg
  name: 'aiServices-${aiResourceToken}'
  params: {
    name: aiServiceName
    location: modelLocation
    tags: tags
    kind: 'AIServices'
    customSubDomainName: toLower(aiServiceName)
    publicNetworkAccess: 'Enabled'
    disableLocalAuth: true
    sku: modelSkuName
    deployments: [
      {
        name: modelName
        model: {
          format: modelFormat
          name: modelName
          version: modelVersion
        }
        sku: {
          name: 'GlobalStandard'
          capacity: modelCapacity
        }
      }
    ]
    roleAssignments: [
      {
        principalId: apiUserAssignedIdentity.outputs.principalId
        roleDefinitionIdOrName: '5e0bd9bd-7b93-4f28-af87-19fc36ad61bd' // Cognitive Services OpenAI User
        principalType: 'ServicePrincipal'
      }
    ]
  }
}

// AI Search using AVM
module aiSearch 'br/public:avm/res/search/search-service:0.9.2' = if (!acsExists) {
  scope: rg
  name: 'aiSearch-${aiResourceToken}'
  params: {
    name: aiSearchNameFinal
    location: modelLocation
    tags: tags
    sku: 'standard'
    authOptions: {
      aadOrApiKey: {
        aadAuthFailureMode: 'http401WithBearerChallenge'
      }
    }
    disableLocalAuth: false
    hostingMode: 'default'
    partitionCount: 1
    replicaCount: 1
    semanticSearch: 'disabled'
  }
}

// AI Hub using AVM
module aiHub 'br/public:avm/res/machine-learning-services/workspace:0.13.0' = {
  scope: rg
  name: '${name}-${aiResourceToken}'
  params: {
    name: '${name}${aiUniqueSuffix}'
    friendlyName: aiHubFriendlyName
    description: aiHubDescription
    location: modelLocation
    tags: tags
    kind: 'Hub'
    sku: 'Basic'
    associatedStorageAccountResourceId: aiStorageExists ? aiStorageAccountResourceId : aiStorage!.outputs.resourceId
    associatedKeyVaultResourceId: aiKeyVault.outputs.resourceId
    connections: [
      {
        name: '${name}${aiUniqueSuffix}-ai'
        category: 'AIServices'
        target: aiServiceExists ? reference(aiServiceAccountResourceId, '2023-05-01').endpoint : aiServices!.outputs.endpoint
        isSharedToAll: true
        connectionProperties: {
          authType: 'AAD'
        }
        metadata: {
          ApiType: 'Azure'
          ResourceId: aiServiceExists ? aiServiceAccountResourceId : aiServices!.outputs.resourceId
          location: modelLocation
        }
      }
      {
        name: '${name}${aiUniqueSuffix}-search'
        category: 'CognitiveSearch'
        target: 'https://${acsExists ? split(aiSearchServiceResourceId, '/')[8] : aiSearchNameFinal}.search.windows.net'
        isSharedToAll: true
        connectionProperties: {
          authType: 'AAD'
        }
        metadata: {
          ApiType: 'Azure'
          ResourceId: acsExists ? aiSearchServiceResourceId : aiSearch!.outputs.resourceId
          location: modelLocation
        }
      }
    ]
  }
}

// Variables for project capability host connections
var projectConnectionString = '${location}.api.azureml.ms;${subscription().subscriptionId};${rg.name};${projectName}${aiUniqueSuffix}'

// AI Project using AVM
module aiProject 'br/public:avm/res/machine-learning-services/workspace:0.13.0' = {
  scope: rg
  name: '${projectName}-${resourceToken}-deployment'
  params: {
    name: '${projectName}${aiUniqueSuffix}'
    friendlyName: aiProjectFriendlyName
    description: aiProjectDescription
    location: modelLocation
    tags: union(tags, {
      ProjectConnectionString: projectConnectionString
    })
    kind: 'Project'
    sku: 'Basic'
    hubResourceId: aiHub.outputs.resourceId
    managedIdentities: {
      systemAssigned: true
    }
    roleAssignments: [
      {
        principalId: apiUserAssignedIdentity.outputs.principalId
        roleDefinitionIdOrName: '64702f94-c441-49e6-a78b-ef80e0188fee' // Azure AI Developer
        principalType: 'ServicePrincipal'
      }
    ]
  }
}

// AI Services role assignments for AI Project using AVM pattern
module aiServiceRoleAssignment1 'br/public:avm/ptn/authorization/resource-role-assignment:0.1.2' = if (!aiServiceExists) {
  name: 'aiServiceRole1-${aiResourceToken}'
  scope: rg
  params: {
    principalId: aiProject.outputs.systemAssignedMIPrincipalId!
    roleDefinitionId: '25fbc0a9-bd7c-42a3-aa1a-3b75d497ee68' // Cognitive Services Contributor
    principalType: 'ServicePrincipal'
    resourceId: aiServices!.outputs.resourceId
  }
}

module aiServiceRoleAssignment2 'br/public:avm/ptn/authorization/resource-role-assignment:0.1.2' = if (!aiServiceExists) {
  scope: rg
  name: 'aiServiceRole2-${aiResourceToken}'
  params: {
    principalId: aiProject.outputs.systemAssignedMIPrincipalId!
    roleDefinitionId: '5e0bd9bd-7b93-4f28-af87-19fc36ad61bd' // Cognitive Services OpenAI User
    principalType: 'ServicePrincipal'
    resourceId: aiServices!.outputs.resourceId
  }
}

module aiServiceRoleAssignment3 'br/public:avm/ptn/authorization/resource-role-assignment:0.1.2' = if (!aiServiceExists) {
  scope: rg
  name: 'aiServiceRole3-${aiResourceToken}'
  params: {
    principalId: aiProject.outputs.systemAssignedMIPrincipalId!
    roleDefinitionId: 'a97b65f3-24c7-4388-baec-2e87135dc908' // Cognitive Services User
    principalType: 'ServicePrincipal'
    resourceId: aiServices!.outputs.resourceId
  }
}

// Role assignment for deployer/user principal to access Azure OpenAI
module aiServiceRoleAssignmentUser 'br/public:avm/ptn/authorization/resource-role-assignment:0.1.2' = if (!aiServiceExists) {
  scope: rg
  name: 'aiServiceRoleUser-${aiResourceToken}'
  params: {
    principalId: principalId
    roleDefinitionId: '5e0bd9bd-7b93-4f28-af87-19fc36ad61bd' // Cognitive Services OpenAI User
    principalType: 'User'
    resourceId: aiServices!.outputs.resourceId
  }
}

// AI Search role assignments for AI Project using AVM pattern
module aiSearchRoleAssignment1 'br/public:avm/ptn/authorization/resource-role-assignment:0.1.2' = if (!acsExists) {
  scope: rg
  name: 'aiSearchRole1-${aiResourceToken}'
  params: {
    principalId: aiProject.outputs.systemAssignedMIPrincipalId!
    roleDefinitionId: '8ebe5a00-799e-43f5-93ac-243d3dce84a7' // Search Index Data Contributor
    principalType: 'ServicePrincipal'
    resourceId: aiSearch!.outputs.resourceId
  }
}

module aiSearchRoleAssignment2 'br/public:avm/ptn/authorization/resource-role-assignment:0.1.2' = if (!acsExists) {
  scope: rg
  name: 'aiSearchRole2-${aiResourceToken}'
  params: {
    principalId: aiProject.outputs.systemAssignedMIPrincipalId!
    roleDefinitionId: '7ca78c08-252a-4471-8644-bb5ff32d4ba0' // Search Service Contributor
    principalType: 'ServicePrincipal'
    resourceId: aiSearch!.outputs.resourceId
  }
}

// Role assignment for deployer/user principal to access AI Search
module aiSearchRoleAssignmentUser 'br/public:avm/ptn/authorization/resource-role-assignment:0.1.2' = if (!acsExists) {
  scope: rg
  name: 'aiSearchRoleUser-${aiResourceToken}'
  params: {
    principalId: principalId
    roleDefinitionId: '8ebe5a00-799e-43f5-93ac-243d3dce84a7' // Search Index Data Contributor
    principalType: 'User'
    resourceId: aiSearch!.outputs.resourceId
  }
}

// aiServiceRoleAssignmentFunctionApp - now inline in aiServices module

// Storage role assignments using AVM pattern
// Blob and table roles now inline in storage module
var storageQueueDataContributorRole = '974c5e8b-45b9-4653-ba55-5f855dd0fb88'

// storageRoleAssignmentApi - now inline in storage module
// storageRoleAssignmentUser - now inline in storage module

module storageQueueRoleApi 'br/public:avm/ptn/authorization/resource-role-assignment:0.1.2' = {
  scope: rg
  name: 'storageQueueApi-${resourceToken}'
  params: {
    principalId: apiUserAssignedIdentity.outputs.principalId
    roleDefinitionId: storageQueueDataContributorRole
    principalType: 'ServicePrincipal'
    resourceId: storage.outputs.resourceId
  }
}

module storageQueueRoleAIProject 'br/public:avm/ptn/authorization/resource-role-assignment:0.1.2' = {
  scope: rg
  name: 'storageQueueAI-${resourceToken}'
  params: {
    principalId: aiProject.outputs.systemAssignedMIPrincipalId!
    roleDefinitionId: storageQueueDataContributorRole
    principalType: 'ServicePrincipal'
    resourceId: storage.outputs.resourceId
  }
}

module storageQueueRoleUser 'br/public:avm/ptn/authorization/resource-role-assignment:0.1.2' = {
  scope: rg
  name: 'storageQueueUser-${resourceToken}'
  params: {
    principalId: principalId
    roleDefinitionId: storageQueueDataContributorRole
    principalType: 'User'
    resourceId: storage.outputs.resourceId
  }
}

// storageTableRoleApi - now inline in storage module

// Virtual Network using AVM
var vnetName = !empty(vNetName) ? vNetName : '${abbrs.networkVirtualNetworks}${resourceToken}'

module serviceVirtualNetwork 'br/public:avm/res/network/virtual-network:0.7.1' = if (!skipVnet) {
  scope: rg
  name: 'vnet-${resourceToken}'
  params: {
    name: vnetName
    location: location
    tags: tags
    addressPrefixes: ['10.0.0.0/16']
    subnets: [
      {
        name: 'app-subnet'
        addressPrefix: '10.0.0.0/24'
        delegation: 'Microsoft.App/environments'
      }
      {
        name: 'pe-subnet'
        addressPrefix: '10.0.1.0/24'
      }
    ]
  }
}

// Private DNS Zone for blob storage
module privateDnsZone 'br/public:avm/res/network/private-dns-zone:0.8.0' = if (!skipVnet) {
  scope: rg
  name: 'pdns-${resourceToken}'
  params: {
    name: 'privatelink.blob.${environment().suffixes.storage}'
    virtualNetworkLinks: [
      {
        virtualNetworkResourceId: serviceVirtualNetwork!.outputs.resourceId
      }
    ]
  }
}

// Private Endpoint for storage using AVM
module storagePrivateEndpoint 'br/public:avm/res/network/private-endpoint:0.11.1' = if (!skipVnet) {
  scope: rg
  name: 'pe-${resourceToken}'
  params: {
    name: 'pe-storage-${resourceToken}'
    location: location
    tags: tags
    subnetResourceId: '${serviceVirtualNetwork!.outputs.resourceId}/subnets/pe-subnet'
    privateLinkServiceConnections: [
      {
        name: 'storage-blob-connection'
        properties: {
          privateLinkServiceId: storage.outputs.resourceId
          groupIds: ['blob']
        }
      }
    ]
    privateDnsZoneGroup: {
      privateDnsZoneGroupConfigs: [
        {
          privateDnsZoneResourceId: privateDnsZone!.outputs.resourceId
        }
      ]
    }
  }
}

// Log Analytics Workspace using AVM
module logAnalytics 'br/public:avm/res/operational-insights/workspace:0.13.0' = {
  scope: rg
  name: 'logs-${resourceToken}'
  params: {
    name: !empty(logAnalyticsName) ? logAnalyticsName : '${abbrs.operationalInsightsWorkspaces}${resourceToken}'
    location: location
    tags: tags
  }
}

// Application Insights using AVM
module monitoring 'br/public:avm/res/insights/component:0.7.1' = {
  scope: rg
  name: 'monitoring-${resourceToken}'
  params: {
    name: !empty(applicationInsightsName) ? applicationInsightsName : '${abbrs.insightsComponents}${resourceToken}'
    location: location
    tags: tags
    workspaceResourceId: logAnalytics.outputs.resourceId
    disableLocalAuth: disableLocalAuth
    roleAssignments: [
      {
        principalId: apiUserAssignedIdentity.outputs.principalId
        roleDefinitionIdOrName: '3913510d-42f4-4e42-8a64-420c390055eb' // Monitoring Metrics Publisher
        principalType: 'ServicePrincipal'
      }
    ]
  }
}

var durableTaskDataContributorRoleDefinitionId = '0ad04412-c4d5-4796-b79c-f76d14c8d402'

module dtsRoleApi 'br/public:avm/ptn/authorization/resource-role-assignment:0.1.2' = {
  scope: rg
  name: 'dtsRoleApi-${resourceToken}'
  params: {
    principalId: apiUserAssignedIdentity.outputs.principalId
    roleDefinitionId: durableTaskDataContributorRoleDefinitionId
    principalType: 'ServicePrincipal'
    resourceId: dts.outputs.dts_ID
  }
}

module dtsRoleUser 'br/public:avm/ptn/authorization/resource-role-assignment:0.1.2' = {
  scope: rg
  name: 'dtsRoleUser-${resourceToken}'
  params: {
    principalId: principalId
    roleDefinitionId: durableTaskDataContributorRoleDefinitionId
    principalType: 'User'
    resourceId: dts.outputs.dts_ID
  }
}

// Durable Task Scheduler doesn't have AVM support yet - 
module dts './app/dts.bicep' = {
  scope: rg 
  name: 'dtsResource-${resourceToken}'
  params: {
    name: !empty(dtsName) ? dtsName : '${abbrs.dts}${resourceToken}'
    taskhubname: !empty(taskHubName) ? taskHubName : '${abbrs.taskhub}${resourceToken}'
    location: location
    tags: tags
    ipAllowlist: [
      '0.0.0.0/0'
    ]
    skuName: dtsSkuName
  }
}

// App outputs
output APPLICATIONINSIGHTS_CONNECTION_STRING string = monitoring.outputs.connectionString
output AZURE_LOCATION string = location
output SERVICE_API_NAME string = api.outputs.name
output SERVICE_API_URI string = 'https://${api.outputs.defaultHostname}'
output AZURE_FUNCTION_APP_NAME string = api.outputs.name
output STATIC_WEB_APP_NAME string = webapp.outputs.name
output STATIC_WEB_APP_URI string = 'https://${webapp.outputs.defaultHostname}'
output PRE_STATIC_WEB_APP_URI string = webAppName
output RESOURCE_GROUP string = rg.name
output PROJECT_CONNECTION_STRING string = projectConnectionString
output STORAGE_CONNECTION__queueServiceUri string = 'https://${storage.outputs.name}.queue.${environment().suffixes.storage}'
output AZURE_OPENAI_ENDPOINT string = aiServiceExists ? reference(aiServiceAccountResourceId, '2023-05-01').endpoint : aiServices!.outputs.endpoint
output AZURE_OPENAI_DEPLOYMENT_NAME string = modelName
