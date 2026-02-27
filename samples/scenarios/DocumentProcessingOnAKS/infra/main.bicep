targetScope = 'subscription'

// Main Bicep module to provision Azure resources for the Durable Task Scheduler on AKS.
// Adapted from the AutoscalingInACA sample to deploy to Azure Kubernetes Service.
// This deployment does NOT include autoscaling support (no KEDA custom scaler).

@minLength(1)
@maxLength(64)
@description('Name of the environment which is used to generate a short unique hash used in all resources.')
param environmentName string

@minLength(1)
@description('Primary location for all resources')
param location string

@description('Id of the user or app to assign application roles')
param principalId string = ''

// AKS parameters
param aksClusterName string = ''
param kubernetesVersion string = '1.32'
param aksVmSize string = 'standard_d4s_v5'
param aksNodeCount int = 2

// Container registry parameters
param containerRegistryName string = ''

// Durable Task Scheduler parameters
param dtsLocation string = location
param dtsSkuName string = 'Consumption'
param dtsName string = ''
param taskHubName string = ''

// Service names (must match azure.yaml service names)
param clientsServiceName string = 'client'
param workerServiceName string = 'worker'

// Optional resource group name override
param resourceGroupName string = ''

var abbrs = loadJsonContent('./abbreviations.json')

// Tags applied to all resources
var tags = {
  'azd-env-name': environmentName
}

// Generate a unique token for resource naming
#disable-next-line no-unused-vars
var resourceToken = toLower(uniqueString(subscription().id, environmentName, location))

// Organize resources in a resource group
resource rg 'Microsoft.Resources/resourceGroups@2021-04-01' = {
  name: !empty(resourceGroupName) ? resourceGroupName : '${abbrs.resourcesResourceGroups}${environmentName}'
  location: location
  tags: tags
}

// ============================
// Identity
// ============================

// User-assigned managed identity for the workloads to authenticate to DTS
module identity './app/user-assigned-identity.bicep' = {
  scope: rg
  params: {
    name: 'dts-aks-identity'
  }
}

// Grant the managed identity the Durable Task Scheduler Contributor role
// Role ID: 0ad04412-c4d5-4796-b79c-f76d14c8d402
module identityAssignDTS './core/security/role.bicep' = {
  name: 'identityAssignDTS'
  scope: rg
  params: {
    principalId: identity.outputs.principalId
    roleDefinitionId: '0ad04412-c4d5-4796-b79c-f76d14c8d402'
    principalType: 'ServicePrincipal'
  }
}

// Grant the deploying user/principal the DTS role for dashboard access
module identityAssignDTSDash './core/security/role.bicep' = {
  name: 'identityAssignDTSDash'
  scope: rg
  params: {
    principalId: principalId
    roleDefinitionId: '0ad04412-c4d5-4796-b79c-f76d14c8d402'
    principalType: 'User'
  }
}

// ============================
// Networking
// ============================

module vnet './core/networking/vnet.bicep' = {
  scope: rg
  params: {
    name: '${abbrs.networkVirtualNetworks}${resourceToken}'
    location: location
    tags: tags
  }
}

// ============================
// Container Registry
// ============================

module containerRegistry './core/host/container-registry.bicep' = {
  name: 'container-registry'
  scope: rg
  params: {
    name: !empty(containerRegistryName) ? containerRegistryName : '${abbrs.containerRegistryRegistries}${resourceToken}'
    location: location
    tags: tags
    sku: {
      name: 'Standard'
    }
    anonymousPullEnabled: false
  }
}

// ============================
// AKS Cluster
// ============================

module aksCluster './core/host/aks-cluster.bicep' = {
  name: 'aks-cluster'
  scope: rg
  params: {
    name: !empty(aksClusterName) ? aksClusterName : '${abbrs.containerServiceManagedClusters}${resourceToken}'
    location: location
    tags: tags
    kubernetesVersion: kubernetesVersion
    agentVMSize: aksVmSize
    agentCount: aksNodeCount
    subnetId: vnet.outputs.aksSubnetId
    containerRegistryName: containerRegistry.outputs.name
  }
}

// ============================
// Workload Identity Federation
// ============================

// Create federated identity credential so that the Kubernetes service account
// can authenticate as the user-assigned managed identity (for DTS access).
module federatedIdentityClient './app/federated-identity.bicep' = {
  name: 'federated-identity-client'
  scope: rg
  params: {
    identityName: identity.outputs.name
    federatedCredentialName: 'fed-${clientsServiceName}'
    oidcIssuerUrl: aksCluster.outputs.oidcIssuerUrl
    serviceAccountNamespace: 'default'
    serviceAccountName: clientsServiceName
  }
}

module federatedIdentityWorker './app/federated-identity.bicep' = {
  name: 'federated-identity-worker'
  scope: rg
  dependsOn: [federatedIdentityClient] // Serialize to avoid concurrent FIC write errors on the same identity
  params: {
    identityName: identity.outputs.name
    federatedCredentialName: 'fed-${workerServiceName}'
    oidcIssuerUrl: aksCluster.outputs.oidcIssuerUrl
    serviceAccountNamespace: 'default'
    serviceAccountName: workerServiceName
  }
}

// ============================
// Durable Task Scheduler
// ============================

module dts './app/dts.bicep' = {
  scope: rg
  name: 'dtsResource'
  params: {
    name: !empty(dtsName) ? dtsName : '${abbrs.dts}${resourceToken}'
    taskhubname: !empty(taskHubName) ? taskHubName : '${abbrs.taskhub}${resourceToken}'
    location: dtsLocation
    tags: tags
    ipAllowlist: [
      '0.0.0.0/0'
    ]
    skuName: dtsSkuName
  }
}

// ============================
// Outputs
// ============================

// Outputs are automatically saved in the local azd environment .env file.
// To see these outputs, run `azd env get-values`.
output AZURE_LOCATION string = location
output AZURE_TENANT_ID string = tenant().tenantId

// Container registry outputs
output AZURE_CONTAINER_REGISTRY_ENDPOINT string = containerRegistry.outputs.loginServer
output AZURE_CONTAINER_REGISTRY_NAME string = containerRegistry.outputs.name

// AKS outputs (used by azd for deployment)
output AZURE_AKS_CLUSTER_NAME string = aksCluster.outputs.clusterName

// Identity outputs (used in Kubernetes manifests)
output AZURE_USER_ASSIGNED_IDENTITY_NAME string = identity.outputs.name
output AZURE_USER_ASSIGNED_IDENTITY_CLIENT_ID string = identity.outputs.clientId

// DTS outputs (used in Kubernetes manifests via env substitution)
output DTS_ENDPOINT string = dts.outputs.dts_URL
output DTS_TASKHUB_NAME string = dts.outputs.TASKHUB_NAME
