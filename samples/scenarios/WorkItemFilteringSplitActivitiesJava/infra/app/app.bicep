param appName string
param location string = resourceGroup().location
param tags object = {}

param identityName string
param containerAppsEnvironmentName string
param containerRegistryName string
param serviceName string = 'aca'
param dtsEndpoint string
param taskHubName string

@description('DTS work item type for KEDA scaling: Orchestration or Activity')
param workItemType string = 'Orchestration'

type managedIdentity = {
  resourceId: string
  clientId: string
}

@description('Unique identifier for user-assigned managed identity.')
param userAssignedManagedIdentity managedIdentity

module containerAppsApp '../core/host/container-app.bicep' = {
  name: 'container-apps-${serviceName}'
  params: {
    name: appName
    containerAppsEnvironmentName: containerAppsEnvironmentName
    containerRegistryName: containerRegistryName
    location: location
    tags: union(tags, { 'azd-service-name': serviceName })
    ingressEnabled: false
    secrets: {
        'azure-managed-identity-client-id':  userAssignedManagedIdentity.clientId
      }
    env: [
      {
        name: 'AZURE_MANAGED_IDENTITY_CLIENT_ID'
        secretRef: 'azure-managed-identity-client-id'
      }
      {
        name: 'ENDPOINT'
        value: dtsEndpoint
      }
      {
        name: 'TASKHUB'
        value: taskHubName
      }
    ]
    identityName: identityName
    containerMinReplicas: 0
    containerMaxReplicas: 10
    enableCustomScaleRule: true
    scaleRuleName: 'dtsscaler-${serviceName}'
    scaleRuleType: 'azure-durabletask-scheduler'
    scaleRuleMetadata: {
      endpoint: dtsEndpoint
      maxConcurrentWorkItemsCount: '1'
      taskhubName: taskHubName
      workItemType: workItemType
    }
    scaleRuleIdentity: userAssignedManagedIdentity.resourceId
  }
}

output endpoint string = containerAppsApp.outputs.uri
output envName string = containerAppsApp.outputs.name
