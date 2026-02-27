---
name: durable-task-js
description: Build durable, fault-tolerant workflows in JavaScript/TypeScript using the Durable Task SDK with Azure Durable Task Scheduler. Use when creating orchestrations, activities, entities, or implementing patterns like function chaining, fan-out/fan-in, human interaction, or durable timers. Applies to any Node.js application requiring durable execution, state persistence, or distributed coordination without Azure Functions dependency.
---

# Durable Task JavaScript/TypeScript SDK with Durable Task Scheduler

Build fault-tolerant, stateful workflows in JavaScript/TypeScript applications using the Durable Task SDK connected to Azure Durable Task Scheduler.

## Quick Start

### Required npm Packages

```bash
npm install @microsoft/durabletask-js @microsoft/durabletask-js-azuremanaged @azure/identity
```

### Prerequisites

- **Node.js 22 or higher** is required
- An Azure Durable Task Scheduler instance, or the DTS Emulator for local development

### Minimal Worker + Client Setup

```javascript
import {
  ActivityContext,
  OrchestrationContext,
  TOrchestrator,
  OrchestrationStatus,
} from "@microsoft/durabletask-js";
import {
  createAzureManagedClient,
  createAzureManagedWorkerBuilder,
} from "@microsoft/durabletask-js-azuremanaged";

// Activity function
const sayHello = async (_ctx, name) => {
  return `Hello, ${name}!`;
};

// Orchestrator function (generator)
const helloCities = async function* (ctx) {
  const result1 = yield ctx.callActivity(sayHello, "Tokyo");
  const result2 = yield ctx.callActivity(sayHello, "London");
  const result3 = yield ctx.callActivity(sayHello, "Seattle");
  return [result1, result2, result3];
};

// Connection string - defaults to local emulator
const connectionString = process.env.DURABLE_TASK_SCHEDULER_CONNECTION_STRING
  ?? "Endpoint=http://localhost:8080;Authentication=None;TaskHub=default";

// Build worker
const worker = createAzureManagedWorkerBuilder(connectionString)
  .addOrchestrator(helloCities)
  .addActivity(sayHello)
  .build();

// Build client
const client = createAzureManagedClient(connectionString);

// Start worker and schedule orchestration
await worker.start();
const id = await client.scheduleNewOrchestration(helloCities);
const state = await client.waitForOrchestrationCompletion(id, true, 60);

if (state && state.runtimeStatus === OrchestrationStatus.COMPLETED) {
  console.log(`Result: ${state.serializedOutput}`);
}
```

### TypeScript Setup

For TypeScript projects, use `TOrchestrator` type annotation:

```typescript
import { TOrchestrator, OrchestrationContext, ActivityContext } from "@microsoft/durabletask-js";

const sayHello = async (_ctx: ActivityContext, name: string): Promise<string> => {
  return `Hello, ${name}!`;
};

const helloCities: TOrchestrator = async function* (ctx: OrchestrationContext): any {
  const result1 = yield ctx.callActivity(sayHello, "Tokyo");
  const result2 = yield ctx.callActivity(sayHello, "London");
  const result3 = yield ctx.callActivity(sayHello, "Seattle");
  return [result1, result2, result3];
};
```

## Pattern Selection Guide

| Pattern | Use When |
|---------|----------|
| **Function Chaining** | Sequential steps where each depends on the previous |
| **Fan-Out/Fan-In** | Parallel processing with aggregated results |
| **Human Interaction** | Workflow pauses for external input/approval |
| **Durable Entities** | Stateful objects with operations (counters, accounts) |
| **Sub-Orchestrations** | Reusable workflow components or version isolation |
| **Eternal Orchestrations** | Long-running background processes with `continueAsNew` |

See [references/patterns.md](references/patterns.md) for detailed implementations.

## Orchestration Structure

### Basic Orchestrator

```javascript
// Orchestrator function - MUST be a generator function (function*)
// MUST be deterministic
const myOrchestrator = async function* myOrchestrator(ctx, input) {
  const step1Result = yield ctx.callActivity(step1Activity, input);
  const step2Result = yield ctx.callActivity(step2Activity, step1Result);
  return step2Result;
};
```

### Basic Activity

```javascript
// Activity function - can have side effects, I/O, non-determinism
const myActivity = async (_ctx, input) => {
  console.log(`Processing: ${input}`);
  return `Processed: ${input}`;
};
```

### Worker Registration

```javascript
const worker = createAzureManagedWorkerBuilder(connectionString)
  .addOrchestrator(myOrchestrator)
  .addActivity(step1Activity)
  .addActivity(step2Activity)
  .build();

await worker.start();

// Keep process running
setInterval(() => {}, 60_000);
```

## Critical Rules

### Orchestration Determinism

Orchestrations replay from history - all code MUST be deterministic. When an orchestration resumes, it replays all previous code to rebuild state. Non-deterministic code produces different results on replay, causing failures.

