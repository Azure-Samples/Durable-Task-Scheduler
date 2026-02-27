# Durable Task JavaScript/TypeScript Patterns

Detailed implementation patterns for the Durable Task JavaScript/TypeScript SDK.

## Function Chaining

Sequential execution where each step depends on the previous:

```javascript
const sayHello = async (_ctx, name) => {
  const message = `Hello ${name}!`;
  console.log(`sayHello -> ${message}`);
  return message;
};

const processGreeting = async (_ctx, greeting) => {
  const message = `${greeting} How are you today?`;
  console.log(`processGreeting -> ${message}`);
  return message;
};

const finalizeResponse = async (_ctx, response) => {
  const message = `${response} I hope you're doing well!`;
  console.log(`finalizeResponse -> ${message}`);
  return message;
};

const functionChainingOrchestrator = async function* functionChainingOrchestrator(ctx, name) {
  const greeting = yield ctx.callActivity(sayHello, name);
  const processedGreeting = yield ctx.callActivity(processGreeting, greeting);
  const finalResponse = yield ctx.callActivity(finalizeResponse, processedGreeting);
  return finalResponse;
};

// Worker registration
const worker = createAzureManagedWorkerBuilder(connectionString)
  .addOrchestrator(functionChainingOrchestrator)
  .addActivity(sayHello)
  .addActivity(processGreeting)
  .addActivity(finalizeResponse)
  .build();
```

## Fan-Out/Fan-In

Parallel processing with aggregated results:

```javascript
import { whenAll } from "@microsoft/durabletask-js";

const getWorkItems = async (_ctx) => {
  // Return a list of work items to process
  return ["item1", "item2", "item3", "item4", "item5"];
};

const processWorkItem = async (_ctx, item) => {
  // Process a single work item
  console.log(`Processing: ${item}`);
  return Math.floor(Math.random() * 100);
};

const fanOutFanInOrchestrator = async function* (ctx) {
  // Get the list of work items
  const workItems = yield ctx.callActivity(getWorkItems);

  // Fan-out: schedule all work items in parallel
  const tasks = [];
  for (const item of workItems) {
    tasks.push(ctx.callActivity(processWorkItem, item));
  }

  // Fan-in: wait for all to complete
  const results = yield whenAll(tasks);

  // Return aggregated results
  return results.reduce((sum, val) => sum + val, 0);
};

// Worker registration
const worker = createAzureManagedWorkerBuilder(connectionString)
  .addOrchestrator(fanOutFanInOrchestrator)
  .addActivity(getWorkItems)
  .addActivity(processWorkItem)
  .build();
```

### Batched Fan-Out (Large Scale)

For large numbers of items, process in batches:

```javascript
const batchedFanOutOrchestrator = async function* (ctx) {
  const workItems = yield ctx.callActivity(getWorkItems);

  const batchSize = 10;
  const allResults = [];

  for (let i = 0; i < workItems.length; i += batchSize) {
    const batch = workItems.slice(i, i + batchSize);
    const tasks = batch.map((item) => ctx.callActivity(processWorkItem, item));
    const batchResults = yield whenAll(tasks);
    allResults.push(...batchResults);
  }

  return { total: allResults.reduce((sum, val) => sum + val, 0) };
};
```

## Human Interaction

Workflow that waits for external approval with timeout:

```javascript
import { whenAny } from "@microsoft/durabletask-js";

const sendApprovalRequest = async (_ctx, order) => {
  console.log(`Approval needed for ${order.product} ($${order.cost})`);
};

const placeOrder = async (_ctx, order) => {
  return `Order placed: ${order.quantity}x ${order.product}`;
};

const purchaseOrderWorkflow = async function* (ctx, order) {
  // Auto-approve small orders
  if (order.cost < 1000) {
    return "Auto-approved";
  }

  // Request approval for larger orders
  yield ctx.callActivity(sendApprovalRequest, order);

  // Wait for approval OR timeout (whichever comes first)
  const approvalEvent = ctx.waitForExternalEvent("approval_received");
  const timeoutEvent = ctx.createTimer(24 * 60 * 60); // 24 hours in seconds

  const winner = yield whenAny([approvalEvent, timeoutEvent]);

  if (winner === timeoutEvent) {
    return "Cancelled";
  }

  // Order was approved
  yield ctx.callActivity(placeOrder, order);
  const approvalDetails = approvalEvent.getResult();
  return `Approved by ${approvalDetails.approver}`;
};

