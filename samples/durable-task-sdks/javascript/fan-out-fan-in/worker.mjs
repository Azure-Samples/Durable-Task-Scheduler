import { whenAll } from "@microsoft/durabletask-js";
import { createAzureManagedWorkerBuilder } from "@microsoft/durabletask-js-azuremanaged";
import { DefaultAzureCredential, ManagedIdentityCredential } from "@azure/identity";

const EMULATOR_ENDPOINT = "http://localhost:8080";
const endpoint = process.env.ENDPOINT ?? EMULATOR_ENDPOINT;
const taskHub = process.env.TASKHUB ?? "default";
const managedIdentityClientId = process.env.AZURE_MANAGED_IDENTITY_CLIENT_ID;

function getWorkerBuilder() {
  if (endpoint === EMULATOR_ENDPOINT) {
    const connectionString = `Endpoint=${endpoint};Authentication=None;TaskHub=${taskHub}`;
    console.log("Using local emulator with no authentication");
    return createAzureManagedWorkerBuilder(connectionString);
  }

  const credential = managedIdentityClientId
    ? new ManagedIdentityCredential({ clientId: managedIdentityClientId })
    : new DefaultAzureCredential();

  if (managedIdentityClientId) {
    console.log(`Using managed identity with client ID: ${managedIdentityClientId}`);
  } else {
    console.log("Using DefaultAzureCredential authentication");
  }

  return createAzureManagedWorkerBuilder(endpoint, taskHub, credential);
}

const processWorkItem = async (_ctx, workItem) => {
  const normalizedItem = Number(workItem);
  const delayMs = 500 + Math.floor(Math.random() * 1500);
  console.log(`Processing work item ${normalizedItem} (delay ${delayMs}ms)`);

  await new Promise((resolve) => setTimeout(resolve, delayMs));

  return {
    item: normalizedItem,
    result: normalizedItem * normalizedItem,
  };
};

const aggregateResults = async (_ctx, results) => {
  const sum = results.reduce((accumulator, current) => accumulator + current.result, 0);

  return {
    totalItems: results.length,
    sum,
    average: results.length ? sum / results.length : 0,
    results,
  };
};

const fanOutFanInOrchestrator = async function* fanOutFanInOrchestrator(ctx, workItems) {
  const items = Array.isArray(workItems) ? workItems : [];
  const tasks = items.map((item) => ctx.callActivity(processWorkItem, item));

  const processedResults = yield whenAll(tasks);
  const finalResult = yield ctx.callActivity(aggregateResults, processedResults);

  return finalResult;
};

let worker;

async function stopWorker(exitCode = 0) {
  if (worker) {
    console.log("Stopping worker...");
    await worker.stop();
  }

  process.exit(exitCode);
}

process.on("SIGINT", async () => {
  await stopWorker(0);
});

process.on("SIGTERM", async () => {
  await stopWorker(0);
});

(async () => {
  console.log("Starting Fan-out/Fan-in worker...");
  console.log(`Endpoint: ${endpoint}`);
  console.log(`Task hub: ${taskHub}`);

  worker = getWorkerBuilder()
    .addOrchestrator(fanOutFanInOrchestrator)
    .addActivity(processWorkItem)
    .addActivity(aggregateResults)
    .build();

  try {
    await worker.start();
    console.log("Worker started and waiting for orchestrations...");

    setInterval(() => {
      // Keep process running for worker mode
    }, 60_000);
  } catch (error) {
    console.error("Worker failed to start", error);
    await stopWorker(1);
  }
})();