**NEVER do inside orchestrations:**
- `Date.now()`, `new Date()` → Use `ctx.currentUtcDateTime`
- `crypto.randomUUID()`, `Math.random()` → Pass random values from activities
- Direct I/O, HTTP calls, database access → Move to activities
- `setTimeout()`, `await sleep()` → Use `ctx.createTimer()`
- `process.env` that may change → Pass as input or use activities
- Non-deterministic iteration (Map, Set without sorting)

**ALWAYS use:**
- `yield ctx.callActivity()` - Call activities
- `yield ctx.callSubOrchestrator()` - Sub-orchestrations
- `yield ctx.createTimer()` - Durable delays
- `yield ctx.waitForExternalEvent()` - External events
- `ctx.currentUtcDateTime` - Current time (deterministic)
- `ctx.setCustomStatus()` - Set status

### Using yield (Generator Functions)

In JavaScript/TypeScript, orchestrator functions MUST be async generator functions (`async function*`) and use `yield` to await durable operations:

```javascript
// CORRECT - use yield with generator function
const myOrchestrator = async function* (ctx, input) {
  const result = yield ctx.callActivity(myActivity, input);
  return result;
};

// WRONG - regular async function won't work
const badOrchestrator = async (ctx, input) => {
  const result = await ctx.callActivity(myActivity, input);  // Won't work!
  return result;
};

// WRONG - forgetting yield
const alsoBad = async function* (ctx, input) {
  const result = ctx.callActivity(myActivity, input);  // Missing yield!
  return result;
};
```

### Non-Determinism Patterns (WRONG vs CORRECT)

#### Getting Current Time

```javascript
// WRONG - Date.now() returns different value on replay
const badOrchestrator = async function* (ctx) {
  const currentTime = Date.now();  // Non-deterministic!
  if (currentTime < deadline) {
    yield ctx.callActivity(processNow);
  }
};

// CORRECT - ctx.currentUtcDateTime is replayed consistently
const goodOrchestrator = async function* (ctx) {
  const currentTime = ctx.currentUtcDateTime;  // Deterministic
  if (currentTime < deadline) {
    yield ctx.callActivity(processNow);
  }
};
```

#### Random Values

```javascript
// WRONG - Math.random() produces different values on replay
const badOrchestrator = async function* (ctx) {
  const delay = Math.floor(Math.random() * 10);  // Non-deterministic!
  yield ctx.createTimer(delay);
};

// CORRECT - generate random in activity, pass to orchestrator
const getRandomDelay = async (_ctx) => {
  return Math.floor(Math.random() * 10);  // OK in activity
};

const goodOrchestrator = async function* (ctx) {
  const delay = yield ctx.callActivity(getRandomDelay);
  yield ctx.createTimer(delay);  // Deterministic
};
```

#### HTTP Calls and I/O

```javascript
// WRONG - fetch in orchestrator is non-deterministic
const badOrchestrator = async function* (ctx, url) {
  const response = await fetch(url);  // Non-deterministic!
  return await response.json();
};

// CORRECT - move I/O to activity
const fetchData = async (_ctx, url) => {
  const response = await fetch(url);  // OK in activity
  return await response.json();
};

const goodOrchestrator = async function* (ctx, url) {
  const data = yield ctx.callActivity(fetchData, url);  // Deterministic
  return data;
};
```

#### Sleeping/Delays

```javascript
// WRONG - setTimeout doesn't persist
const badOrchestrator = async function* (ctx) {
  yield ctx.callActivity(step1);
  await new Promise(resolve => setTimeout(resolve, 60000));  // Non-durable!
  yield ctx.callActivity(step2);
};

// CORRECT - ctx.createTimer is durable
const goodOrchestrator = async function* (ctx) {
  yield ctx.callActivity(step1);
  yield ctx.createTimer(60);  // Durable timer (seconds)
  yield ctx.callActivity(step2);
};
```

#### Environment Variables

```javascript
// WRONG - env var might change between replays
const badOrchestrator = async function* (ctx) {
  const apiEndpoint = process.env.API_ENDPOINT;  // Could change!
  yield ctx.callActivity(callApi, apiEndpoint);
};

// CORRECT - pass config as input or read in activity
const callApi = async (_ctx, _input) => {
  const apiEndpoint = process.env.API_ENDPOINT;  // OK in activity
  // make the call...
};

const goodOrchestrator = async function* (ctx, config) {
  const apiEndpoint = config.apiEndpoint;  // From input, deterministic
  yield ctx.callActivity(callApi, apiEndpoint);
};
```

### Error Handling

```javascript
const orchestratorWithErrorHandling = async function* (ctx, input) {
  try {
    const result = yield ctx.callActivity(riskyActivity, input);
    return result;
  } catch (error) {
    // Activity failed - implement compensation
    ctx.setCustomStatus({ error: error.message });
    yield ctx.callActivity(compensationActivity, input);
    return "Compensated";
  }
};
```

### Retry Policies

