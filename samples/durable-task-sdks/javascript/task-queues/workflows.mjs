export const recordOrder = async (_ctx, orderId) => `Recorded ${orderId}`;
export const callPartner = async (_ctx, orderId) => `Partner accepted ${orderId}`;

export const validateOrder = async function* validateOrder(ctx, orderId) {
  yield ctx.callActivity(recordOrder, orderId);
  return `Validated ${orderId}`;
};

export const processOrder = async function* processOrder(ctx, orderId) {
  const recorded = yield ctx.callActivity(recordOrder, orderId);
  const partner = yield ctx.callActivity(callPartner, orderId, {
    tags: { DTS_TaskQueue: "partner-api" },
  });
  const validated = yield ctx.callSubOrchestrator(validateOrder, orderId, {
    tags: { DTS_TaskQueue: "partner-api" },
  });
  return `${recorded}; ${partner}; ${validated}`;
};
