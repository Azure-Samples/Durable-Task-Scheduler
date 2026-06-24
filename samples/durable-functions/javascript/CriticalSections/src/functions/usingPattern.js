const df = require("durable-functions");

// ============================================================================
// Pattern 1 — `using` (zero-boilerplate auto-release) — JavaScript sample.
//
//   *** REQUIRES Node.js 24+ ***
//
// The `using` declaration (ECMAScript Explicit Resource Management) calls the
// lock's [Symbol.dispose]() automatically when the block scope exits — even if
// the body throws. It is the cleanest pattern: no try/finally, no manual
// release(). Because it relies on native `using` support, it needs Node.js 24+.
//
// This orchestration lives in its own file so the other three patterns in
// ./criticalSections.js still load on Node.js 18–22. If you are on an older
// Node.js version, delete this file before running `func start` — the `using`
// pattern (and the POST /api/transfer/using route) will then be unavailable.
//
// The `account` entity, `sendReceipt` activity, and the StartTransfer HTTP
// trigger (which routes POST /api/transfer/using here) are registered in
// ./criticalSections.js, which is always loaded alongside this file.
// ============================================================================

df.app.orchestration("transferUsing", function* (context) {
  const { fromKey, toKey, amount } = context.df.getInput();
  const src = new df.EntityId("account", fromKey);
  const dst = new df.EntityId("account", toKey);

  // Seed known starting balances so the sample is self-contained (outside the lock).
  yield context.df.callEntity(src, "set", 100);
  yield context.df.callEntity(dst, "set", 0);

  let lockedInside;
  {
    using lock = yield context.df.lock(src, dst); // auto-released at block exit
    const lockState = context.df.isLocked();
    lockedInside = lockState.isLocked; // true (lockState.ownedLocks lists the locked entities)
    yield context.df.callEntity(src, "add", -amount);
    yield context.df.callEntity(dst, "add", amount);
  } // <- lock[Symbol.dispose]() runs here, releasing the lock

  const { isLocked: lockedAfter } = context.df.isLocked(); // false
  const fromBalance = yield context.df.callEntity(src, "get");
  const toBalance = yield context.df.callEntity(dst, "get");

  return {
    pattern: "using",
    lockedInside,
    lockedAfter,
    balances: { [fromKey]: fromBalance, [toKey]: toBalance },
  };
});
