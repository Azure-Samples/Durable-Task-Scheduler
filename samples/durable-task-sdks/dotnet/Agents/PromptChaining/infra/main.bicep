targetScope = 'subscription'

// The main bicep module to provision Azure resources.
// For a more complete walkthrough to understand how this file works with azd,
// see https://learn.microsoft.com/en-us/azure/developer/azure-developer-cli/make-azd-compatible?pivots=azd-create

// For a more complete walkthrough to understand how this file works with azd,
// see https://learn.microsoft.com/en-us/azure/developer/azure-developer-cli/make-azd-compatible?pivots=azd-create

@minLength(1)
@maxLength(64)
@description('Name of the the environment which is used to generate a short unique hash used in all resources.')
param environmentName string

@minLength(1)
@description('Primary location for all resources')
param location string

@description('Id of the user identity to be used for testing and debugging. This is not required in production. Leave empty if not needed.')
param principalId string = deployer().objectId

@description('Name for the AI project resources.')
param aiProjectName string = 'project-demo'

@description('Friendly name for your Azure AI resource')
param aiProjectFriendlyName string = 'Agents Project resource'

@description('Description of your Azure AI resource displayed in AI studio')
param aiProjectDescription string = 'This is an example AI Project resource for use in Azure AI Studio.'

@description('Name of the Azure AI Search account')
param aiSearchName string = 'agent-ai-search'

@description('Name for capabilityHost.')
param accountCapabilityHostName string = 'caphostacc'

@description('Name for capabilityHost.')
param projectCapabilityHostName string = 'caphostproj'

@description('Name of the Azure AI Services account')
param aiServicesName string = 'agent-ai-services'

@description('Model name for deployment')
param modelName string = 'gpt-4.1-mini'

@description('Model format for deployment')
param modelFormat string = 'OpenAI'

@description('Model version for deployment')
param modelVersion string = '2025-04-14'

@description('Model deployment SKU name')
param modelSkuName string = 'GlobalStandard'

@description('Model deployment capacity')
param modelCapacity int = 50

@description('Name of the Cosmos DB account for agent thread storage')
param cosmosDbName string = 'agent-ai-cosmos'

@description('The AI Service Account full ARM Resource ID. This is an optional field, and if not provided, the resource will be created.')
param aiServiceAccountResourceId string = ''

@description('The Ai Search Service full ARM Resource ID. This is an optional field, and if not provided, the resource will be created.')
param aiSearchServiceResourceId string = ''

@description('The Ai Storage Account full ARM Resource ID. This is an optional field, and if not provided, the resource will be created.')
param aiStorageAccountResourceId string = ''

@description('The Cosmos DB Account full ARM Resource ID. This is an optional field, and if not provided, the resource will be created.')
param aiCosmosDbAccountResourceId string = ''

var projectName = toLower('${aiProjectName}')
param vnetEnabled bool

param containerAppsEnvName string = ''
param containerAppsAppName string = ''
param containerRegistryName string = ''
param dtsLocation string = 'centralus'
param dtsSkuName string = 'Dedicated'
param dtsCapacity int = 1
param dtsName string = ''
param taskHubName string = ''
var deploymentStorageContainerName = 'app-package-${take(workerServiceName, 32)}-${take(toLower(uniqueString(workerServiceName, resourceToken)), 7)}'
param storageAccountName string = ''
param clientsServiceName string = 'client'
param workerServiceName string = 'worker'
// Create a short, unique suffix, that will be unique to each resource group
var uniqueSuffix = toLower(uniqueString(subscription().id, environmentName, location))

// Optional parameters to override the default azd resource naming conventions.
// Add the following to main.parameters.json to provide values:
// "resourceGroupName": {
//      "value": "myGroupName"
// }
param resourceGroupName string = ''

var abbrs = loadJsonContent('./abbreviations.json')

// tags that should be applied to all resources.
var tags = {
  // Tag all resources with the environment name.
  'azd-env-name': environmentName
}

// Generate a unique token to be used in naming resources.
// Remove linter suppression after using.
#disable-next-line no-unused-vars
var resourceToken = toLower(uniqueString(subscription().id, environmentName, location))

