# Task queues (preview)

Task queues group orchestration and activity work inside one task hub and
limit the sustained rate at which the scheduler hands that work to workers.
The samples start order workflows with the tag `DTS_TaskQueue=orders`.
Where supported, partner activities and child orchestrations override the
tag with `partner-api`. The example configuration sets `orders` to 20
handouts per second and `partner-api` to 5. These limits apply only when the
queues are provisioned.

Activities simulate order processing; they do not call a real partner service.
See the [task queue documentation](https://learn.microsoft.com/azure/durable-task/scheduler/durable-task-scheduler-task-queues)
for the feature's management and routing behavior.

## Supported languages and bindings

| SDK or binding | Start tags | Activity override | Child override |
|----------------|------------|-------------------|----------------|
| Durable Task .NET | Yes | Yes | Yes |
| Durable Task JavaScript | Yes | Yes | Yes |
| Durable Task TypeScript | Yes | Yes | Yes |
| Durable Task Python | Yes | Yes | No |
| Durable Task Java (1.6.0+) | Yes | No | No |
| Durable Task Go (beta) | Yes | Yes | Yes |
| Durable Functions .NET isolated | Yes | Yes | Yes |
| Durable Functions Java | Yes | No | No |

Durable Functions .NET in-process, JavaScript, TypeScript, Python, and
PowerShell do not expose the explicit tag options needed by these samples.
Standalone SDK support does not imply support in that language's Functions
binding. Java and Python children still inherit their parent's queue.
Entity work items do not use task queues. Each language's sample is
listed in the [sample catalog](../samples/README.md).

## Prerequisites

- An existing Azure scheduler and a task hub dedicated to the sample you run.
- Permission to manage task queue resources, such as Contributor on the
  resource group or scheduler. Data-plane permissions do not grant this.
- Durable Task Data Contributor access for the worker/client identity.
- Azure CLI with an existing signed-in session and the correct subscription
  selected. The REST examples use Bash and need no task queue CLI extension.

Use the local emulator for workflow development. Several samples default to
it; the Java samples require an explicit emulator connection. Use Azure to
create the preview queue resources and evaluate rate enforcement. Running
successfully with a queue tag does not prove that its named queue exists or
that throttling occurred.

## Create the queues

From the repository root, set the existing resource names:

```bash
subscriptionId=$(az account show --query id --output tsv)
resourceGroup="RESOURCE_GROUP"
schedulerName="SCHEDULER_NAME"
taskHubName="TASK_HUB_NAME"
apiVersion="2026-07-01-preview"
taskHubResourceId="/subscriptions/$subscriptionId/resourceGroups/$resourceGroup/providers/Microsoft.DurableTask/schedulers/$schedulerName/taskHubs/$taskHubName"
queueBaseUrl="https://management.azure.com$taskHubResourceId/taskQueues"
```

Create `orders` and `partner-api` with ARM REST:

```bash
az rest --method put \
  --url "$queueBaseUrl/orders?api-version=$apiVersion" \
  --body '{"properties":{"throughput":{"actionsPerSecond":20}}}'
az rest --method put \
  --url "$queueBaseUrl/partner-api?api-version=$apiVersion" \
  --body '{"properties":{"throughput":{"actionsPerSecond":5}}}'
```

Alternatively, deploy the shared Bicep template. It references the existing
scheduler and hub and creates only the two queues:

```bash
az deployment group create \
  --resource-group "$resourceGroup" \
  --template-file samples/infra/task-queues/task-queues.bicep \
  --parameters schedulerName="$schedulerName" taskHubName="$taskHubName"
```

Queues require management API `2026-07-01-preview` or later. Some Bicep
versions emit `BCP081` because preview resource types are not available in
their type definitions. Check the
[resource schema](https://learn.microsoft.com/azure/templates/microsoft.durabletask/2026-07-01-preview/schedulers/taskhubs/taskqueues).
Queue names must contain 3 to 64 letters, digits, or hyphens. A queue has no
independent `location` or Azure resource `tags`.

## Wait for provisioning

Run this bounded poll after creating or updating queues. Do not schedule
tagged work until each required queue reports `Succeeded`. The Java samples
need only `orders`, but the shared template creates both queues.

```bash
waitForQueue() {
  local queueName="$1"
  local state
  for ((attempt = 0; attempt < 60; attempt++)); do
    state=$(az rest --method get \
      --url "$queueBaseUrl/$queueName?api-version=$apiVersion" \
      --query properties.provisioningState --output tsv) || return 1
    case "$state" in
      Succeeded) return 0 ;;
      Failed|Canceled)
        printf 'Queue %s provisioning %s\n' "$queueName" "$state" >&2
        return 1 ;;
    esac
    sleep 5
  done
  printf 'Timed out provisioning queue %s\n' "$queueName" >&2
  return 1
}
waitForQueue orders && waitForQueue partner-api
```

If polling fails, stop and resolve the resource error before running a sample.
An accepted PUT or PATCH is not confirmation that provisioning finished.
Each REST request also has its own network latency; the loop allows at most
60 requests and five-second polling intervals per queue.

## Connect eligible workers

Only workers without work-item filters can receive work from defined queues.
Even wildcard filters make a worker ineligible. The .NET SDK sample calls
`UseWorkItemFilters(null)` explicitly, and the Functions apps set
`storageProvider.workItemFilteringEnabled` to `false`. Java, Python, and Go
workers do not opt in to filtering. The JavaScript/TypeScript workers connect
without filters.

Use a separate task hub for each sample, or stop every other worker before
running it. Unfiltered workers must register all tasks routed to that hub;
different language samples are not interchangeable workers.

The literal `DTS_TaskQueue` tag selects an existing queue in the same hub.
It does not create a resource. Untagged activities and children inherit the
parent queue. An override changes the called task's queue, leaving the parent
in its original queue. A defined queue retains work until an eligible worker
connects.

## Inspect and change rates

```bash
az rest --method get --url "$queueBaseUrl?api-version=$apiVersion"
az rest --method get --url "$queueBaseUrl/orders?api-version=$apiVersion"

az rest --method patch \
  --url "$queueBaseUrl/orders?api-version=$apiVersion" \
  --body '{"properties":{"throughput":{"actionsPerSecond":10}}}'
waitForQueue orders
az rest --method get --url "$queueBaseUrl/orders?api-version=$apiVersion"
```

To remove the queue-specific limit while keeping the queue:

```bash
az rest --method patch \
  --url "$queueBaseUrl/orders?api-version=$apiVersion" \
  --body '{"properties":{"throughput":null}}'
waitForQueue orders
```

Omitting throughput from PATCH preserves the existing rate. Omitting it from
PUT removes the queue-specific limit. Use a positive integer for a configured
rate. Inspect the returned configuration after provisioning succeeds.

## Interpret results

Each orchestration turn, activity delivery, and redelivery counts as one
handout. Orchestrations and activities in a queue share its sustained rate;
short bursts can exceed it. A rate of 20 does not mean 20 completed orders,
activities, or partner API requests per second. Worker or scheduler capacity
can reduce actual throughput.

Queues do not reserve compute, provide physical isolation, or assign
priority. Apply a separate limiter at the downstream call site when an API
requires a strict request limit.

Inspect orchestration status and tags in the scheduler dashboard. The
sample's output checks verify workflow completion, not the queue's rate.
For a rate experiment, provision both queues, keep other workers out of the
hub, submit enough work to sustain a backlog, and measure scheduler handouts
over a sustained interval. Do not infer enforcement from elapsed time for
one short workflow.

## Cleanup and troubleshooting

Deleting or misspelling a queue causes tagged work to use ordinary task hub
routing without that queue's limit. Deletion does not pause or terminate
existing instances. After all sample work finishes, remove only the queues
you created:

```bash
az rest --method delete --url "$queueBaseUrl/orders?api-version=$apiVersion"
az rest --method delete --url "$queueBaseUrl/partner-api?api-version=$apiVersion"
```

Deletion is asynchronous. Follow ARM operation tracking to completion, or
poll GET with a bounded timeout until it returns HTTP 404. Authorization
errors and other request failures do not confirm deletion. See
[ARM asynchronous operations](https://learn.microsoft.com/azure/azure-resource-manager/management/async-operations).

If work stalls, check queue provisioning, the tag's spelling and task hub,
worker health, filtering configuration, and task registrations. If it runs
without the expected limit, check whether the queue exists and distinguish
handouts from completions and brief bursts.