// Raising the approval event from client
await client.raiseOrchestrationEvent(instanceId, "approval_received", {
  approver: "manager@company.com",
});
```

## Durable Timers

Schedule delayed execution:

```javascript
const delayedWorkflow = async function* (ctx) {
  yield ctx.callActivity(startActivity);

  // Wait for 5 minutes (survives restarts)
  yield ctx.createTimer(5 * 60); // seconds

  yield ctx.callActivity(continueActivity);
  return "Done";
};
```

## Sub-Orchestrations

Compose orchestrations from smaller pieces:

```javascript
const childOrchestration = async function* childOrchestration(ctx, data) {
  const result1 = yield ctx.callActivity(activityA, data);
  const result2 = yield ctx.callActivity(activityB, result1);
  return result2;
};

const parentOrchestration = async function* parentOrchestration(ctx, items) {
  // Call child orchestrations in parallel
  const tasks = items.map((item) =>
    ctx.callSubOrchestrator(childOrchestration, item)
  );
  const results = yield whenAll(tasks);
  return results;
};

// Registration - both must be registered
const worker = createAzureManagedWorkerBuilder(connectionString)
  .addOrchestrator(parentOrchestration)
  .addOrchestrator(childOrchestration)
  .addActivity(activityA)
  .addActivity(activityB)
  .build();
```

## Durable Entities

Stateful objects with operations:

### Class-Based Entity

```javascript
import { TaskEntity } from "@microsoft/durabletask-js";

class CounterEntity extends TaskEntity {
  add(amount) {
    this.state.value += amount;
    return this.state.value;
  }

  get() {
    return this.state.value;
  }

  reset() {
    this.state.value = 0;
  }

  initializeState() {
    return { value: 0 };
  }
}

// Register with the worker
worker.addNamedEntity("Counter", () => new CounterEntity());
```

### TypeScript Entity

```typescript
import { TaskEntity } from "@microsoft/durabletask-js";

interface CounterState {
  value: number;
}

class CounterEntity extends TaskEntity<CounterState> {
  add(amount: number): number {
    this.state.value += amount;
    return this.state.value;
  }

  get(): number {
    return this.state.value;
  }

  reset(): void {
    this.state.value = 0;
  }

  protected initializeState(): CounterState {
    return { value: 0 };
  }
}
```

## Eternal Orchestrations (Continue-As-New)

Long-running processes that periodically restart:

```javascript
const eternalOrchestration = async function* eternalOrchestration(ctx, iteration) {
  // Do periodic work
  yield ctx.callActivity(periodicWork, iteration);

  // Wait before next iteration
  yield ctx.createTimer(5 * 60); // 5 minutes

  // Restart with new iteration count
  // This prevents history from growing unbounded
  ctx.continueAsNew(iteration + 1);
};

// Start with iteration 0
await client.scheduleNewOrchestration(eternalOrchestration, 0);
```

## Versioning

Handle breaking changes gracefully using orchestration versioning:

```javascript
const versionedOrchestrator = async function* versionedOrchestrator(ctx, input) {
  const version = ctx.version;

  // v1.0.0+: Always run this
  const result1 = yield ctx.callActivity(step1, input);

  // v2.0.0+: Added in version 2
  if (ctx.compareVersionTo("2.0.0") >= 0) {
    const result2 = yield ctx.callActivity(newStep, result1);
    return result2;
  }

  return result1;
};
```

### Version Match Strategies

```javascript
import { VersionMatchStrategy, VersionFailureStrategy } from "@microsoft/durabletask-js";

const worker = createAzureManagedWorkerBuilder(connectionString, {
  versioning: {
    version: "2.0.0",
    defaultVersion: "2.0.0",
    matchStrategy: VersionMatchStrategy.CURRENT_OR_OLDER,
    failureStrategy: VersionFailureStrategy.FAIL,
  },
})
  .addOrchestrator(versionedOrchestrator)
  .build();
```