// Organize resources in a resource group
resource rg 'Microsoft.Resources/resourceGroups@2021-04-01' = {
  name: !empty(resourceGroupName) ? resourceGroupName : '${abbrs.resourcesResourceGroups}${environmentName}'
  location: location
  tags: tags
}

// Add resources to be provisioned below.
// A full example that leverages azd bicep modules can be seen in the todo-python-mongo template:
// https://github.com/Azure-Samples/todo-python-mongo/tree/main/infra

// Create a user assigned identity
module identity './app/user-assigned-identity.bicep' = {
  name: 'identity'
  scope: rg
  params: {
    name: 'dts-ca-identity'
  }
}

module identityAssignDTS './core/security/role.bicep' = {
  name: 'identityAssignDTS'
  scope: rg
  params: {
    principalId: identity.outputs.principalId
    roleDefinitionId: '0ad04412-c4d5-4796-b79c-f76d14c8d402'
    principalType: 'ServicePrincipal'
  }
}

module identityAssignDTSDash './core/security/role.bicep' = {
  name: 'identityAssignDTSDash'
  scope: rg
  params: {
    principalId: principalId
    roleDefinitionId: '0ad04412-c4d5-4796-b79c-f76d14c8d402'
    principalType: 'User'
  }
}

// Create virtual network with subnets for Container Apps
module vnet './core/networking/vnet.bicep' = {
  name: 'vnet'
  scope: rg
  params: {
    name: '${abbrs.networkVirtualNetworks}${resourceToken}'
    location: location
    tags: tags
  }
}

// Container apps env and registry
module containerAppsEnv './core/host/container-apps.bicep' = {
  name: 'container-apps'
  scope: rg
  params: {
    name: 'app'
    containerAppsEnvironmentName: !empty(containerAppsEnvName) ? containerAppsEnvName : '${abbrs.appManagedEnvironments}${resourceToken}'
    containerRegistryName: !empty(containerRegistryName) ? containerRegistryName : '${abbrs.containerRegistryRegistries}${resourceToken}'
    location: location
    tags: tags
    internal: false
    vnetId: vnet.outputs.vnetId
    infrastructureSubnetName: 'infrastructure-subnet'
    infrastructureSubnetId: vnet.outputs.infrastructureSubnetId
  }
}

module dts './app/dts.bicep' = {
  scope: rg
  name: 'dtsResource'
  params: {
    name: !empty(dtsName) ? dtsName : '${abbrs.dts}${resourceToken}'
    taskhubname: !empty(taskHubName) ? taskHubName : '${abbrs.taskhub}${resourceToken}'
    location: dtsLocation
    tags: tags
    ipAllowlist: [
      '0.0.0.0/0'
    ]
    skuName: dtsSkuName
    skuCapacity: dtsCapacity
  }
}


// Client registry access must be deployed before client container app
module clientRegistryAccess 'app/registry-access.bicep' = {
  name: 'client-registry-access'
  scope: rg
  params: {
    containerRegistryName: containerAppsEnv.outputs.registryName
    principalID: identity.outputs.principalId
    principalType: 'ServicePrincipal'
  }
}

// Container app
module client 'app/app.bicep' = {
  name: clientsServiceName
  scope: rg
  params: {
    appName: !empty(containerAppsAppName) ? '${containerAppsAppName}-client' : '${abbrs.appContainerApps}${resourceToken}-client'
    containerAppsEnvironmentName: containerAppsEnv.outputs.environmentName
    containerRegistryName: containerAppsEnv.outputs.registryName
    userAssignedManagedIdentity: {
      resourceId: identity.outputs.resourceId
      clientId: identity.outputs.clientId
    }
    location: location
    tags: tags
    serviceName: 'client'
    identityName: identity.outputs.name
    dtsEndpoint: dts.outputs.dts_URL
    taskHubName: dts.outputs.TASKHUB_NAME
    AGENT_CONNECTION_STRING: aiProject.outputs.projectEndpoint
    OPENAI_DEPLOYMENT_NAME: modelName
    AGENT_CONNECTION_STRING__clientId: identity.outputs.clientId
    clientBaseUrl: '' // Not used by client
  }
  dependsOn: [
    clientRegistryAccess // Make sure registry access is set up before deploying the app
  ]
}

