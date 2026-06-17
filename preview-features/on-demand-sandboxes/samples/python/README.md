# On-demand Sandboxes demo (Python): LLM-generated code interpreter

The Python port of the [.NET demo](../dotnet/README.md). A three-step Durable Task
workflow that demonstrates the **On-demand Sandboxes** preview of Azure Durable
Task Scheduler (DTS), using the
[`durabletask.azuremanaged.preview.sandboxes`](https://github.com/microsoft/durabletask-python/pull/151)
package.

```
   ┌─────────────────────────┐    ┌─────────────────────────┐    ┌─────────────────────────┐
   │  generate_code          │    │  execute_code           │    │  format_answer          │
   │  (in-process Python)    │ -> │  (on-demand sandbox)    │ -> │  (in-process Python)    │
   │  Azure OpenAI -> Python │    │  python3 + pandas       │    │  Pick top region        │
   └─────────────────────────┘    └─────────────────────────┘    └─────────────────────────┘
```

The orchestrator asks a natural-language question over `data/sales_q1.csv`. The LLM
returns a self-contained pandas script. That script is **untrusted** code, so it runs
in a DTS-managed on-demand sandbox — not in the orchestrator's process. The first and
last activities stay in-process; `execute_code` is fanned out one sandbox execution
per region partition.

## Layout

```
python/
├── activities.py        # Shared activity identities (execute_code is a SandboxActivity)
├── main_app.py          # Declarer app: orchestrator + in-process activities + profile
├── remote_worker.py     # Sandbox worker image entrypoint: runs execute_code via python3
├── Containerfile        # Builds the remote worker image (installs SDK + pandas)
├── requirements.txt     # Declarer-app dependencies
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
- The `durabletask-python` preview source checked out (PR #151)

## Install

From the `python/` directory:

```bash
pip install -r requirements.txt
# Durable Task preview SDK from source (PR microsoft/durabletask-python#151):
pip install -e /path/to/durabletask-python -e /path/to/durabletask-python/durabletask-azuremanaged
```

## Build the sandbox image

From the `python/` directory, pass the durabletask-python checkout as the `sdk`
build context:

```bash
ACR=<your-acr-name>
IMAGE=$ACR.azurecr.io/dts-codegen-sandbox-python:v1

docker build \
  -f Containerfile \
  --build-context sdk=$HOME/durabletask-python \
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
