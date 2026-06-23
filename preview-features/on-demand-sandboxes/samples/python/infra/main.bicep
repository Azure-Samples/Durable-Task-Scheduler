targetScope = 'subscription'

// Provisions the Azure resources to run the On-demand Sandboxes code-interpreter demo
// in the cloud: the orchestrator (main-app) runs on Azure Container Apps, and DTS starts
// the sandbox worker image on demand. The Durable Task Scheduler is NOT created here — it
// is passed in as an existing resource (schedulerName + schedulerResourceGroupName)
// because it is patched out of band to enable the On-demand Sandboxes preview feature.

@minLength(1)
@maxLength(64)
@description('Name of the environment which is used to generate a short unique hash used in all resources.')
param environmentName string

@minLength(1)
@description('Primary location for all resources')
param location string

@description('Id of the user or app to assign application roles')
param principalId string = ''

// Container Apps parameters
param containerAppsEnvironmentName string = ''
param mainAppContainerAppName string = ''

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
//  - the Container App authenticates to DTS and Azure OpenAI
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

// The sandbox worker image DTS starts on demand. The reference is deterministic so it can
// be set on the Container App at provision time; scripts/acr-build.sh pushes the sandbox
// image to this exact repository and tag.
var sandboxImage = '${containerRegistry.outputs.loginServer}/dts-ondemand-sandboxes/sandbox-worker-${environmentName}:latest'

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
// Container Apps
// ============================

module containerAppsEnvironment './core/host/container-apps-environment.bicep' = {
  name: 'container-apps-environment'
  scope: rg
  params: {
    name: !empty(containerAppsEnvironmentName) ? containerAppsEnvironmentName : '${abbrs.appManagedEnvironments}${resourceToken}'
    location: location
    tags: tags
  }
}

// The orchestrator (main-app). azd builds and pushes the image, then updates this app.
// The container is a one-shot demo client; it is restarted by Container Apps after each
// run, matching the previous AKS Deployment behavior.
module mainApp './core/host/container-app.bicep' = {
  name: 'mainapp'
  scope: rg
  params: {
    name: !empty(mainAppContainerAppName) ? mainAppContainerAppName : '${abbrs.appContainerApps}${resourceToken}-mainapp'
    location: location
    tags: union(tags, { 'azd-service-name': mainAppServiceName })
    containerAppsEnvironmentName: containerAppsEnvironment.outputs.name
    containerRegistryName: containerRegistry.outputs.name
    identityName: identity.outputs.name
    ingressEnabled: false
    env: [
      {
        name: 'DTS_ENDPOINT'
        value: schedulerAccess.outputs.endpoint
      }
      {
        name: 'DTS_TASK_HUB'
        value: schedulerAccess.outputs.taskHubName
      }
      {
        name: 'AOAI_ENDPOINT'
        value: openAi.outputs.endpoint
      }
      {
        name: 'AOAI_DEPLOYMENT'
        value: openAi.outputs.chatDeploymentName
      }
      {
        name: 'AZURE_CLIENT_ID'
        value: identity.outputs.clientId
      }
      // The sandbox worker profile (main_app.py) reads these.
      {
        name: 'DTS_SANDBOX_CONTAINER_IMAGE'
        value: sandboxImage
      }
      {
        name: 'DTS_SANDBOX_IMAGE_PULL_UMI_CLIENT_ID'
        value: identity.outputs.clientId
      }
      {
        name: 'DTS_SANDBOX_SCHEDULER_UMI_CLIENT_ID'
        value: identity.outputs.clientId
      }
    ]
  }
}

// ============================
// Outputs
// ============================

output AZURE_LOCATION string = location
output AZURE_TENANT_ID string = tenant().tenantId

output AZURE_CONTAINER_REGISTRY_ENDPOINT string = containerRegistry.outputs.loginServer
output AZURE_CONTAINER_REGISTRY_NAME string = containerRegistry.outputs.name

output AZURE_CONTAINER_APPS_ENVIRONMENT_NAME string = containerAppsEnvironment.outputs.name

output AZURE_USER_ASSIGNED_IDENTITY_NAME string = identity.outputs.name
output AZURE_USER_ASSIGNED_IDENTITY_CLIENT_ID string = identity.outputs.clientId
output AZURE_USER_ASSIGNED_IDENTITY_RESOURCE_ID string = identity.outputs.resourceId

// Scheduler details (DTS_ENDPOINT/DTS_TASK_HUB feed the app; name/RG feed the
// postprovision identity-attach hook).
output DTS_ENDPOINT string = schedulerAccess.outputs.endpoint
output DTS_TASK_HUB string = schedulerAccess.outputs.taskHubName
output DTS_SCHEDULER_NAME string = schedulerName
output DTS_SCHEDULER_RESOURCE_GROUP string = schedulerResourceGroupName

// Full sandbox worker image reference (DTS starts this on demand). scripts/acr-build.sh
// must push the sandbox image to this exact reference.
output DTS_SANDBOX_CONTAINER_IMAGE string = sandboxImage

output AOAI_ENDPOINT string = openAi.outputs.endpoint
output AOAI_DEPLOYMENT string = openAi.outputs.chatDeploymentName
