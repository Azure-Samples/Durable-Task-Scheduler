# On-demand Sandboxes demo (Python): LLM-generated code interpreter

The Python port of the [.NET demo](../dotnet/README.md). A three-step Durable Task
workflow that demonstrates the **On-demand Sandboxes** preview of Azure Durable
Task Scheduler (DTS), using the `durabletask.azuremanaged.preview.sandboxes`
package.

The orchestrator asks a natural-language question over `data/sales_q1.csv`. The LLM
returns a self-contained pandas script. That script is **untrusted** code, so it runs
in a DTS-managed on-demand sandbox, not in the orchestrator's process. The first and
last activities stay in-process; `execute_code` is fanned out one sandbox execution
per region partition.

## Layout

```
python/
├── activities.py        # Shared activity identities (execute_code is a SandboxActivity)
├── main_app.py          # Declarer app: orchestrator + in-process activities + profile
├── remote_worker.py     # Sandbox worker image entrypoint: runs execute_code via python3
├── Containerfile        # Builds the remote worker (sandbox) image
├── Containerfile.mainapp # Builds the main_app image deployed to AKS
├── requirements.txt     # Declarer-app dependencies
├── azure.yaml           # azd service + hooks (Deploy to Azure)
├── infra/               # Bicep: AKS, ACR, identity, Azure OpenAI, scheduler wiring
├── scripts/             # acr-build.sh + attach-scheduler-identity.sh (azd hooks)
├── manifests/           # K8s deployment template for main_app
└── data/sales_q1.csv    # Sample dataset (~300 rows)
```

- `execute_code` is declared as an on-demand sandbox activity by the `code-executor`
  worker profile (the `@sandbox_worker_profile` class in `main_app.py`). It is never
  registered on the main app worker.
- `generate_code` and `format_answer` run in-process in the main app worker.

## Prerequisites

- Python 3.12+
- Docker (to build the sandbox image)
- A DTS scheduler + task hub with the On-demand Sandboxes preview enabled
- An Azure Container Registry the sandbox platform can pull from
- Two user-assigned managed identities (image pull + scheduler connect)
- An Azure OpenAI deployment of a chat model (GPT-4o, GPT-4.1, etc.)

## Install

From the `python/` directory:

```bash
pip install -r requirements.txt
pip install durabletask==1.6.0 durabletask-azuremanaged==1.6.0
```

## Build the sandbox image

From the `python/` directory:

```bash
ACR=<your-acr-name>
IMAGE=$ACR.azurecr.io/dts-codegen-sandbox-python:v1

docker build \
  -f Containerfile \
  -t $IMAGE \
  .

# Enable anonymous pull so DTS can fetch the sandbox image without credentials
az acr update --name $ACR --anonymous-pull-enabled true
az acr login --name $ACR
docker push $IMAGE
```

## Run the orchestrator

```bash
export DTS_ENDPOINT="https://<scheduler-endpoint>"
export DTS_TASK_HUB="<task-hub>"
export DTS_WORKER_PROFILE_ID="code-executor"
export DTS_SANDBOX_CONTAINER_IMAGE="<acr>.azurecr.io/dts-codegen-sandbox-python:v1"
export DTS_SANDBOX_IMAGE_PULL_UMI_CLIENT_ID="<image-pull UMI client ID>"
export DTS_SANDBOX_SCHEDULER_UMI_CLIENT_ID="<scheduler UMI client ID>"

export AOAI_ENDPOINT="https://<your-aoai>.openai.azure.com"
export AOAI_DEPLOYMENT="<your-chat-deployment>"

# Sign in so DefaultAzureCredential can reach DTS and Azure OpenAI
az login

python main_app.py "Which region had the highest total revenue in March 2025?"
```

The declarer prints a dataset preview, the AOAI-generated Python (prefixed
`[generate]`), the orchestration id, and the final answer. The sandbox container
logs (prefixed `[sandbox]`) stream through the DTS dashboard's **On-demand
Sandboxes** tab while `execute_code` runs.