// Give the worker access to ACR
module workerRegistryAccess 'app/registry-access.bicep' = {
  name: 'worker-registry-access'
  scope: rg
  params: {
    containerRegistryName: containerAppsEnv.outputs.registryName
    principalID: identity.outputs.principalId
    principalType: 'ServicePrincipal'
  }
}

  // Container app
module worker 'app/app.bicep' = {
  name: workerServiceName
  scope: rg
  params: {
    appName: !empty(containerAppsAppName) ? '${containerAppsAppName}-worker' : '${abbrs.appContainerApps}${resourceToken}-worker'
    containerAppsEnvironmentName: containerAppsEnv.outputs.environmentName
    containerRegistryName: containerAppsEnv.outputs.registryName
    userAssignedManagedIdentity: {
      resourceId: identity.outputs.resourceId
      clientId: identity.outputs.clientId
    }
    location: location
    tags: tags
    serviceName: 'worker'
    identityName: identity.outputs.name
    dtsEndpoint: dts.outputs.dts_URL
    taskHubName: dts.outputs.TASKHUB_NAME
    // Use the connection string in URL format for direct client usage
    AGENT_CONNECTION_STRING: aiProject.outputs.projectEndpoint
    OPENAI_DEPLOYMENT_NAME: modelName
    AGENT_CONNECTION_STRING__clientId: identity.outputs.clientId
    DALLE_ENDPOINT: aiDependencies.outputs.dalleEndpoint
    clientBaseUrl: 'https://${client.outputs.endpoint}'
  }
}

// Dependent resources for the Azure Machine Learning workspace
module aiDependencies './agent/standard-dependent-resources.bicep' = {
  name: 'dependencies${projectName}${uniqueSuffix}deployment'
  scope: rg
  params: {
    location: location
    storageName: 'stai${uniqueSuffix}'
    aiServicesName: '${aiServicesName}${uniqueSuffix}'
    aiSearchName: '${aiSearchName}${uniqueSuffix}'
    cosmosDbName: '${cosmosDbName}${uniqueSuffix}'
    tags: tags

     // Model deployment parameters
     modelName: modelName
     modelFormat: modelFormat
     modelVersion: modelVersion
     modelSkuName: modelSkuName
     modelCapacity: modelCapacity  
     modelLocation: location

     aiServiceAccountResourceId: aiServiceAccountResourceId
     aiSearchServiceResourceId: aiSearchServiceResourceId
     aiStorageAccountResourceId: aiStorageAccountResourceId
     aiCosmosDbAccountResourceId: aiCosmosDbAccountResourceId
    }
}

module aiProject './agent/standard-ai-project.bicep' = {
  name: '${projectName}${uniqueSuffix}deployment'
  scope: rg
  params: {
    // workspace organization
    aiServicesAccountName: aiDependencies.outputs.aiServicesName
    aiProjectName: '${projectName}${uniqueSuffix}'
    aiProjectFriendlyName: aiProjectFriendlyName
    aiProjectDescription: aiProjectDescription
    location: location
    tags: tags
    
    // dependent resources
    aiSearchName: aiDependencies.outputs.aiSearchName
    aiSearchSubscriptionId: aiDependencies.outputs.aiSearchServiceSubscriptionId
    aiSearchResourceGroupName: aiDependencies.outputs.aiSearchServiceResourceGroupName
    storageAccountName: aiDependencies.outputs.storageAccountName
    storageAccountSubscriptionId: aiDependencies.outputs.storageAccountSubscriptionId
    storageAccountResourceGroupName: aiDependencies.outputs.storageAccountResourceGroupName
    cosmosDbAccountName: aiDependencies.outputs.cosmosDbAccountName
    cosmosDbAccountSubscriptionId: aiDependencies.outputs.cosmosDbAccountSubscriptionId
    cosmosDbAccountResourceGroupName: aiDependencies.outputs.cosmosDbAccountResourceGroupName
  }
}

