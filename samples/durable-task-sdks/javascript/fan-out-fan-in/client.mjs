import { OrchestrationStatus } from "@microsoft/durabletask-js";
import { createAzureManagedClient } from "@microsoft/durabletask-js-azuremanaged";
import { DefaultAzureCredential, ManagedIdentityCredential } from "@azure/identity";

const EMULATOR_ENDPOINT = "http://localhost:8080";
const endpoint = process.env.ENDPOINT ?? EMULATOR_ENDPOINT;
const taskHub = process.env.TASKHUB ?? "default";
const managedIdentityClientId = process.env.AZURE_MANAGED_IDENTITY_CLIENT_ID;

function createClient() {
  if (endpoint === EMULATOR_ENDPOINT) {
    const connectionString = `Endpoint=${endpoint};Authentication=None;TaskHub=${taskHub}`;
    console.log("Using local emulator with no authentication");
    return createAzureManagedClient(connectionString);
  }

  const credential = managedIdentityClientId
    ? new ManagedIdentityCredential({ clientId: managedIdentityClientId })
    : new DefaultAzureCredential();

  if (managedIdentityClientId) {
    console.log(`Using managed identity with client ID: ${managedIdentityClientId}`);
  } else {
    console.log("Using DefaultAzureCredential authentication");
  }

  return createAzureManagedClient(endpoint, taskHub, credential);
}

function getWorkItemCount() {
  const input = process.argv[2];

  if (!input) {
    return 10;
  }

  const count = Number(input);

  if (!Number.isInteger(count) || count <= 0) {
    throw new Error("The optional work-item count argument must be a positive integer.");
  }

  return count;
}

(async () => {
  console.log("Starting Fan-out/Fan-in client...");
  console.log(`Endpoint: ${endpoint}`);
  console.log(`Task hub: ${taskHub}`);

  const count = getWorkItemCount();
  const workItems = Array.from({ length: count }, (_unused, index) => index + 1);

  const client = createClient();

  try {
    console.log(`Scheduling orchestration with ${count} work items...`);

    const instanceId = await client.scheduleNewOrchestration("fanOutFanInOrchestrator", workItems);

    console.log(`Started orchestration with ID: ${instanceId}`);
    console.log("Waiting for orchestration to complete...");

    const state = await client.waitForOrchestrationCompletion(instanceId, true, 120);

    if (!state) {
      console.log("No orchestration state was returned.");
      process.exitCode = 1;
      return;
    }

    if (state.runtimeStatus === OrchestrationStatus.COMPLETED) {
      console.log(`Orchestration completed with status: ${state.runtimeStatus}`);
      console.log(`Result: ${state.serializedOutput}`);
      return;
    }

    console.log(`Orchestration finished with status: ${state.runtimeStatus}`);

    if (state.runtimeStatus === OrchestrationStatus.FAILED && state.failureDetails) {
      console.log(`Failure: ${state.failureDetails.errorType} - ${state.failureDetails.message}`);
    }

    process.exitCode = 1;
  } finally {
    await client.stop();
  }
})();