## Deploy to Azure (AKS) with `azd`

The `infra/` folder and `azure.yaml` deploy the **main_app** orchestrator to **Azure
Kubernetes Service** with [`azd`](https://learn.microsoft.com/azure/developer/azure-developer-cli/install-azd).
The sandbox worker image (`remote_worker.py`) is built and pushed to ACR; DTS starts it
on demand, so it is never deployed to the cluster.

> The Durable Task Scheduler is **not created** by this template. You pass in an
> existing one. On-demand Sandboxes is a private-preview feature that must be enabled on
> the scheduler out of band, so the scheduler is patched separately and supplied here by
> name.

### What gets provisioned

| Resource | Purpose |
|----------|---------|
| **AKS cluster** | Hosts the `main_app` orchestrator pod (workload identity enabled) |
| **Azure Container Registry** | Stores the main-app and sandbox-worker images (built server-side via ACR Tasks) |
| **User-assigned managed identity** + federated credential | Pod auth to DTS/Azure OpenAI, ACR pull for the sandbox, and the sandbox's connection back to DTS |
| **Azure OpenAI** + `gpt-4o` deployment | Backs the in-process `generate_code` activity |
| **VNet** | Network isolation for AKS |

The deployment also **ensures the task hub** exists, grants the identity the roles it
needs (AcrPull, Durable Task data access, Cognitive Services OpenAI User), and a
`postprovision` hook **attaches the identity to your scheduler** (a merge-safe PATCH).

### Prerequisites

- An existing **DTS scheduler** with the On-demand Sandboxes preview enabled, and its
  resource group name.
- [Azure Developer CLI (`azd`)](https://learn.microsoft.com/azure/developer/azure-developer-cli/install-azd), [Azure CLI](https://learn.microsoft.com/cli/azure/install-azure-cli), and [kubectl](https://kubernetes.io/docs/tasks/tools/).
- Azure OpenAI quota for `gpt-4o` in your target region.

### Deploy

```bash
azd auth login && az login

# Point the template at your existing (preview-enabled) scheduler.
azd env set DTS_SCHEDULER_NAME "<scheduler-name>"
azd env set DTS_SCHEDULER_RESOURCE_GROUP "<scheduler-resource-group>"
# Optional overrides: DTS_TASK_HUB (default: default), AZURE_OPENAI_LOCATION

azd up
```

`azd` provisions the resources, builds both images via ACR Tasks, attaches the identity
to your scheduler, and deploys the `main_app` pod. If you don't set `DTS_SCHEDULER_NAME`
/ `DTS_SCHEDULER_RESOURCE_GROUP` first, `azd` prompts for them.

### Verify

```bash
az aks get-credentials --resource-group <rg-name> --name <aks-name>   # from `azd env get-values`
kubectl get pods
kubectl logs -l app=mainapp --tail=50
```

The `main_app` pod runs the orchestration; `[sandbox]` logs from `execute_code` stream in
the DTS dashboard's **On-demand Sandboxes** tab.

### Clean up

```bash
azd down
```

This removes the resources the template created. Your scheduler is left untouched (it was
not created here); detach the identity manually if you no longer need it.

## Sample questions to try

- `Which region had the highest total revenue in March 2025?`
- `What was the best-selling product in Q1?`
- `Average revenue per transaction in February?`

## What's in-process vs on-demand sandbox

| Activity        | Runs where    | Why                                                   |
| --------------- | ------------- | ----------------------------------------------------- |
| generate_code   | In-process    | Plain Azure OpenAI HTTP call. No reason to split out. |
| execute_code    | **Sandbox**   | Untrusted LLM-generated code + different runtime.     |
| format_answer   | In-process    | Trivial result aggregation.                           |

Only `execute_code` is declared on the `code-executor` sandbox worker profile via
`options.add_activity(...)`. Everything else runs wherever the orchestrator runs.
