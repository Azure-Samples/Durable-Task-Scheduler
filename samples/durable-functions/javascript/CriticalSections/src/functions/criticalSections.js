const { app } = require("@azure/functions");
const df = require("durable-functions");

// ============================================================================
// Critical sections (entity locks) — JavaScript sample.
//
// Feature: context.df.lock(...) acquires a critical section over one or more
// entities so that only one orchestration at a time can operate on them. This
// is the building block for safe, atomic multi-entity operations such as a
// money transfer between two accounts.
//
//   - SDK:        durable-functions 3.4.0+
//   - Extension:  Microsoft.Azure.WebJobs.Extensions.DurableTask 3.13.0+
//                 (installed manually — see README; not in extension bundles yet)
//   - Backend:    Durable Task Scheduler (azureManaged)
//
// This file demonstrates three lock-release patterns that work on Node.js 18+:
//   2. try / finally               -> explicit release in a finally block
//   3. Implicit                    -> no release; extension frees it at orch end
//   4. try / finally + early release -> release early, then do non-critical work
//
// The fourth pattern, `using` (zero-boilerplate auto-release), requires
// Node.js 24+ and lives in ./usingPattern.js.
// ============================================================================

// ----------------------------------------------------------------------------
// Account entity — stores a numeric balance and supports set / add / get.
// ----------------------------------------------------------------------------
df.app.entity("account", function (context) {
  let balance = context.df.getState(() => 0);
  switch (context.df.operationName) {
    case "set":
      balance = context.df.getInput();
      break;
    case "add":
      balance += context.df.getInput();
      break;
    case "get":
      context.df.return(balance);
      break;
  }
  context.df.setState(balance);
});

// ----------------------------------------------------------------------------
// Activity used by the early-release pattern to do work AFTER the lock is freed.
// ----------------------------------------------------------------------------
df.app.activity("sendReceipt", {
  handler: (input) => `receipt:${input.fromKey}->${input.toKey}:${input.amount}`,
});

// ============================================================================
// Pattern 2 — try / finally (explicit release in finally; Node 18+)
//
// The lock is held for the whole try block and released in finally, so it is
// freed even if the body throws. This is the universal pattern: it works on
// any Node.js/TypeScript version.
// ============================================================================
df.app.orchestration("transferTryFinally", function* (context) {
  const { fromKey, toKey, amount } = context.df.getInput();
  const src = new df.EntityId("account", fromKey);
  const dst = new df.EntityId("account", toKey);

  // Seed known starting balances so the sample is self-contained (outside the lock).
  yield context.df.callEntity(src, "set", 100);
  yield context.df.callEntity(dst, "set", 0);

  const lock = yield context.df.lock(src, dst);
  const { isLocked: lockedInside } = context.df.isLocked(); // true — the lock is held
  try {
    yield context.df.callEntity(src, "add", -amount);
    yield context.df.callEntity(dst, "add", amount);
  } finally {
    lock.release();
  }

  const { isLocked: lockedAfter } = context.df.isLocked(); // released -> false
  return yield* summarize(context, "try/finally", src, dst, fromKey, toKey, {
    lockedInside,
    lockedAfter,
  });
});

// ============================================================================
// Pattern 3 — Implicit (no release; extension frees the lock at orch end; Node 18+)
//
// No release() call. The lock is held until the orchestration terminates, at
// which point the extension releases it automatically. Always exception-safe,
// but the lock is held for the entire orchestration — the longest hold time.
// ============================================================================
df.app.orchestration("transferImplicit", function* (context) {
  const { fromKey, toKey, amount } = context.df.getInput();
  const src = new df.EntityId("account", fromKey);
  const dst = new df.EntityId("account", toKey);

  yield context.df.callEntity(src, "set", 100);
  yield context.df.callEntity(dst, "set", 0);

  yield context.df.lock(src, dst);
  const { isLocked: lockedInside } = context.df.isLocked(); // true
  yield context.df.callEntity(src, "add", -amount);
  yield context.df.callEntity(dst, "add", amount);
  // No release(). lockedAfter stays true here; the extension frees the lock
  // when the orchestration completes.

  const { isLocked: lockedAfter } = context.df.isLocked(); // still true
  return yield* summarize(context, "implicit", src, dst, fromKey, toKey, {
    lockedInside,
    lockedAfter,
  });
});

