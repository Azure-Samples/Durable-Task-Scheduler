// container-app.bicep - Creates a Container App resource in a Container Apps Environment

@description('The name of the container app')
param name string

@description('The location for the resources')
param location string = resourceGroup().location

@description('Tags for the resources')
param tags object = {}

@description('The name of the container apps environment')
param containerAppsEnvironmentName string

@description('The ID of the container registry')
param containerRegistryName string = ''

@description('The name of the user-assigned managed identity')
param identityName string = ''

@description('The container image to deploy')
param containerImage string

@description('Target port for the container')
param targetPort int = 80

@description('Environment variables for the container')
param environmentVariables array = []

@description('CPU resources for the container')
param containerCpu string = '0.5'

@description('Memory resources for the container')
param containerMemory string = '1.0Gi'

@description('Minimum number of replicas')
param minReplicas int = 1

@description('Maximum number of replicas')
param maxReplicas int = 10

@description('Enable ingress for the container app')
param enableIngress bool = true

@description('Additional secrets to be set on the container app')
param secrets array = []

@description('Enable custom scale rules')
param enableCustomScaleRule bool = false

@description('Scale rule name')
param scaleRuleName string = ''

@description('Scale rule type')
param scaleRuleType string = ''

@description('Scale rule metadata')
param scaleRuleMetadata object = {}

@description('Scale rule identity')
param scaleRuleIdentity string = ''

@description('Make the container app visible externally')
param external bool = true

@description('Probes to configure for the container app')
param probes array = []

var hasIdentity = !empty(identityName)
var hasRegistry = !empty(containerRegistryName)

resource containerAppsEnvironment 'Microsoft.App/managedEnvironments@2023-11-02-preview' existing = {
  name: containerAppsEnvironmentName
}

resource identity 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' existing = if (hasIdentity) {
  name: identityName
}

resource containerRegistry 'Microsoft.ContainerRegistry/registries@2023-11-01-preview' existing = if (hasRegistry) {
  name: containerRegistryName
}

resource containerApp 'Microsoft.App/containerApps@2023-11-02-preview' = {
  name: name
  location: location
  tags: tags
  identity: hasIdentity ? {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '${identity.id}': {}
    }
  } : null
  properties: {
    managedEnvironmentId: containerAppsEnvironment.id
    configuration: {
      ingress: enableIngress ? {
        external: external
        targetPort: targetPort
        transport: 'auto'
        allowInsecure: false
      } : null
      registries: hasRegistry ? [
        {
          #disable-next-line BCP318
          server: containerRegistry.properties.loginServer
          identity: hasIdentity ? identity.id : null
        }
      ] : []
      secrets: secrets
      activeRevisionsMode: 'Single'
    }
    template: {
      containers: [
        {
          name: name
          image: containerImage
          env: environmentVariables
          resources: {
            cpu: json(containerCpu)
            memory: containerMemory
          }
          probes: !empty(probes) ? probes : []
        }
      ]
      scale: {
        minReplicas: minReplicas
        maxReplicas: maxReplicas
        rules: enableCustomScaleRule ? [
          {
            name: scaleRuleName
            custom: {
              type: scaleRuleType
              metadata: scaleRuleMetadata
              auth: !empty(scaleRuleIdentity) ? [
                {
                  secretRef: 'scale-rule-auth'
                  triggerParameter: 'userAssignedIdentity'
                }
              ] : []
            }
          }
        ] : []
      }
    }
  }
}

output containerAppId string = containerApp.id
output containerAppFqdn string = enableIngress && external ? containerApp.properties.configuration.ingress.fqdn : ''
