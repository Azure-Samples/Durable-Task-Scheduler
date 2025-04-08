param ipAllowlist array
param location string
param tags object = {}
param name string
param taskhubname string
param skuName string 
param skuCapacity int

resource dts 'Microsoft.DurableTask/schedulers@2024-10-01-preview' = {
  location: location
  tags: tags
  name: name
  properties: {
    ipAllowlist: ipAllowlist
    sku: {
      name: skuName
      capacity: skuCapacity
    }
  }
}

resource taskhub 'Microsoft.DurableTask/schedulers/taskhubs@2024-10-01-preview' = {
  parent: dts
  name: taskhubname
}

output dts_NAME string = dts.name
output dts_URL string = dts.properties.endpoint
output TASKHUB_NAME string = taskhub.name
output connectionString string = 'Endpoint=${dts.properties.endpoint};TaskHub=${taskhub.name};Authentication=DefaultAzure'
