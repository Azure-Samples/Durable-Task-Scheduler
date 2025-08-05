// container-apps.bicep - Create container apps environment and container registry

@description('The name prefix for resources')
param name string = 'app'

@description('The name of the container apps environment')
param containerAppsEnvironmentName string

@description('The name of the container registry')
param containerRegistryName string

@description('The location of the container apps environment')
param location string = resourceGroup().location

@description('Tags for the resources')
param tags object = {}

@description('Whether the container apps environment should be internal')
param internal bool = false

@description('Resource ID of the virtual network subnet to use for the container apps environment')
param infrastructureSubnetId string = ''

@description('Virtual network resource ID')
param vnetId string = ''

@description('Subnet name for infrastructure components')
param infrastructureSubnetName string = ''

// Create a Log Analytics workspace for the Container Apps environment
resource logAnalyticsWorkspace 'Microsoft.OperationalInsights/workspaces@2022-10-01' = {
  name: '${name}-logs'
  location: location
  tags: tags
  properties: {
    sku: {
      name: 'PerGB2018'
    }
    retentionInDays: 30
    features: {
      enableLogAccessUsingOnlyResourcePermissions: true
    }
    workspaceCapping: {
      dailyQuotaGb: 1
    }
  }
}

// Deploy the Container Apps environment using the module
module containerAppsEnvironment '../host/container-apps-environment.bicep' = {
  name: 'container-apps-environment-deploy'
  params: {
    name: containerAppsEnvironmentName
    location: location
    tags: tags
    logAnalyticsWorkspaceId: logAnalyticsWorkspace.id
    infrastructureSubnetId: infrastructureSubnetId
    vnetId: vnetId
    infrastructureSubnetName: infrastructureSubnetName
    internal: internal
  }
}

resource containerRegistry 'Microsoft.ContainerRegistry/registries@2023-11-01-preview' = {
  name: containerRegistryName
  location: location
  tags: tags
  sku: {
    name: 'Basic'
  }
  properties: {
    adminUserEnabled: false
  }
}

// Output necessary properties
output environmentName string = containerAppsEnvironment.outputs.name
output environmentDefaultDomain string = containerAppsEnvironment.outputs.defaultDomain
output registryName string = containerRegistry.name
output registryLoginServer string = containerRegistry.properties.loginServer
