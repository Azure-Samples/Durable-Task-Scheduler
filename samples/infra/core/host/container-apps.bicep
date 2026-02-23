metadata description = 'Creates an Azure Container Registry and an Azure Container Apps environment.'
param name string
param location string = resourceGroup().location
param tags object = {}

param containerAppsEnvironmentName string
param containerRegistryName string
param containerRegistryResourceGroupName string = ''
param containerRegistryAdminUserEnabled bool = false
param logAnalyticsWorkspaceName string = ''
param applicationInsightsName string = ''
param daprEnabled bool = false

// Virtual network and subnet parameters
param subnetResourceId string = ''
param loadBalancerType string = 'External'

module containerAppsEnvironment 'container-apps-environment.bicep' = {
  name: '${name}-container-apps-environment'
  params: {
    name: containerAppsEnvironmentName
    location: location
    tags: tags
    logAnalyticsWorkspaceName: logAnalyticsWorkspaceName
    applicationInsightsName: applicationInsightsName
    daprEnabled: daprEnabled
    subnetResourceId: subnetResourceId
    loadBalancerType: loadBalancerType
  }
}

module containerRegistryCurrentRg 'container-registry.bicep' = if (empty(containerRegistryResourceGroupName)) {
  name: '${name}-container-registry-current-rg'
  scope: resourceGroup()
  params: {
    name: containerRegistryName
    location: location
    adminUserEnabled: containerRegistryAdminUserEnabled
    tags: tags
    sku: {
      name: 'Standard'
    }
    anonymousPullEnabled: false
  }
}

module containerRegistryCustomRg 'container-registry.bicep' = if (!empty(containerRegistryResourceGroupName)) {
  name: '${name}-container-registry-custom-rg'
  scope: resourceGroup(containerRegistryResourceGroupName)
  params: {
    name: containerRegistryName
    location: location
    adminUserEnabled: containerRegistryAdminUserEnabled
    tags: tags
    sku: {
      name: 'Standard'
    }
    anonymousPullEnabled: false
  }
}

output defaultDomain string = containerAppsEnvironment.outputs.defaultDomain
output environmentName string = containerAppsEnvironment.outputs.name
output environmentId string = containerAppsEnvironment.outputs.id

output registryLoginServer string = empty(containerRegistryResourceGroupName) ? containerRegistryCurrentRg.outputs.loginServer : containerRegistryCustomRg.outputs.loginServer
output registryName string = empty(containerRegistryResourceGroupName) ? containerRegistryCurrentRg.outputs.name : containerRegistryCustomRg.outputs.name
