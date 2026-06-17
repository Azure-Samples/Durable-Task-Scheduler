# On-demand Sandboxes: LLM-generated code interpreter demo

A Durable Task Scheduler (DTS) demo of the **On-demand Sandboxes** preview, built
in two languages. A three-step workflow asks a natural-language question over a
CSV: an LLM generates a pandas script, the **untrusted** script runs in a
DTS-managed on-demand sandbox (fanned out per region), and an in-process step
aggregates the answer.

| Directory | Implementation | SDK |
| --- | --- | --- |
| [`dotnet/`](dotnet/README.md) | .NET 10 | [durabletask-dotnet #736](https://github.com/microsoft/durabletask-dotnet/pull/736) |
| [`python/`](python/README.md) | Python 3.12 | [durabletask-python #151](https://github.com/microsoft/durabletask-python/pull/151) |

Both implementations follow the same shape: a main/declarer app hosts the
orchestrator and in-process activities and declares the sandbox worker profile,
while a separate sandbox worker image runs the offloaded `ExecuteCode` /
`execute_code` activity in DTS-provisioned compute.

See each directory's README for prerequisites, build, and run instructions.
