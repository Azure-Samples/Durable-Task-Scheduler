metadata description = 'Creates an Azure Container Apps environment.'
param name string
param location string = resourceGroup().location
param tags object = {}

@description('Subnet resource ID for the Container Apps environment')
param subnetResourceId string = ''

@description('Whether to use an internal or external load balancer')
@allowed(['Internal', 'External'])
param loadBalancerType string = 'External'

resource containerAppsEnvironment 'Microsoft.App/managedEnvironments@2023-05-01' = {
  name: name
  location: location
  tags: tags
  properties: {
    vnetConfiguration: !empty(subnetResourceId) ? {
      infrastructureSubnetId: subnetResourceId
      internal: loadBalancerType == 'Internal'
    } : null
  }
}

output defaultDomain string = containerAppsEnvironment.properties.defaultDomain
output id string = containerAppsEnvironment.id
output name string = containerAppsEnvironment.name