// ============================================================================
// Pattern 4 — try / finally + early release (Node 18+)
//
// Release the lock as soon as the critical work is done, then continue with
// non-critical work (here, sending a receipt) while OTHER orchestrations are
// free to acquire the lock. The finally block's release() is an idempotent
// safety net in case the body throws before the early release runs.
// ============================================================================
df.app.orchestration("transferEarlyRelease", function* (context) {
  const { fromKey, toKey, amount } = context.df.getInput();
  const src = new df.EntityId("account", fromKey);
  const dst = new df.EntityId("account", toKey);

  yield context.df.callEntity(src, "set", 100);
  yield context.df.callEntity(dst, "set", 0);

  const lock = yield context.df.lock(src, dst);
  const { isLocked: lockedInside } = context.df.isLocked(); // true — the lock is held
  try {
    yield context.df.callEntity(src, "add", -amount);
    yield context.df.callEntity(dst, "add", amount);

    lock.release(); // release early — other orchestrations can now proceed
    const { isLocked: lockedAfterEarlyRelease } = context.df.isLocked(); // false

    // Non-critical work done AFTER the lock is freed.
    const receipt = yield context.df.callActivity("sendReceipt", { fromKey, toKey, amount });

    const { isLocked: lockedAfter } = context.df.isLocked(); // false
    return yield* summarize(context, "try/finally+early", src, dst, fromKey, toKey, {
      lockedInside,
      lockedAfterEarlyRelease,
      receipt,
      lockedAfter,
    });
  } finally {
    lock.release(); // idempotent safety net; no-op if already released
  }
});

// ----------------------------------------------------------------------------
// Helper: read the final balances (unlocked get calls) and build the result.
// Implemented as a generator so it can `yield` durable calls from the caller.
// ----------------------------------------------------------------------------
function* summarize(context, pattern, src, dst, fromKey, toKey, extra) {
  const fromBalance = yield context.df.callEntity(src, "get");
  const toBalance = yield context.df.callEntity(dst, "get");
  return {
    pattern,
    ...extra,
    balances: { [fromKey]: fromBalance, [toKey]: toBalance },
  };
}

// Map the {pattern} route value to the orchestration name. The `using`
// orchestration (transferUsing) is registered in ./usingPattern.js, which is
// loaded alongside this file on Node.js 24+.
const PATTERN_TO_ORCH = {
  using: "transferUsing",
  "try-finally": "transferTryFinally",
  implicit: "transferImplicit",
  "early-release": "transferEarlyRelease",
};

// ============================================================================
// HTTP trigger — start a transfer using the chosen release pattern.
//
//   POST /api/transfer/try-finally
//   POST /api/transfer/implicit
//   POST /api/transfer/early-release
//
// Optional query params: ?from=alice&to=bob&amount=30 (defaults shown).
// ============================================================================
app.http("StartTransfer", {
  route: "transfer/{pattern}",
  methods: ["POST"],
  authLevel: "anonymous",
  extraInputs: [df.input.durableClient()],
  handler: async (request, context) => {
    const client = df.getClient(context);
    const pattern = request.params.pattern;
    const orchestration = PATTERN_TO_ORCH[pattern];
    if (!orchestration) {
      return {
        status: 400,
        jsonBody: { error: `Unknown pattern '${pattern}'.`, validPatterns: Object.keys(PATTERN_TO_ORCH) },
      };
    }

    const fromKey = request.query.get("from") || "alice";
    const toKey = request.query.get("to") || "bob";
    const amount = parseInt(request.query.get("amount") || "30", 10);

    const instanceId = await client.startNew(orchestration, { input: { fromKey, toKey, amount } });
    context.log(`Started '${orchestration}' (${pattern}) transfer with ID = '${instanceId}'.`);
    return client.createCheckStatusResponse(request, instanceId);
  },
});

// ============================================================================
// HTTP trigger — read an account balance.  GET /api/balance?key=alice
// ============================================================================
app.http("GetBalance", {
  route: "balance",
  methods: ["GET"],
  authLevel: "anonymous",
  extraInputs: [df.input.durableClient()],
  handler: async (request, context) => {
    const client = df.getClient(context);
    const key = request.query.get("key") || "alice";
    const state = await client.readEntityState(new df.EntityId("account", key));
    return { status: 200, jsonBody: { key, exists: state.entityExists, balance: state.entityState ?? null } };
  },
});
