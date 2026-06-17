metadata description = 'Creates a federated identity credential for AKS workload identity.'

@description('The name of the user-assigned managed identity')
param identityName string

@description('The name of the federated credential')
param federatedCredentialName string

@description('The OIDC issuer URL from the AKS cluster')
param oidcIssuerUrl string

@description('The Kubernetes namespace for the service account')
param serviceAccountNamespace string = 'default'

@description('The Kubernetes service account name')
param serviceAccountName string

resource identity 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' existing = {
  name: identityName
}

// Federated identity credential binds a Kubernetes service account to the
// user-assigned managed identity, enabling pods to authenticate as that identity.
resource federatedCredential 'Microsoft.ManagedIdentity/userAssignedIdentities/federatedIdentityCredentials@2024-11-30' = {
  parent: identity
  name: federatedCredentialName
  properties: {
    issuer: oidcIssuerUrl
    subject: 'system:serviceaccount:${serviceAccountNamespace}:${serviceAccountName}'
    audiences: [
      'api://AzureADTokenExchange'
    ]
  }
}
