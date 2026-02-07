const { app } = require("@azure/functions");
const df = require("durable-functions");

// Activity: Say hello to a city
df.app.activity("sayHello", {
  handler: (city) => {
    return `Hello ${city}!`;
  },
});

// Orchestrator: Function chaining — sequential greetings
df.app.orchestration("chainingOrchestration", function* (context) {
  const outputs = [];
  outputs.push(yield context.df.callActivity("sayHello", "Tokyo"));
  outputs.push(yield context.df.callActivity("sayHello", "Seattle"));
  outputs.push(yield context.df.callActivity("sayHello", "London"));
  return outputs;
});

// Orchestrator: Fan-out/Fan-in — parallel greetings
df.app.orchestration("fanOutFanInOrchestration", function* (context) {
  const cities = ["Tokyo", "Seattle", "London", "Paris", "Berlin"];

  // Fan-out: schedule all activities in parallel
  const tasks = cities.map((city) => context.df.callActivity("sayHello", city));

  // Fan-in: wait for all to complete
  const results = yield context.df.Task.all(tasks);
  return results;
});

// HTTP trigger: Start chaining orchestration
app.http("StartChaining", {
  route: "StartChaining",
  methods: ["POST"],
  authLevel: "anonymous",
  extraInputs: [df.input.durableClient()],
  handler: async (request, context) => {
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
  handler: async (request, context) => {
    const client = df.getClient(context);
    const instanceId = await client.startNew("fanOutFanInOrchestration");
    context.log(`Started fan-out/fan-in orchestration with ID = '${instanceId}'.`);
    return client.createCheckStatusResponse(request, instanceId);
  },
});
