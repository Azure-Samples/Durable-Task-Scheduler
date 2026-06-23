# Critical Sections — Durable Functions TypeScript

TypeScript | Durable Functions | Durable Task Scheduler

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

```ts
using lock = yield context.df.lock(src, dst);
yield context.df.callEntity(src, "add", -amount);
yield context.df.callEntity(dst, "add", amount);
// lock auto-released at block scope exit, even on throw
```

### Release patterns demonstrated

All four patterns live in [criticalSections.ts](src/functions/criticalSections.ts):

| Pattern | Boilerplate | Exception-safe? | Hold time | Min TypeScript | Min Node |
|---------|-------------|-----------------|-----------|----------------|----------|
| `using` | None | ✅ Automatic | Block scope | 5.2+ | 18+ |
| `try / finally` | 3 lines | ✅ If written correctly | Block scope | Any | 18+ |
| Implicit (no release) | None | ✅ Always | **Entire orchestration** | Any | 18+ |
| `try / finally` + early `release()` | 4 lines | ✅ + minimal hold time | Until first release | Any | 18+ |

> **The `using` pattern needs TypeScript 5.2+.** It uses the `using` declaration
> (ECMAScript Explicit Resource Management). TypeScript 5.2+ compiles `using`
> down to a `Symbol.dispose` helper, so the **compiled** sample runs on
> Node.js 18+ — no Node.js 24 requirement. The `tsconfig.json` here targets
> `es2022` and includes the `esnext.disposable` lib so `using` type-checks and
> compiles. (In raw JavaScript, `using` instead requires Node.js 24+ — see the
> JavaScript sample.)

## Prerequisites

1. [Node.js 18+](https://nodejs.org/)
2. [TypeScript 5.2+](https://www.typescriptlang.org/) (installed as a dev dependency)
3. [Azure Functions Core Tools v4](https://learn.microsoft.com/azure/azure-functions/functions-run-local) (4.0.5382 or higher)
4. [.NET SDK 8.0+](https://dotnet.microsoft.com/download) (to build/install the extension)
5. [Docker](https://www.docker.com/products/docker-desktop/) (for the Durable Task Scheduler emulator)

## Install the Durable extension manually

The 3.13.0 extension that powers critical sections is **not yet in the Azure
Functions extension bundles**, so this sample does **not** use a bundle. Instead,
it ships an [extensions.csproj](extensions.csproj) that references the required
packages, and you build it once to generate `bin/extensions.json`:

```bash
cd samples/durable-functions/typescript/CriticalSections
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

2. Install the extension (once), install dependencies, build, and start the host:

   ```bash
   dotnet build extensions.csproj -o bin
   npm install
   npm run build
   func start
   ```

3. Start a transfer with one of the release patterns:

   ```bash
   curl -X POST http://localhost:7071/api/transfer/using
   curl -X POST http://localhost:7071/api/transfer/try-finally
   curl -X POST http://localhost:7071/api/transfer/implicit
   curl -X POST http://localhost:7071/api/transfer/early-release
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
and the resulting balances, for example (`using`):

```json
{
  "pattern": "using",
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
- [Durable Functions JavaScript/TypeScript API reference](https://learn.microsoft.com/javascript/api/durable-functions/)
- [Durable Task Scheduler documentation](https://aka.ms/dts-documentation)
