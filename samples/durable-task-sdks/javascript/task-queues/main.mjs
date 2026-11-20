import { OrchestrationStatus } from "@microsoft/durabletask-js";
import {
  createAzureManagedClient,
  createAzureManagedWorkerBuilder,
} from "@microsoft/durabletask-js-azuremanaged";
import { callPartner, processOrder, recordOrder, validateOrder } from "./workflows.mjs";

const connectionString = process.env.DTS_CONNECTION_STRING
  ?? "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";
const orderId = process.argv[2] ?? "order-001";
if (!orderId.trim()) {
  throw new Error("Order ID must not be empty.");
}

const worker = createAzureManagedWorkerBuilder(connectionString)
  .addOrchestrator(processOrder)
  .addOrchestrator(validateOrder)
  .addActivity(recordOrder)
  .addActivity(callPartner)
  .build();
const client = createAzureManagedClient(connectionString);
try {
  await worker.start();
  const instanceId = await client.scheduleNewOrchestration(processOrder, orderId, {
    tags: { DTS_TaskQueue: "orders" },
  });
  console.log(`Started ${instanceId} with DTS_TaskQueue=orders`);
  const state = await client.waitForOrchestrationCompletion(instanceId, true, 120);
  if (!state || state.runtimeStatus !== OrchestrationStatus.COMPLETED) {
    throw new Error(`Instance ${instanceId} did not complete successfully: ${state?.runtimeStatus}`);
  }
  const output = JSON.parse(state.serializedOutput);
  const expected = `Recorded ${orderId}; Partner accepted ${orderId}; Validated ${orderId}`;
  if (output !== expected) {
    throw new Error(`Unexpected output: ${JSON.stringify(output)}`);
  }
  console.log(output);
} finally {
  try {
    await client.stop();
  } finally {
    await worker.stop();
  }
}
