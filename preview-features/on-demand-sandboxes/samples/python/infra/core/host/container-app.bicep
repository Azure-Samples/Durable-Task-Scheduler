metadata description = 'Creates a container app in an Azure Container Apps environment.'
param name string
param location string = resourceGroup().location
param tags object = {}

@description('Name of the environment for container apps')
param containerAppsEnvironmentName string

@description('CPU cores allocated to a single container instance, e.g., 0.5')
param containerCpuCoreCount string = '0.5'

@description('The maximum number of replicas to run. Must be at least 1.')
@minValue(1)
param containerMaxReplicas int = 1

@description('Memory allocated to a single container instance, e.g., 1Gi')
param containerMemory string = '1.0Gi'

@description('The minimum number of replicas to run. Must be at least 0.')
param containerMinReplicas int = 1

@description('The name of the container')
param containerName string = 'main'

@description('The name of the container registry')
param containerRegistryName string = ''

@description('Hostname suffix for container registry. Set when deploying to sovereign clouds')
param containerRegistryHostSuffix string = 'azurecr.io'

@description('The environment variables for the container')
param env array = []

@description('Specifies if the resource ingress is exposed externally')
param external bool = true

@description('The name of the user-assigned identity')
param identityName string = ''

@description('The name of the container image')
param imageName string = ''

@description('Specifies if Ingress is enabled for the container app')
param ingressEnabled bool = true

param revisionMode string = 'Single'

@description('The target port for the container')
param targetPort int = 80

resource userIdentity 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' existing = if (!empty(identityName)) {
  name: identityName
}

// Private registry support requires both an ACR name and a user-assigned managed identity.
var usePrivateRegistry = !empty(identityName) && !empty(containerRegistryName)

// Grant the identity AcrPull before the app is created, otherwise the container app
// throws a provisioning error when it tries to pull the image.
module containerRegistryAccess '../security/registry-access.bicep' = if (usePrivateRegistry) {
  name: '${deployment().name}-registry-access'
  params: {
    containerRegistryName: containerRegistryName
    principalId: usePrivateRegistry ? userIdentity.properties.principalId : ''
  }
}

resource app 'Microsoft.App/containerApps@2025-01-01' = {
  name: name
  location: location
  tags: tags
  dependsOn: usePrivateRegistry ? [ containerRegistryAccess ] : []
  identity: {
    type: !empty(identityName) ? 'UserAssigned' : 'None'
    userAssignedIdentities: !empty(identityName) ? { '${userIdentity.id}': {} } : null
  }
  properties: {
    managedEnvironmentId: containerAppsEnvironment.id
    configuration: {
      activeRevisionsMode: revisionMode
      ingress: ingressEnabled ? {
        external: external
        targetPort: targetPort
        transport: 'auto'
      } : null
      registries: usePrivateRegistry ? [
        {
          server: '${containerRegistryName}.${containerRegistryHostSuffix}'
          identity: userIdentity.id
        }
      ] : []
    }
    template: {
      containers: [
        {
          image: !empty(imageName) ? imageName : 'mcr.microsoft.com/azuredocs/containerapps-helloworld:latest'
          name: containerName
          env: env
          resources: {
            cpu: json(containerCpuCoreCount)
            memory: containerMemory
          }
        }
      ]
      scale: {
        minReplicas: containerMinReplicas
        maxReplicas: containerMaxReplicas
      }
    }
  }
}

resource containerAppsEnvironment 'Microsoft.App/managedEnvironments@2023-05-01' existing = {
  name: containerAppsEnvironmentName
}

output identityPrincipalId string = !empty(identityName) ? userIdentity.properties.principalId : ''
output imageName string = imageName
output name string = app.name
output uri string = ingressEnabled ? 'https://${app.properties.configuration.ingress.fqdn}' : ''
