param name string
param location string = resourceGroup().location
param tags object = {}

@description('Log Analytics workspace resource ID')
param logAnalyticsWorkspaceId string

@description('Virtual network resource ID')
param vnetId string = ''

@description('Subnet name for infrastructure components')
param infrastructureSubnetName string = ''

@description('Existing subnet ID for infrastructure resources. If not specified, a delegated subnet will be created.')
param infrastructureSubnetId string = ''

@description('Whether to create the Container Apps Environment in an internal or external network. Default is external.')
param internal bool = false

// Use existing subnet if provided, otherwise reference it from the VNET
var subnetId = !empty(infrastructureSubnetId) ? infrastructureSubnetId : !empty(vnetId) && !empty(infrastructureSubnetName) ? '${vnetId}/subnets/${infrastructureSubnetName}' : ''

resource environment 'Microsoft.App/managedEnvironments@2023-05-01' = {
  name: name
  location: location
  tags: tags
  properties: {
    appLogsConfiguration: {
      destination: 'log-analytics'
      logAnalyticsConfiguration: {
        customerId: reference(logAnalyticsWorkspaceId, '2021-06-01').customerId
        sharedKey: listKeys(logAnalyticsWorkspaceId, '2021-06-01').primarySharedKey
      }
    }
    vnetConfiguration: !empty(subnetId) ? {
      infrastructureSubnetId: subnetId
      internal: internal
    } : null
  }
}

output id string = environment.id
output name string = environment.name
output defaultDomain string = environment.properties.defaultDomain
output staticIp string = environment.properties.staticIp