module projectRoleAssignments './agent/standard-ai-project-role-assignments.bicep' = {
  name: 'aiprojectroleassignments${projectName}${uniqueSuffix}deployment'
  scope: rg
  params: {
    aiProjectPrincipalId: aiProject.outputs.aiProjectPrincipalId
    userPrincipalId: principalId
    allowUserIdentityPrincipal: true // Enable user identity role assignments
    aiServicesName: aiDependencies.outputs.aiServicesName
    aiSearchName: aiDependencies.outputs.aiSearchName
    aiCosmosDbName: aiDependencies.outputs.cosmosDbAccountName
    integrationStorageAccountName: storage.outputs.name
    aiStorageAccountName: aiDependencies.outputs.storageAccountName
    functionAppManagedIdentityPrincipalId: identity.outputs.principalId
    allowFunctionAppIdentityPrincipal: true // Enable function app identity role assignments
    dalleAiServicesId: aiDependencies.outputs.dalleAiServicesId
  }
}

module aiProjectCapabilityHost './agent/standard-ai-project-capability-host.bicep' = {
  name: 'capabilityhost${projectName}${uniqueSuffix}deployment'
  scope: rg
  params: {
    aiServicesAccountName: aiDependencies.outputs.aiServicesName
    projectName: aiProject.outputs.aiProjectName
    aiSearchConnection: aiProject.outputs.aiSearchConnection
    azureStorageConnection: aiProject.outputs.azureStorageConnection
    cosmosDbConnection: aiProject.outputs.cosmosDbConnection

    accountCapHost: '${accountCapabilityHostName}${uniqueSuffix}'
    projectCapHost: '${projectCapabilityHostName}${uniqueSuffix}'
  }
  dependsOn: [ projectRoleAssignments ]
}

module postCapabilityHostCreationRoleAssignments './agent/post-capability-host-role-assignments.bicep' = {
  name: 'postcaphostra${projectName}${uniqueSuffix}deployment'
  scope: rg
  params: {
    aiProjectPrincipalId: aiProject.outputs.aiProjectPrincipalId
    aiProjectWorkspaceId: aiProject.outputs.projectWorkspaceId
    aiStorageAccountName: aiDependencies.outputs.storageAccountName
    cosmosDbAccountName: aiDependencies.outputs.cosmosDbAccountName
  }
  dependsOn: [ aiProjectCapabilityHost ]
}

// Backing storage for Azure functions backend API
module storage 'br/public:avm/res/storage/storage-account:0.8.3' = {
  name: 'storage'
  scope: rg
  params: {
    name: !empty(storageAccountName) ? storageAccountName : '${abbrs.storageStorageAccounts}${resourceToken}'
    allowBlobPublicAccess: false
    allowSharedKeyAccess: false // Disable local authentication methods as per policy
    dnsEndpointType: 'Standard'
    publicNetworkAccess: vnetEnabled ? 'Disabled' : 'Enabled'
    // Explicitly disable the property that can't be updated
    requireInfrastructureEncryption: false
    // When vNet is enabled, restrict access but allow Azure services and specifically grant access to the AI Agent service
    networkAcls: vnetEnabled ? {
      defaultAction: 'Deny'
      bypass: 'AzureServices' // Allow Azure services including AI Agent service
      resourceAccessRules: [
        {
          tenantId: tenant().tenantId
          resourceId: aiDependencies.outputs.aiservicesID // Grant explicit access to AI Agent service
        }
      ]
    } : {
      defaultAction: 'Allow'
      bypass: 'AzureServices'
    }
    blobServices: {
      containers: [{name: deploymentStorageContainerName}]
    }
    queueServices: {
      queues: [
        { name: 'input' }
        { name: 'output' }
      ]
    }
    minimumTlsVersion: 'TLS1_2'  // Enforcing TLS 1.2 for better security
    location: location
    tags: tags
  }
}


output AZURE_LOCATION string = location
output AZURE_TENANT_ID string = tenant().tenantId
// Container outputs
output AZURE_CONTAINER_REGISTRY_ENDPOINT string = containerAppsEnv.outputs.registryLoginServer
output AZURE_CONTAINER_REGISTRY_NAME string = containerAppsEnv.outputs.registryName

// Application outputs
output AZURE_CONTAINER_APP_ENDPOINT string = client.outputs.endpoint
output AZURE_CONTAINER_ENVIRONMENT_NAME string = client.outputs.envName
output DTS_URL string = dts.outputs.dts_URL
output TASKHUB_NAME string = dts.outputs.TASKHUB_NAME

// AI Project outputs
output AI_PROJECT_NAME string = aiProject.outputs.aiProjectName

// Identity outputs
output AZURE_USER_ASSIGNED_IDENTITY_NAME string = identity.outputs.name
