# Orchestration Patterns

This guide maps each common orchestration pattern to available samples and documentation. All patterns can be developed locally using the [Durable Task Scheduler emulator](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/develop-with-durable-task-scheduler).

---

## Function Chaining
Sequential execution of activities where the output of one becomes the input of the next.

```
Activity A → Activity B → Activity C → Result
```

**Use cases:** Data processing pipelines, multi-step approval workflows, document generation

| Language | Durable Task SDK | Durable Functions |
|----------|-----------------|-------------------|
| .NET | [Sample](../samples/durable-task-sdks/dotnet/FunctionChaining) | [HelloCities](../samples/durable-functions/dotnet/HelloCities) |
| Python | [Sample](../samples/durable-task-sdks/python/function-chaining) | — |
| Java | [Sample](../samples/durable-task-sdks/java/function-chaining) | [HelloCities](../samples/durable-functions/java/HelloCities) |
| JavaScript | — | [HelloCities](../samples/durable-functions/javascript/HelloCities) |

📖 [Learn more on Microsoft Learn →](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-sequence)

---

## Fan-out/Fan-in
Execute multiple activities in parallel and aggregate results when all complete.

```
         ┌→ Activity A ─┐
Input ───┼→ Activity B ──┼→ Aggregate → Result
         └→ Activity C ─┘
```

**Use cases:** Batch processing, parallel API calls, map-reduce operations

| Language | Durable Task SDK | Durable Functions |
|----------|-----------------|-------------------|
| .NET | [Sample](../samples/durable-task-sdks/dotnet/FanOutFanIn) | — |
| Python | [Sample](../samples/durable-task-sdks/python/fan-out-fan-in) | [Fan-out/Fan-in](../samples/durable-functions/python/fan-out-fan-in) |
| Java | [Sample](../samples/durable-task-sdks/java/fan-out-fan-in) | [HelloCities](../samples/durable-functions/java/HelloCities) |
| JavaScript | — | [HelloCities](../samples/durable-functions/javascript/HelloCities) |

📖 [Learn more on Microsoft Learn →](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-cloud-backup)

---

## Async HTTP API
Start a long-running operation via HTTP and poll for results.

```
Client → POST /start → 202 Accepted (with status URL)
Client → GET /status  → 200 Running...
Client → GET /status  → 200 Completed (with result)
```

**Use cases:** Long-running API operations, background job processing

| Language | Durable Task SDK | Durable Functions |
|----------|-----------------|-------------------|
| .NET | [ASP.NET Web App](../samples/durable-task-sdks/dotnet/AspNetWebApp) | [HelloCities](../samples/durable-functions/dotnet/HelloCities) |
| Python | [Sample](../samples/durable-task-sdks/python/async-http-api) | — |
| Java | [Sample](../samples/durable-task-sdks/java/async-http-api) | — |

📖 [Learn more on Microsoft Learn →](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-http-api)

---

## Human Interaction
Pause an orchestration to wait for external input (approval, user decision) with timeout support.

```
Orchestration → Wait for approval event
                  ├→ Approved → Continue
                  └→ Timeout  → Escalate/Cancel
```

**Use cases:** Approval workflows, manual review steps, interactive processes

| Language | Durable Task SDK | Durable Functions |
|----------|-----------------|-------------------|
| .NET | [Sample](../samples/durable-task-sdks/dotnet/HumanInteraction) | — |
| Python | [Sample](../samples/durable-task-sdks/python/human-interaction) | — |
| Java | [Sample](../samples/durable-task-sdks/java/human-interaction) | — |

📖 [Learn more on Microsoft Learn →](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-phone-verification)

---

## Monitoring
Periodic polling pattern that checks status at intervals until a condition is met.

```
Check status → Not ready → Wait → Check status → Ready → Done
```

**Use cases:** Health checks, deployment monitoring, SLA enforcement

| Language | Durable Task SDK | Durable Functions |
|----------|-----------------|-------------------|
| .NET | [Sample](../samples/durable-task-sdks/dotnet/Monitoring) | — |
| Python | [Sample](../samples/durable-task-sdks/python/monitoring) | — |
| Java | [Sample](../samples/durable-task-sdks/java/monitoring) | — |

📖 [Learn more on Microsoft Learn →](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-monitor)

---

## Sub-orchestrations
Compose complex workflows by calling child orchestrations from a parent.

```
Parent Orchestration
  ├→ Sub-orchestration A → Result A
  └→ Sub-orchestration B → Result B
```

