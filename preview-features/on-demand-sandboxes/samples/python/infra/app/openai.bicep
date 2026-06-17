metadata description = 'Creates an Azure OpenAI account with a chat model deployment for the GenerateCode activity.'

@description('Name of the Azure OpenAI (Cognitive Services) account')
param name string

@description('Azure region for the Azure OpenAI account')
param location string = resourceGroup().location

@description('Tags to apply to the account')
param tags object = {}

@description('Custom subdomain used to build the account endpoint')
param customSubDomainName string = name

@description('Name of the chat model deployment the app calls')
param chatDeploymentName string = 'gpt-5.1'

@description('Chat model name')
param chatModelName string = 'gpt-5.1'

@description('Chat model version')
param chatModelVersion string = '2025-11-13'

@description('Deployment SKU name for the chat model')
param chatDeploymentSkuName string = 'GlobalStandard'

@description('Tokens-per-minute capacity (in thousands) for the chat deployment')
param chatDeploymentCapacity int = 30

@description('Principal id of the workload identity that calls Azure OpenAI')
param workloadPrincipalId string

@description('Principal id of the deploying user (optional)')
param userPrincipalId string = ''

resource account 'Microsoft.CognitiveServices/accounts@2024-10-01' = {
  name: name
  location: location
  tags: tags
  kind: 'OpenAI'
  sku: {
    name: 'S0'
  }
  properties: {
    customSubDomainName: customSubDomainName
    publicNetworkAccess: 'Enabled'
    disableLocalAuth: true
  }
}

resource chatDeployment 'Microsoft.CognitiveServices/accounts/deployments@2024-10-01' = {
  parent: account
  name: chatDeploymentName
  sku: {
    name: chatDeploymentSkuName
    capacity: chatDeploymentCapacity
  }
  properties: {
    model: {
      format: 'OpenAI'
      name: chatModelName
      version: chatModelVersion
    }
  }
}

// Cognitive Services OpenAI User role (5e0bd9bd-7b93-4f28-af87-19fc36ad61bd)
var openAiUserRole = subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '5e0bd9bd-7b93-4f28-af87-19fc36ad61bd')

resource workloadOpenAiAccess 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  scope: account
  name: guid(account.id, workloadPrincipalId, openAiUserRole)
  properties: {
    roleDefinitionId: openAiUserRole
    principalId: workloadPrincipalId
    principalType: 'ServicePrincipal'
  }
}

resource userOpenAiAccess 'Microsoft.Authorization/roleAssignments@2022-04-01' = if (!empty(userPrincipalId)) {
  scope: account
  name: guid(account.id, userPrincipalId, openAiUserRole)
  properties: {
    roleDefinitionId: openAiUserRole
    principalId: userPrincipalId
    principalType: 'User'
  }
}

output endpoint string = account.properties.endpoint
output name string = account.name
output chatDeploymentName string = chatDeployment.name
