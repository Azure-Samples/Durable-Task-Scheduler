// filepath: /Users/nickgreenfield1/workspace/Durable-Task-Scheduler/samples/durable-task-sdks/dotnet/FunctionChaining/infra/core/networking/vnet.bicep
@description('The name of the Virtual Network')
param name string

@description('The Azure region where the Virtual Network should exist')
param location string = resourceGroup().location

@description('Optional tags for the resources')
param tags object = {}

@description('The address prefixes of the Virtual Network')
param addressPrefixes array = ['10.0.0.0/16']

@description('The subnets to create in the Virtual Network')
param subnets array = [
  {
    name: 'infrastructure-subnet'
    properties: {
      addressPrefix: '10.0.0.0/21'
      // Container Apps environments don't need pre-configured delegations - they handle this themselves
      delegations: []
      privateEndpointNetworkPolicies: 'Disabled'
      privateLinkServiceNetworkPolicies: 'Enabled'
    }
  }
  {
    name: 'workload-subnet'
    properties: {
      addressPrefix: '10.0.8.0/21'
      delegations: []
      privateEndpointNetworkPolicies: 'Disabled'
      privateLinkServiceNetworkPolicies: 'Enabled'
    }
  }
]

resource vnet 'Microsoft.Network/virtualNetworks@2022-07-01' = {
  name: name
  location: location
  tags: tags
  properties: {
    addressSpace: {
      addressPrefixes: addressPrefixes
    }
    subnets: subnets
  }
}

output id string = vnet.id
output name string = vnet.name
output infrastructureSubnetId string = resourceId('Microsoft.Network/virtualNetworks/subnets', name, 'infrastructure-subnet')
output workloadSubnetId string = resourceId('Microsoft.Network/virtualNetworks/subnets', name, 'workload-subnet')