**Use cases:** Modular workflow design, code reuse, scoped retry logic

| Language | Durable Task SDK | Durable Functions |
|----------|-----------------|-------------------|
| .NET | [Sample](../samples/durable-task-sdks/dotnet/SubOrchestrations) | — |
| Python | [Sample](../samples/durable-task-sdks/python/sub-orchestrations) | — |
| Java | [Sample](../samples/durable-task-sdks/java/sub-orchestrations) | — |

📖 [Learn more on Microsoft Learn →](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-sub-orchestrations)

---

## Eternal Orchestrations
Long-running orchestrations that use `continue_as_new` to prevent unbounded history growth.

```
Process batch → Continue as new → Process batch → Continue as new → ...
```

**Use cases:** Event processors, scheduled jobs, continuous monitoring

| Language | Durable Task SDK | Durable Functions |
|----------|-----------------|-------------------|
| .NET | [Sample](../samples/durable-task-sdks/dotnet/EternalOrchestrations) | — |
| Python | [Sample](../samples/durable-task-sdks/python/eternal-orchestrations) | — |
| Java | [Sample](../samples/durable-task-sdks/java/eternal-orchestrations) | — |

📖 [Learn more on Microsoft Learn →](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-eternal-orchestrations)

---

## Bounded Coordinator
A coordinator that fans out child work in bounded batches, waits for all children, then resets history via `continue_as_new` after each batch.

```
Coordinator (batch N) → fan out children → wait all → continue_as_new → batch N+1
```

**Use cases:** Continuous queue draining, long-running ingestion/ETL pipelines, message-pump coordinators that must avoid history bloat

| Language | Durable Task SDK | Durable Functions |
|----------|-----------------|-------------------|
| .NET | [Sample](../samples/durable-task-sdks/dotnet/BoundedCoordinator) | — |
| Python | [Sample](../samples/durable-task-sdks/python/bounded-coordinator) | — |

📖 [Learn more on Microsoft Learn →](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-eternal-orchestrations)

---

## Durable Entities
Stateful objects that persist their state and support operations via messages.

```
Entity: Counter
  ├→ Add(5)  → State: 5
  ├→ Add(3)  → State: 8
  └→ Get()   → Returns: 8
```

**Use cases:** Shopping carts, user sessions, aggregations, IoT device state

| Language | Durable Task SDK | Durable Functions |
|----------|-----------------|-------------------|
| .NET | [Sample](../samples/durable-task-sdks/dotnet/EntitiesSample) | — |
| Python | [Sample](../samples/durable-task-sdks/python/entities) | — |

📖 [Learn more on Microsoft Learn →](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-entities)

---

## Saga / Compensation
Execute a sequence of operations with compensating actions if any step fails.

```
Step 1 → Step 2 → Step 3 (fails!)
                    └→ Compensate Step 2 → Compensate Step 1
```

**Use cases:** Distributed transactions, order processing, booking systems

| Language | Durable Task SDK | Durable Functions |
|----------|-----------------|-------------------|
| .NET | — | [Saga Sample](../samples/durable-functions/dotnet/Saga) |
| Python | [Sample](../samples/durable-task-sdks/python/saga) | — |

📖 [Learn more on Microsoft Learn →](https://learn.microsoft.com/azure/architecture/reference-architectures/saga/saga)

---

## Orchestration Versioning
Safely evolve orchestration logic without breaking in-flight instances.

| Language | Durable Task SDK | Durable Functions |
|----------|-----------------|-------------------|
| .NET | [Sample](../samples/durable-task-sdks/dotnet/OrchestrationVersioning) | — |
| Python | [Sample](../samples/durable-task-sdks/python/versioning) | — |

📖 [Learn more on Microsoft Learn →](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-versioning)

---

## Scheduled Tasks
Run a target orchestration on a recurring interval, with pause/resume and runtime management.

```
Schedule (every 5s) → Start orchestration → ... → Start orchestration
```

**Use cases:** Periodic background jobs, cache clearing, report generation, cron-like recurring workflows

| Language | Durable Task SDK | Durable Functions |
|----------|-----------------|-------------------|
| .NET | [Schedule Web App](../samples/durable-task-sdks/dotnet/ScheduleWebApp) | — |
| Python | [Sample](../samples/durable-task-sdks/python/scheduled-tasks) | — |

---

## Next Steps

- [Full Sample Catalog →](../samples/README.md)
- [Durable Functions Documentation →](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-overview)
- [Durable Task Scheduler Documentation →](https://aka.ms/dts-documentation)
