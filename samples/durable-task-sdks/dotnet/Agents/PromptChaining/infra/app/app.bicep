param appName string
param location string = resourceGroup().location
param tags object = {}

param identityName string
param containerAppsEnvironmentName string
param containerRegistryName string
param serviceName string = 'aca'
param dtsEndpoint string
param taskHubName string
// Legacy parameters (kept for backward compatibility)
param agentConnectionString string = ''
param openAiEndpoint string = ''
param openAiDeploymentName string = 'gpt-4o-mini'
// New parameters using direct naming convention
param AGENT_CONNECTION_STRING string = ''
param OPENAI_DEPLOYMENT_NAME string = 'gpt-4o-mini'
param AGENT_CONNECTION_STRING__clientId string = ''
param DALLE_ENDPOINT string = ''
param clientBaseUrl string = ''

type managedIdentity = {
  resourceId: string
  clientId: string
}

@description('Unique identifier for user-assigned managed identity.')
param userAssignedManagedIdentity managedIdentity

// Define different container configurations based on service type
var serviceConfig = serviceName == 'client' ? {
  enableIngress: true
  external: true
  targetPort: 5000
  containerImage: 'mcr.microsoft.com/dotnet/aspnet:8.0'
  probes: [
    {
      type: 'Startup'
      httpGet: {
        path: '/health'
        port: 5000
        scheme: 'HTTP'
      }
      initialDelaySeconds: 5
      periodSeconds: 10
      failureThreshold: 3
      timeoutSeconds: 5
    }
    {
      type: 'Liveness'
      httpGet: {
        path: '/health'
        port: 5000
        scheme: 'HTTP'
      }
      initialDelaySeconds: 10
      periodSeconds: 30
      failureThreshold: 3
      timeoutSeconds: 5
    }
  ]
} : {
  enableIngress: false // Worker doesn't need ingress
  external: false
  targetPort: 0 // Not needed for worker
  containerImage: 'mcr.microsoft.com/dotnet/runtime:8.0'
  probes: [] // No probes for worker
}

module containerAppsApp '../core/host/container-app.bicep' = {
  name: 'container-apps-${serviceName}'
  params: {
    name: appName
    containerAppsEnvironmentName: containerAppsEnvironmentName
    containerRegistryName: containerRegistryName
    location: location
    tags: union(tags, { 'azd-service-name': serviceName })
    enableIngress: serviceConfig.enableIngress
    external: serviceConfig.external
    targetPort: serviceConfig.targetPort
    containerImage: serviceConfig.containerImage
    identityName: identityName
    minReplicas: 1
    maxReplicas: 10
    environmentVariables: serviceName == 'worker' ? [
      {
        name: 'AZURE_MANAGED_IDENTITY_CLIENT_ID'
        secretRef: 'azure-managed-identity-client-id'
      }
      {
        name: 'AZURE_CLIENT_ID'
        value: userAssignedManagedIdentity.clientId
      }
      {
        name: 'ENDPOINT'
        value: 'Endpoint=${dtsEndpoint};Authentication=ManagedIdentity;ClientID=${userAssignedManagedIdentity.clientId}'
      }
      {
        name: 'TASKHUB'
        value: taskHubName
      }
      {
        name: 'AGENT_CONNECTION_STRING'
        value: !empty(AGENT_CONNECTION_STRING) ? AGENT_CONNECTION_STRING : !empty(agentConnectionString) ? agentConnectionString : ''
      }
      {
        name: 'OPENAI_DEPLOYMENT_NAME'
        value: !empty(OPENAI_DEPLOYMENT_NAME) ? OPENAI_DEPLOYMENT_NAME : !empty(openAiDeploymentName) ? openAiDeploymentName : 'gpt-4o-mini'
      }
      {
        name: 'AGENT_CONNECTION_STRING__clientId'
        value: !empty(AGENT_CONNECTION_STRING__clientId) ? AGENT_CONNECTION_STRING__clientId : userAssignedManagedIdentity.clientId
      }
      {
        name: 'AZURE_OPENAI_ENDPOINT'
        value: !empty(openAiEndpoint) ? openAiEndpoint : ''
      }
      {
        name: 'DALLE_ENDPOINT'
        value: DALLE_ENDPOINT
      }
      {
        name: 'CLIENT_BASE_URL'
        value: clientBaseUrl
      }
      {
        name: 'CLIENT_BASE_URL'
        value: clientBaseUrl
      }
    ] : [
      {
        name: 'AZURE_MANAGED_IDENTITY_CLIENT_ID'
        secretRef: 'azure-managed-identity-client-id'
      }
      {
        name: 'AZURE_CLIENT_ID'
        value: userAssignedManagedIdentity.clientId
      }
      {
        name: 'ENDPOINT'
        value: 'Endpoint=${dtsEndpoint};Authentication=ManagedIdentity;ClientID=${userAssignedManagedIdentity.clientId}'
      }
      {
        name: 'TASKHUB'
        value: taskHubName
      }
      {
        name: 'AGENT_CONNECTION_STRING'
        value: !empty(AGENT_CONNECTION_STRING) ? AGENT_CONNECTION_STRING : !empty(agentConnectionString) ? agentConnectionString : ''
      }
      {
        name: 'OPENAI_DEPLOYMENT_NAME'
        value: !empty(OPENAI_DEPLOYMENT_NAME) ? OPENAI_DEPLOYMENT_NAME : !empty(openAiDeploymentName) ? openAiDeploymentName : 'gpt-4o-mini'
      }
      {
        name: 'AGENT_CONNECTION_STRING__clientId'
        value: !empty(AGENT_CONNECTION_STRING__clientId) ? AGENT_CONNECTION_STRING__clientId : userAssignedManagedIdentity.clientId
      }
    ]
    secrets: [
      {
        name: 'azure-managed-identity-client-id'
        value: userAssignedManagedIdentity.clientId
      }
    ]
    enableCustomScaleRule: false // Disable custom scale rule for initial deployment
    scaleRuleName: 'dtsscaler-orchestration'
    scaleRuleType: 'azure-durabletask-scheduler'
    scaleRuleMetadata: {
      endpoint: dtsEndpoint
      maxConcurrentWorkItemsCount: '1'
      taskhubName: taskHubName
      workItemType: 'Orchestration'
    }
    scaleRuleIdentity: userAssignedManagedIdentity.resourceId
    probes: serviceConfig.probes
  }
}

output endpoint string = containerAppsApp.outputs.containerAppFqdn
output envName string = appName
