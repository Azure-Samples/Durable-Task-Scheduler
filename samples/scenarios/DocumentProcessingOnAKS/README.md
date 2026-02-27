# Durable Task Scheduler on AKS

A document-processing sample built with the [.NET Durable Task SDK](https://www.nuget.org/packages/Microsoft.DurableTask.Client.AzureManaged) and [Azure Durable Task Scheduler](https://learn.microsoft.com/en-us/azure/durable-task-scheduler/), deployed to **Azure Kubernetes Service (AKS)** with `azd`.

## How It Works

The app has two components — a **Client** and a **Worker** — that communicate through the Durable Task Scheduler (DTS).

The **Client** submits three sample documents and waits for each to be processed. The **Worker** hosts a `DocumentProcessingOrchestration` that runs a two-stage pipeline:

```
Client ──▶ DocumentProcessingOrchestration
               │
               ├─ 1. ValidateDocument            (activity chaining)
               │
               ├─ 2. ClassifyDocument × 3         (fan-out / fan-in)
               │      ├─ Sentiment
               │      ├─ Topic
               │      └─ Priority
               │
               └─ 3. Assemble result string ──▶ return to Client
```

**Key patterns demonstrated:**

- **Activity chaining** — `ValidateDocument` must pass before classification begins.
- **Fan-out / fan-in** — Three `ClassifyDocument` activities run in parallel (Sentiment, Topic, Priority) and the orchestration awaits all three.
- **Client scheduling** — `ScheduleNewOrchestrationInstanceAsync` + `WaitForInstanceCompletionAsync` for fire-and-wait semantics.
- **Workload Identity** — In AKS, pods authenticate to DTS using federated credentials on a user-assigned managed identity (no secrets stored).

Both Client and Worker auto-detect the environment: locally they connect to `http://localhost:8080` with no auth; in AKS they use the DTS endpoint with managed-identity auth via the `ENDPOINT`, `TASKHUB`, and `AZURE_CLIENT_ID` environment variables injected by the Kubernetes manifests.

## Project Structure

```
├── azure.yaml                          # azd service definitions (host: aks)
├── scripts/acr-build.sh                # Predeploy hook — builds images via ACR Tasks
├── Client/
│   ├── Program.cs                      # Submits 3 documents, waits for results
│   ├── Models/DocumentInfo.cs          # Input record
│   ├── Dockerfile
│   └── manifests/deployment.tmpl.yaml  # K8s deployment + service account
├── Worker/
│   ├── Program.cs                      # Registers orchestrations & activities
│   ├── Orchestrations/
│   │   └── DocumentProcessingOrchestration.cs
│   ├── Activities/
│   │   ├── ValidateDocument.cs         # Checks title + content (returns bool)
│   │   └── ClassifyDocument.cs         # Stub classifier (called 3× in parallel)
│   ├── Models/DocumentModels.cs        # DocumentInfo, ClassifyRequest, ClassificationResult
│   ├── Dockerfile
│   └── manifests/deployment.tmpl.yaml  # K8s deployment (2 replicas) + service account
└── infra/                              # Bicep modules: AKS, ACR, VNet, DTS, identity, RBAC
```

## Prerequisites

- [.NET 10 SDK](https://dotnet.microsoft.com/download/dotnet/10.0)
- [Docker Desktop](https://www.docker.com/products/docker-desktop/) — for the local DTS emulator
- [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli) & [Azure Developer CLI (`azd`)](https://learn.microsoft.com/en-us/azure/developer/azure-developer-cli/install-azd) — for Azure deployment
- [kubectl](https://kubernetes.io/docs/tasks/tools/) — for verifying the AKS deployment

## Run Locally

### 1. Start the DTS emulator

```bash
docker run -d --name dts-emulator -p 8080:8080 -p 8082:8082 \
  mcr.microsoft.com/dts/dts-emulator:latest
```

- Port **8080** — gRPC endpoint (used by Client and Worker)
- Port **8082** — [Dashboard UI](http://localhost:8082) for inspecting orchestrations

### 2. Build the solution

```bash
dotnet build DurableTaskOnAKS.sln
```

### 3. Start the Worker

```bash
cd Worker && dotnet run
```

Wait until you see `Sidecar work-item streaming connection established.`

### 4. Start the Client (in a separate terminal)

```bash
cd Client && dotnet run
```

### 5. Expected output

```
Endpoint: http://localhost:8080 | TaskHub: default
Submitting 3 documents...

  Scheduled [abc123] 'Cloud Migration Strategy'
  -> Processed 'Cloud Migration Strategy': Sentiment=Positive, Topic=Technology, Priority=Normal

  Scheduled [def456] 'Quarterly Incident Report'
  -> Processed 'Quarterly Incident Report': Sentiment=Positive, Topic=Technology, Priority=Normal

  Scheduled [ghi789] 'ML Model Evaluation'
  -> Processed 'ML Model Evaluation': Sentiment=Positive, Topic=Technology, Priority=Normal

Done.
```

### 6. Clean up

```bash
docker stop dts-emulator && docker rm dts-emulator
```

## Deploy to Azure

### 1. Provision and deploy

```bash
azd auth login && az login
azd up
```

This provisions all required Azure resources via Bicep (~5–10 min):

| Resource | Purpose |
|----------|---------|
| **AKS cluster** | Hosts Client (1 replica) and Worker (2 replicas) pods |
| **Azure Container Registry** | Stores Docker images (built server-side via ACR Tasks) |
| **Durable Task Scheduler** (Consumption SKU) | Managed orchestration backend |
| **VNet** | Network isolation for AKS |
| **User-assigned managed identity** + federated credentials | Workload Identity auth from pods to DTS |

### 2. Verify the deployment

```bash
# Get AKS credentials
az aks get-credentials --resource-group <rg-name> --name <aks-name>

# Check pods are running (1 client + 2 workers)
kubectl get pods

# View client output (orchestration results)
kubectl logs -l app=client --tail=30

# View worker logs (activity execution)
kubectl logs -l app=worker --tail=30
```

You can find the resource group and cluster names from `azd env get-values`.

Expected client log output:

```
Submitting 3 documents...

  Scheduled [...] 'Cloud Migration Strategy'
  -> Processed 'Cloud Migration Strategy': Sentiment=Positive, Topic=Technology, Priority=Normal

  Scheduled [...] 'Quarterly Incident Report'
  -> Processed 'Quarterly Incident Report': Sentiment=Positive, Topic=Technology, Priority=Normal

  Scheduled [...] 'ML Model Evaluation'
  -> Processed 'ML Model Evaluation': Sentiment=Positive, Topic=Technology, Priority=Normal

Done.
```

You can also view orchestration history in the **Azure Portal → Durable Task Scheduler → Task Hub** dashboard.

## Cleanup

```bash
azd down
```

## Resources

- [Durable Task Scheduler documentation](https://learn.microsoft.com/en-us/azure/durable-task-scheduler/)
- [.NET Durable Task SDK (NuGet)](https://www.nuget.org/packages/Microsoft.DurableTask.Client.AzureManaged)
- [AKS Workload Identity](https://learn.microsoft.com/en-us/azure/aks/workload-identity-overview)
