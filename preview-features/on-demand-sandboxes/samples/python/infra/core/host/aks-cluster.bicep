metadata description = 'Creates an Azure Kubernetes Service (AKS) cluster.'

@description('The name of the AKS cluster')
param name string

@description('The Azure region for the AKS cluster')
param location string = resourceGroup().location

@description('Tags to apply to the AKS cluster')
param tags object = {}

@description('The Kubernetes version for the AKS cluster')
param kubernetesVersion string = '1.32'

@description('The VM size for the default node pool')
param agentVMSize string = 'standard_d4s_v5'

@description('The number of nodes in the default node pool')
param agentCount int = 2

@description('The minimum number of nodes for autoscaling')
param agentMinCount int = 1

@description('The maximum number of nodes for autoscaling')
param agentMaxCount int = 5

@description('The subnet resource ID for the AKS nodes')
param subnetId string = ''

@description('The name of the container registry to attach')
param containerRegistryName string = ''

@description('Enable OIDC issuer for workload identity')
param enableOidcIssuer bool = true

@description('Enable workload identity')
param enableWorkloadIdentity bool = true

// AKS cluster with workload identity and OIDC issuer enabled
resource aksCluster 'Microsoft.ContainerService/managedClusters@2024-09-01' = {
  name: name
  location: location
  tags: tags
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    kubernetesVersion: kubernetesVersion
    dnsPrefix: name
    enableRBAC: true
    agentPoolProfiles: [
      {
        name: 'system'
        count: agentCount
        vmSize: agentVMSize
        mode: 'System'
        osType: 'Linux'
        osSKU: 'AzureLinux'
        enableAutoScaling: true
        minCount: agentMinCount
        maxCount: agentMaxCount
        vnetSubnetID: !empty(subnetId) ? subnetId : null
      }
    ]
    networkProfile: {
      networkPlugin: 'azure'
      networkPolicy: 'azure'
      serviceCidr: '10.1.0.0/16'
      dnsServiceIP: '10.1.0.10'
    }
    oidcIssuerProfile: {
      enabled: enableOidcIssuer
    }
    securityProfile: {
      workloadIdentity: {
        enabled: enableWorkloadIdentity
      }
    }
  }
}

// Grant AKS kubelet identity AcrPull access to the container registry
module registryAccess '../security/registry-access.bicep' = if (!empty(containerRegistryName)) {
  name: 'aks-registry-access'
  params: {
    containerRegistryName: containerRegistryName
    principalId: aksCluster.properties.identityProfile.kubeletidentity.objectId
  }
}

output clusterName string = aksCluster.name
output clusterId string = aksCluster.id
output oidcIssuerUrl string = aksCluster.properties.oidcIssuerProfile.issuerURL
output kubeletIdentityObjectId string = aksCluster.properties.identityProfile.kubeletidentity.objectId
