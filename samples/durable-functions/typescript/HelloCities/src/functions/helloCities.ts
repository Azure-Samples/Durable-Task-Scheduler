import { app, HttpRequest, HttpResponse, InvocationContext } from "@azure/functions";
import * as df from "durable-functions";
import { OrchestrationContext, OrchestrationHandler } from "durable-functions";

// Activity: Say hello to a city
df.app.activity("sayHello", {
  handler: (city: string): string => {
    return `Hello ${city}!`;
  },
});

// Orchestrator: Function chaining — sequential greetings
const chainingOrchestrator: OrchestrationHandler = function* (context: OrchestrationContext) {
  const outputs: string[] = [];
  outputs.push(yield context.df.callActivity("sayHello", "Tokyo"));
  outputs.push(yield context.df.callActivity("sayHello", "Seattle"));
  outputs.push(yield context.df.callActivity("sayHello", "London"));
  return outputs;
};
df.app.orchestration("chainingOrchestration", chainingOrchestrator);

// Orchestrator: Fan-out/Fan-in — parallel greetings
const fanOutFanInOrchestrator: OrchestrationHandler = function* (context: OrchestrationContext) {
  const cities: string[] = ["Tokyo", "Seattle", "London", "Paris", "Berlin"];

  // Fan-out: schedule all activities in parallel
  const tasks = cities.map((city) => context.df.callActivity("sayHello", city));

  // Fan-in: wait for all to complete
  const results: string[] = yield context.df.Task.all(tasks);
  return results;
};
df.app.orchestration("fanOutFanInOrchestration", fanOutFanInOrchestrator);

// HTTP trigger: Start chaining orchestration
app.http("StartChaining", {
  route: "StartChaining",
  methods: ["POST"],
  authLevel: "anonymous",
  extraInputs: [df.input.durableClient()],
  handler: async (request: HttpRequest, context: InvocationContext): Promise<HttpResponse> => {
    const client = df.getClient(context);
    const instanceId = await client.startNew("chainingOrchestration");
    context.log(`Started chaining orchestration with ID = '${instanceId}'.`);
    return client.createCheckStatusResponse(request, instanceId);
  },
});

// HTTP trigger: Start fan-out/fan-in orchestration
app.http("StartFanOutFanIn", {
  route: "StartFanOutFanIn",
  methods: ["POST"],
  authLevel: "anonymous",
  extraInputs: [df.input.durableClient()],
  handler: async (request: HttpRequest, context: InvocationContext): Promise<HttpResponse> => {
    const client = df.getClient(context);
    const instanceId = await client.startNew("fanOutFanInOrchestration");
    context.log(`Started fan-out/fan-in orchestration with ID = '${instanceId}'.`);
    return client.createCheckStatusResponse(request, instanceId);
  },
});