```javascript
import { RetryPolicy } from "@microsoft/durabletask-js";

const retryPolicy = new RetryPolicy({
  maxNumberOfAttempts: 3,
  firstRetryIntervalInMilliseconds: 5000,
  backoffCoefficient: 2.0,
  maxRetryIntervalInMilliseconds: 60000,
});

const orchestrator = async function* (ctx) {
  const result = yield ctx.callActivity(unreliableActivity, "data", {
    retry: retryPolicy,
  });
  return result;
};
```

## Connection & Authentication

### Connection String Formats

```javascript
// Local emulator (no auth)
"Endpoint=http://localhost:8080;Authentication=None;TaskHub=default"

// Azure with DefaultAzureCredential
"Endpoint=https://my-scheduler.region.durabletask.io;Authentication=DefaultAzure;TaskHub=my-hub"

// Azure with Managed Identity
"Endpoint=https://my-scheduler.region.durabletask.io;Authentication=ManagedIdentity;TaskHub=my-hub"

// Azure with Azure CLI
"Endpoint=https://my-scheduler.region.durabletask.io;Authentication=AzureCli;TaskHub=my-hub"
```

### Using Connection String

```javascript
import { createAzureManagedClient, createAzureManagedWorkerBuilder } from "@microsoft/durabletask-js-azuremanaged";

const connectionString = process.env.DURABLE_TASK_SCHEDULER_CONNECTION_STRING;

const client = createAzureManagedClient(connectionString);
const workerBuilder = createAzureManagedWorkerBuilder(connectionString);
```

### Using Explicit Credentials

```javascript
import { DefaultAzureCredential, ManagedIdentityCredential } from "@azure/identity";

const endpoint = process.env.ENDPOINT ?? "http://localhost:8080";
const taskHub = process.env.TASKHUB ?? "default";

// For local emulator
if (endpoint === "http://localhost:8080") {
  const connectionString = `Endpoint=${endpoint};Authentication=None;TaskHub=${taskHub}`;
  const client = createAzureManagedClient(connectionString);
} else {
  // For Azure
  const credential = new DefaultAzureCredential();
  const client = createAzureManagedClient(endpoint, taskHub, credential);
}
```

### Authentication Helper

```javascript
import { DefaultAzureCredential, ManagedIdentityCredential } from "@azure/identity";

const EMULATOR_ENDPOINT = "http://localhost:8080";

function createClient() {
  const endpoint = process.env.ENDPOINT ?? EMULATOR_ENDPOINT;
  const taskHub = process.env.TASKHUB ?? "default";
  const managedIdentityClientId = process.env.AZURE_MANAGED_IDENTITY_CLIENT_ID;

  if (endpoint === EMULATOR_ENDPOINT) {
    const connectionString = `Endpoint=${endpoint};Authentication=None;TaskHub=${taskHub}`;
    return createAzureManagedClient(connectionString);
  }

  const credential = managedIdentityClientId
    ? new ManagedIdentityCredential({ clientId: managedIdentityClientId })
    : new DefaultAzureCredential();

  return createAzureManagedClient(endpoint, taskHub, credential);
}
```

## Local Development with Emulator

```bash
# Pull and run the emulator
docker pull mcr.microsoft.com/dts/dts-emulator:latest
docker run -d -p 8080:8080 -p 8082:8082 --name dts-emulator mcr.microsoft.com/dts/dts-emulator:latest

# Dashboard available at http://localhost:8082
```

## Client Operations

```javascript
const client = createAzureManagedClient(connectionString);

// Schedule new orchestration
const instanceId = await client.scheduleNewOrchestration(myOrchestrator, input);

// Schedule with custom instance ID
const instanceId = await client.scheduleNewOrchestration(myOrchestrator, input, {
  instanceId: "my-custom-id",
});

// Wait for completion
const state = await client.waitForOrchestrationCompletion(instanceId, true, 60);

// Get current status
const state = await client.getOrchestrationState(instanceId, true);

// Raise external event
await client.raiseOrchestrationEvent(instanceId, "ApprovalEvent", approvalData);

// Terminate orchestration
await client.terminateOrchestration(instanceId, "User cancelled");

// Suspend/Resume
await client.suspendOrchestration(instanceId);
await client.resumeOrchestration(instanceId);

// Stop client when done
await client.stop();
```

## Project Setup

### package.json

```json
{
  "type": "module",
  "engines": {
    "node": ">=22.0.0"
  },
  "dependencies": {
    "@microsoft/durabletask-js": "^0.2.0",
    "@microsoft/durabletask-js-azuremanaged": "^0.2.0",
    "@azure/identity": "^4.13.0"
  }
}
```

Note: Use `"type": "module"` in package.json for ES module support with `.mjs` files or import/export syntax.

## References

- **[patterns.md](references/patterns.md)** - Detailed pattern implementations (Fan-Out/Fan-In, Human Interaction, Entities, Sub-Orchestrations)
- **[setup.md](references/setup.md)** - Azure Durable Task Scheduler provisioning and deployment
