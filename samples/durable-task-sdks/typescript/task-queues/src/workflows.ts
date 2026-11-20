import type { ActivityContext, OrchestrationContext, Task } from "@microsoft/durabletask-js";

export const recordOrder = (_ctx: ActivityContext, orderId: string): string =>
  `Recorded ${orderId}`;
export const callPartner = (_ctx: ActivityContext, orderId: string): string =>
  `Partner accepted ${orderId}`;

export async function* validateOrder(
  ctx: OrchestrationContext, orderId: string,
): AsyncGenerator<Task<string>, string, string> {
  yield ctx.callActivity(recordOrder, orderId);
  return `Validated ${orderId}`;
}

export async function* processOrder(
  ctx: OrchestrationContext, orderId: string,
): AsyncGenerator<Task<string>, string, string> {
  const recorded = yield ctx.callActivity(recordOrder, orderId);
  const partner = yield ctx.callActivity(callPartner, orderId, {
    tags: { DTS_TaskQueue: "partner-api" },
  });
  const validated = yield ctx.callSubOrchestrator<string, string>(validateOrder, orderId, {
    tags: { DTS_TaskQueue: "partner-api" },
  });
  return `${recorded}; ${partner}; ${validated}`;
}
