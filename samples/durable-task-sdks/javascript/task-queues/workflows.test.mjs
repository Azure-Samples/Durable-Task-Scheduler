import assert from "node:assert/strict";
import { test } from "node:test";
import { OrchestrationStatus } from "@microsoft/durabletask-js";
import {
  InMemoryOrchestrationBackend, TestOrchestrationClient, TestOrchestrationWorker,
} from "@microsoft/durabletask-js/dist/testing/index.js";
import { callPartner, processOrder, recordOrder, validateOrder } from "./workflows.mjs";

test("parent activity inherits; partner activity and child override", async () => {
  const calls = [];
  const ctx = {
    callActivity: (...args) => { calls.push(args); return "activity"; },
    callSubOrchestrator: (...args) => { calls.push(args); return "child"; },
  };
  const workflow = processOrder(ctx, "order-001");
  await workflow.next();
  assert.deepEqual(calls[0], [recordOrder, "order-001"]);
  await workflow.next("Recorded order-001");
  assert.deepEqual(calls[1], [callPartner, "order-001", { tags: { DTS_TaskQueue: "partner-api" } }]);
  await workflow.next("Partner accepted order-001");
  assert.deepEqual(calls[2], [validateOrder, "order-001", { tags: { DTS_TaskQueue: "partner-api" } }]);
  assert.deepEqual(await workflow.next("Validated order-001"), {
    done: true,
    value: "Recorded order-001; Partner accepted order-001; Validated order-001",
  });
});

test("child activity inherits the child's queue", async () => {
  const calls = [];
  const workflow = validateOrder({
    callActivity: (...args) => { calls.push(args); return "activity"; },
  }, "order-001");
  await workflow.next();
  assert.deepEqual(calls, [[recordOrder, "order-001"]]);
  assert.deepEqual(await workflow.next("Recorded order-001"), {
    done: true, value: "Validated order-001",
  });
});

test("activity failure propagates", async () => {
  const workflow = processOrder({ callActivity: () => "activity" }, "order-001");
  await workflow.next();
  await assert.rejects(workflow.throw(new Error("recording failed")), /recording failed/);
});

test("SDK executes parent and child to completion", { timeout: 30_000 }, async () => {
  const backend = new InMemoryOrchestrationBackend();
  const worker = new TestOrchestrationWorker(backend);
  const client = new TestOrchestrationClient(backend);
  worker.addOrchestrator(processOrder);
  worker.addOrchestrator(validateOrder);
  worker.addActivity(recordOrder);
  worker.addActivity(callPartner);
  try {
    await worker.start();
    const id = await client.scheduleNewOrchestration(processOrder, "order-001");
    const state = await client.waitForOrchestrationCompletion(id, true, 20);
    assert.equal(state?.runtimeStatus, OrchestrationStatus.COMPLETED);
    assert.equal(JSON.parse(state.serializedOutput),
      "Recorded order-001; Partner accepted order-001; Validated order-001");
  } finally {
    await client.stop();
    await worker.stop();
  }
});
