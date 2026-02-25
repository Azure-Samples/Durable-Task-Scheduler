import { OrchestrationStatus } from "@microsoft/durabletask-js";
import { createAzureManagedClient } from "@microsoft/durabletask-js-azuremanaged";
import { DefaultAzureCredential, ManagedIdentityCredential } from "@azure/identity";

const EMULATOR_ENDPOINT = "http://localhost:8080";
const endpoint = process.env.ENDPOINT ?? EMULATOR_ENDPOINT;
const taskHub = process.env.TASKHUB ?? "default";
const managedIdentityClientId = process.env.AZURE_MANAGED_IDENTITY_CLIENT_ID;

const TOTAL_ORCHESTRATIONS = Number(process.env.TOTAL_ORCHESTRATIONS ?? 20);
const INTERVAL_SECONDS = Number(process.env.ORCHESTRATION_INTERVAL ?? 5);

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

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

(async () => {
  console.log("Starting Function Chaining client...");
  console.log(`Endpoint: ${endpoint}`);
  console.log(`Task hub: ${taskHub}`);

  const baseName = process.argv[2] ?? "User";
  const client = createClient();
  const orchestrationIds = [];

  let completed = 0;
  let failed = 0;

  try {
    for (let index = 0; index < TOTAL_ORCHESTRATIONS; index += 1) {
      const orchestrationInput = `${baseName}_${index + 1}`;
      console.log(`Scheduling orchestration ${index + 1}/${TOTAL_ORCHESTRATIONS}: ${orchestrationInput}`);

      const instanceId = await client.scheduleNewOrchestration("functionChainingOrchestrator", orchestrationInput);
      orchestrationIds.push(instanceId);
      console.log(`Scheduled orchestration ID: ${instanceId}`);

      if (index < TOTAL_ORCHESTRATIONS - 1) {
        await sleep(INTERVAL_SECONDS * 1000);
      }
    }

    console.log(`All ${orchestrationIds.length} orchestrations scheduled. Waiting for completion...`);

    for (const instanceId of orchestrationIds) {
      const state = await client.waitForOrchestrationCompletion(instanceId, true, 120);

      if (!state) {
        console.log(`No orchestration state returned for ${instanceId}`);
        failed += 1;
        continue;
      }

      if (state.runtimeStatus === OrchestrationStatus.COMPLETED) {
        completed += 1;
        console.log(`Completed ${instanceId} -> ${state.serializedOutput}`);
      } else {
        failed += 1;
        console.log(`Orchestration ${instanceId} finished with status: ${state.runtimeStatus}`);
      }
    }

    console.log(`FINAL RESULTS: ${completed} completed, ${failed} failed, ${orchestrationIds.length} total orchestrations`);
  } finally {
    await client.stop();
  }
})();
