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
    name: 'aks-subnet'
    properties: {
      addressPrefix: '10.0.0.0/21'
      delegations: []
      privateEndpointNetworkPolicies: 'Disabled'
      privateLinkServiceNetworkPolicies: 'Enabled'
    }
  }
]

resource vnet 'Microsoft.Network/virtualNetworks@2023-11-01' = {
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
output aksSubnetId string = resourceId('Microsoft.Network/virtualNetworks/subnets', name, 'aks-subnet')
