param name string
param location string
param tags object = {}
param sku object = {
  name: 'EP1'
  tier: 'ElasticPremium'
}

module appServicePlan 'br/public:avm/res/web/serverfarm:0.4.1' = {
  name: 'serverfarmDeployment'
  params: {
    name: name
    location: location
    tags: tags
    skuName: sku.name
    kind: 'linux'
    zoneRedundant: false
  }
}

output id string = appServicePlan.outputs.resourceId
output name string = appServicePlan.outputs.name
