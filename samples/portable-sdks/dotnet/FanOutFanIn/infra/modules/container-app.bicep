param name string
param location string = resourceGroup().location
param tags object = {}

param containerImage string
param containerPort int
param environmentId string
param containerRegistry string = ''
param containerRegistryUsername string = ''
@secure()
param containerRegistryPassword string = ''
param env array = []
param external bool = true
param userAssignedIdentityId string = ''

resource containerApp 'Microsoft.App/containerApps@2023-05-01' = {
  name: name
  location: location
  tags: tags
  identity: !empty(userAssignedIdentityId) ? {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '${userAssignedIdentityId}': {}
    }
  } : null
  properties: {
    managedEnvironmentId: environmentId
    configuration: {
      ingress: {
        external: external
        targetPort: containerPort
        transport: 'auto'
      }
      registries: !empty(containerRegistry) ? [
        {
          server: containerRegistry
          username: containerRegistryUsername
          passwordSecretRef: 'registry-password'
        }
      ] : []
      secrets: !empty(containerRegistryPassword) ? [
        {
          name: 'registry-password'
          value: containerRegistryPassword
        }
      ] : []
    }
    template: {
      containers: [
        {
          name: name
          image: containerImage
          env: env
        }
      ]
      scale: {
        minReplicas: 0
        maxReplicas: 10
      }
    }
  }
}

output uri string = containerApp.properties.configuration.ingress.fqdn
output name string = containerApp.name
