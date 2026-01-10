// vnet.bicep - Create a virtual network with subnets

@description('Name of the virtual network')
param name string

@description('Location for the virtual network')
param location string = resourceGroup().location

@description('Tags for the resources')
param tags object = {}

@description('Address prefix for the virtual network')
param vnetAddressPrefix string = '10.0.0.0/16'

@description('Address prefix for the infrastructure subnet')
param infrastructureSubnetPrefix string = '10.0.0.0/23'

@description('Address prefix for the app subnet')
param appSubnetPrefix string = '10.0.2.0/23'

var infrastructureSubnetName = 'infrastructure-subnet'
var appSubnetName = 'app-subnet'

resource vnet 'Microsoft.Network/virtualNetworks@2023-05-01' = {
  name: name
  location: location
  tags: tags
  properties: {
    addressSpace: {
      addressPrefixes: [
        vnetAddressPrefix
      ]
    }
    subnets: [
      {
        name: infrastructureSubnetName
        properties: {
          addressPrefix: infrastructureSubnetPrefix
          // Don't pre-delegate the subnet - Container Apps will handle this
        }
      }
      {
        name: appSubnetName
        properties: {
          addressPrefix: appSubnetPrefix
        }
      }
    ]
  }
}

output vnetId string = vnet.id
output vnetName string = vnet.name
output infrastructureSubnetName string = infrastructureSubnetName
output infrastructureSubnetId string = resourceId('Microsoft.Network/virtualNetworks/subnets', name, infrastructureSubnetName)
output appSubnetName string = appSubnetName
output appSubnetId string = resourceId('Microsoft.Network/virtualNetworks/subnets', name, appSubnetName)
