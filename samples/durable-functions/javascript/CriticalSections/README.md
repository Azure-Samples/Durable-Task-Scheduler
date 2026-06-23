# Critical Sections — Durable Functions JavaScript

JavaScript | Durable Functions | Durable Task Scheduler

## Description

This sample demonstrates **critical sections** — the ability to lock one or more
[durable entities](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-entities)
so that only one orchestration at a time can operate on them. Locks make
multi-entity operations (like transferring money between two accounts) safe and
atomic.

The lock API was added in **durable-functions 3.4.0** and requires the
**Microsoft.Azure.WebJobs.Extensions.DurableTask 3.13.0** extension. Acquire a
lock with `context.df.lock(...)`, which returns a `DurableLock` you can release
explicitly, dispose automatically, or let the extension free for you.

```js
const lock = yield context.df.lock(src, dst);
try {
  yield context.df.callEntity(src, "add", -amount);
  yield context.df.callEntity(dst, "add", amount);
} finally {
  lock.release();
}
```

### Release patterns demonstrated

| Pattern | Boilerplate | Lock released on error? | Hold time | Min Node | File |
|---------|-------------|-------------------------|-----------|----------|------|
| `using` | None | ✅ Automatic | Block scope | 24+ | [usingPattern.js](src/functions/usingPattern.js) |
| `try / finally` | 3 lines | ✅ If `release()` is in `finally` | Block scope | 18+ | [criticalSections.js](src/functions/criticalSections.js) |
| Implicit (no release) | None | ✅ Always | **Entire orchestration** | 18+ | [criticalSections.js](src/functions/criticalSections.js) |
| `try / finally` + early `release()` | 4 lines | ✅ + minimal hold time | Until first release | 18+ | [criticalSections.js](src/functions/criticalSections.js) |

> **The `using` pattern needs Node.js 24+.** It uses the native `using`
> declaration (ECMAScript Explicit Resource Management), which only parses on
> Node.js 24+. It is isolated in [usingPattern.js](src/functions/usingPattern.js)
> so the other three patterns still load on Node.js 18–22. **If you are on
> Node.js 18–22, delete `src/functions/usingPattern.js` before running.**
> (TypeScript users can use `using` on Node.js 18+ — see the TypeScript sample,
> where TypeScript compiles `using` down for older runtimes.)

## Prerequisites

1. [Node.js 18+](https://nodejs.org/) (Node.js 24+ to run the `using` pattern)
2. [Azure Functions Core Tools v4](https://learn.microsoft.com/azure/azure-functions/functions-run-local) (4.0.5382 or higher)
3. [.NET SDK 8.0+](https://dotnet.microsoft.com/download) (to build/install the extension)
4. [Docker](https://www.docker.com/products/docker-desktop/) (for the Durable Task Scheduler emulator)

## Install the Durable extension manually

The 3.13.0 extension that powers critical sections is **not yet in the Azure
Functions extension bundles**, so this sample does **not** use a bundle. Instead,
it ships an [extensions.csproj](extensions.csproj) that references the required
packages, and you build it once to generate `bin/extensions.json`:

```bash
cd samples/durable-functions/javascript/CriticalSections
dotnet build extensions.csproj -o bin
```

This restores from nuget.org and writes the extension assemblies plus
`extensions.json` into `bin/`, where the Functions host loads them. You only need
to run this once (or whenever you change [extensions.csproj](extensions.csproj)).

> **Why no `extensionBundle` in host.json?** Extension bundles pin a fixed set of
> extension versions. Because 3.13.0 isn't in a published bundle yet at the time of publishing this sample,
> we remove the `extensionBundle` section and install the extension explicitly. Once a
> bundle ships with 3.13.0+, you can switch back to a bundle and delete
> `extensions.csproj`, `NuGet.Config`, and the `bin/`/`obj/` folders. Check the
> [Extension Bundles Release](https://github.com/Azure/azure-functions-extension-bundles/releases)
> page to check if 3.13.0 is published.

The extension packages installed (see [extensions.csproj](extensions.csproj)):

| Package | Version | Purpose |
|---------|---------|---------|
| `Microsoft.Azure.WebJobs.Extensions.DurableTask` | 3.13.0 | Durable extension with critical-section (lock) support |
| `Microsoft.Azure.WebJobs.Extensions.DurableTask.AzureManaged` | 1.9.0 | Durable Task Scheduler (azureManaged) backend |
| `Microsoft.Azure.WebJobs.Script.ExtensionsMetadataGenerator` | 4.0.1 | Generates `bin/extensions.json` |

## Run the sample

1. Start the Durable Task Scheduler emulator:

   ```bash
   docker run -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
   ```

2. Install the extension (once) and the npm dependencies, then start the host:

   ```bash
   dotnet build extensions.csproj -o bin
   npm install
   func start
   ```

3. Start a transfer with one of the release patterns:

   ```bash
   curl -X POST http://localhost:7071/api/transfer/try-finally
   curl -X POST http://localhost:7071/api/transfer/implicit
   curl -X POST http://localhost:7071/api/transfer/early-release
   curl -X POST http://localhost:7071/api/transfer/using      # Node.js 24+ only
   ```

   Optional query parameters: `?from=alice&to=bob&amount=30` (defaults shown).

4. Follow the `statusQueryGetUri` returned by the call to see the orchestration
   output, or read an account balance directly:

   ```bash
   curl "http://localhost:7071/api/balance?key=alice"
   curl "http://localhost:7071/api/balance?key=bob"
   ```

5. View instances in the dashboard: http://localhost:8082

## Expected output

Each transfer seeds `alice = 100` and `bob = 0`, then moves `amount` from `alice`
to `bob` inside a critical section. The orchestration output shows the lock state
and the resulting balances, for example (`try-finally`):

```json
{
  "pattern": "try/finally",
  "lockedInside": true,
  "lockedAfter": false,
  "balances": { "alice": 70, "bob": 30 }
}
```

- `lockedInside` is `true` while the critical section is active.
- `lockedAfter` is `false` once the lock is released — **except** for the
  `implicit` pattern, where it stays `true` because the extension only frees the
  lock when the orchestration ends.
- The `early-release` result additionally includes `lockedAfterEarlyRelease`
  (`false`) and a `receipt`, showing that non-critical work ran after the lock
  was freed.

## Learn more

- [Durable entities](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-entities)
- [Durable Functions JavaScript API reference](https://learn.microsoft.com/javascript/api/durable-functions/)
- [Durable Task Scheduler documentation](https://aka.ms/dts-documentation)
