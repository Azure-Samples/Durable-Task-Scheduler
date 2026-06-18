targetScope = 'subscription'

// Provisions the Azure resources to run the On-demand Sandboxes code-interpreter demo
// in the cloud: the orchestrator (main-app) runs on AKS, and DTS starts the sandbox
// worker image on demand. The Durable Task Scheduler is NOT created here — it is passed
// in as an existing resource (schedulerName + schedulerResourceGroupName) because it is
// patched out of band to enable the On-demand Sandboxes preview feature.

@minLength(1)
@maxLength(64)
@description('Name of the environment which is used to generate a short unique hash used in all resources.')
param environmentName string

@minLength(1)
@description('Primary location for all resources')
param location string

@description('Id of the user or app to assign application roles')
param principalId string = ''

// AKS parameters
param aksClusterName string = ''
param kubernetesVersion string = '1.32'
param aksVmSize string = 'standard_d4s_v5'
param aksNodeCount int = 2

// Container registry parameters
param containerRegistryName string = ''

// Existing Durable Task Scheduler (created + patched out of band for the preview).
@description('Name of the existing Durable Task Scheduler to use as the durable backend.')
param schedulerName string

@description('Resource group that contains the existing Durable Task Scheduler.')
param schedulerResourceGroupName string

@description('Task hub to use on the scheduler (created if it does not exist).')
param taskHubName string = 'default'

// Azure OpenAI parameters (used by the in-process GenerateCode activity).
param openAiServiceName string = ''
param openAiLocation string = 'eastus'
param chatDeploymentName string = 'gpt-5.1'
param chatModelName string = 'gpt-5.1'
param chatModelVersion string = '2025-11-13'
param chatDeploymentSkuName string = 'GlobalStandard'
param chatDeploymentCapacity int = 30

// Service name (must match the service name in azure.yaml).
param mainAppServiceName string = 'mainapp'

// Optional resource group name override
param resourceGroupName string = ''

var abbrs = loadJsonContent('./abbreviations.json')

var tags = {
  'azd-env-name': environmentName
}

var resourceToken = toLower(uniqueString(subscription().id, environmentName, location))

resource rg 'Microsoft.Resources/resourceGroups@2021-04-01' = {
  name: !empty(resourceGroupName) ? resourceGroupName : '${abbrs.resourcesResourceGroups}${environmentName}'
  location: location
  tags: tags
}

// ============================
// Identity
// ============================

// A single user-assigned managed identity is used for everything in this sample:
//  - AKS workload identity (the app authenticates to DTS and Azure OpenAI)
//  - DTS image pull for the sandbox (AcrPull on the registry)
//  - the sandbox worker connecting back to DTS
module identity './app/user-assigned-identity.bicep' = {
  scope: rg
  params: {
    name: '${abbrs.managedIdentityUserAssignedIdentities}${resourceToken}'
    location: location
    tags: tags
  }
}

// ============================
// Container Registry
// ============================

module containerRegistry './core/host/container-registry.bicep' = {
  name: 'container-registry'
  scope: rg
  params: {
    name: !empty(containerRegistryName) ? containerRegistryName : '${abbrs.containerRegistryRegistries}${resourceToken}'
    location: location
    tags: tags
    sku: {
      name: 'Standard'
    }
    anonymousPullEnabled: false
  }
}

// Grant the managed identity AcrPull so DTS can pull the sandbox worker image.
module identityAcrPull './core/security/registry-access.bicep' = {
  name: 'identity-acr-pull'
  scope: rg
  params: {
    containerRegistryName: containerRegistry.outputs.name
    principalId: identity.outputs.principalId
  }
}

// ============================
// AKS Cluster
// ============================

module aksCluster './core/host/aks-cluster.bicep' = {
  name: 'aks-cluster'
  scope: rg
  params: {
    name: !empty(aksClusterName) ? aksClusterName : '${abbrs.containerServiceManagedClusters}${resourceToken}'
    location: location
    tags: tags
    kubernetesVersion: kubernetesVersion
    agentVMSize: aksVmSize
    agentCount: aksNodeCount
    containerRegistryName: containerRegistry.outputs.name
  }
}

// ============================
// Workload Identity Federation
// ============================

module federatedIdentityMainApp './app/federated-identity.bicep' = {
  name: 'federated-identity-mainapp'
  scope: rg
  params: {
    identityName: identity.outputs.name
    federatedCredentialName: 'fed-${mainAppServiceName}'
    oidcIssuerUrl: aksCluster.outputs.oidcIssuerUrl
    serviceAccountNamespace: 'default'
    serviceAccountName: mainAppServiceName
  }
}

// ============================
// Azure OpenAI
// ============================

module openAi './app/openai.bicep' = {
  name: 'openai'
  scope: rg
  params: {
    name: !empty(openAiServiceName) ? openAiServiceName : 'aoai-${resourceToken}'
    location: openAiLocation
    tags: tags
    chatDeploymentName: chatDeploymentName
    chatModelName: chatModelName
    chatModelVersion: chatModelVersion
    chatDeploymentSkuName: chatDeploymentSkuName
    chatDeploymentCapacity: chatDeploymentCapacity
    workloadPrincipalId: identity.outputs.principalId
    userPrincipalId: principalId
  }
}

// ============================
// Existing Durable Task Scheduler
// ============================

module schedulerAccess './app/scheduler-access.bicep' = {
  name: 'scheduler-access'
  scope: resourceGroup(schedulerResourceGroupName)
  params: {
    schedulerName: schedulerName
    taskHubName: taskHubName
    workloadPrincipalId: identity.outputs.principalId
    userPrincipalId: principalId
  }
}

// ============================
// Outputs
// ============================

output AZURE_LOCATION string = location
output AZURE_TENANT_ID string = tenant().tenantId

output AZURE_CONTAINER_REGISTRY_ENDPOINT string = containerRegistry.outputs.loginServer
output AZURE_CONTAINER_REGISTRY_NAME string = containerRegistry.outputs.name

output AZURE_AKS_CLUSTER_NAME string = aksCluster.outputs.clusterName

output AZURE_USER_ASSIGNED_IDENTITY_NAME string = identity.outputs.name
output AZURE_USER_ASSIGNED_IDENTITY_CLIENT_ID string = identity.outputs.clientId
output AZURE_USER_ASSIGNED_IDENTITY_RESOURCE_ID string = identity.outputs.resourceId

// Scheduler details (DTS_ENDPOINT/DTS_TASK_HUB feed the app; name/RG feed the
// postprovision identity-attach hook).
output DTS_ENDPOINT string = schedulerAccess.outputs.endpoint
output DTS_TASK_HUB string = schedulerAccess.outputs.taskHubName
output DTS_SCHEDULER_NAME string = schedulerName
output DTS_SCHEDULER_RESOURCE_GROUP string = schedulerResourceGroupName

output AOAI_ENDPOINT string = openAi.outputs.endpoint
output AOAI_DEPLOYMENT string = openAi.outputs.chatDeploymentName
