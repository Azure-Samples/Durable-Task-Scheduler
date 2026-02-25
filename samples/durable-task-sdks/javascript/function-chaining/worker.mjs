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

const sayHello = async (_ctx, name) => {
  const safeName = typeof name === "string" && name.length ? name : "User";
  const message = `Hello ${safeName}!`;
  console.log(`sayHello -> ${message}`);
  return message;
};

const processGreeting = async (_ctx, greeting) => {
  const value = typeof greeting === "string" ? greeting : "Hello User!";
  const message = `${value} How are you today?`;
  console.log(`processGreeting -> ${message}`);
  return message;
};

const finalizeResponse = async (_ctx, response) => {
  const value = typeof response === "string" ? response : "Hello User! How are you today?";
  const message = `${value} I hope you're doing well!`;
  console.log(`finalizeResponse -> ${message}`);
  return message;
};

const functionChainingOrchestrator = async function* functionChainingOrchestrator(ctx, name) {
  const greeting = yield ctx.callActivity(sayHello, name);
  const processedGreeting = yield ctx.callActivity(processGreeting, greeting);
  const finalResponse = yield ctx.callActivity(finalizeResponse, processedGreeting);

  return finalResponse;
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
  console.log("Starting Function Chaining worker...");
  console.log(`Endpoint: ${endpoint}`);
  console.log(`Task hub: ${taskHub}`);

  worker = getWorkerBuilder()
    .addOrchestrator(functionChainingOrchestrator)
    .addActivity(sayHello)
    .addActivity(processGreeting)
    .addActivity(finalizeResponse)
    .build();

  try {
    await worker.start();
    console.log("Worker started and waiting for orchestrations...");

    setInterval(() => {
    }, 60_000);
  } catch (error) {
    console.error("Worker failed to start", error);
    await stopWorker(1);
  }
})();
