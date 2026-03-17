# Scheduled Agent (Copilot SDK)

This sample demonstrates a **timer-triggered Copilot SDK agent** implemented as a Durable Task eternal orchestration.

This recipe currently ships only a `copilot-sdk/` implementation, so this README is the runnable guide for the full sample.

## Eternal orchestration pattern

The orchestration runs one agent invocation, stores the result, waits on a durable timer, and then calls `continue_as_new()` to start the next cycle with a clean history.

This pattern is useful for recurring AI jobs such as:

- daily code review summaries
- hourly monitoring reports
- repeated TODO or dependency audits
- periodic repository health checks

## Files

- `activities.py` - runs the scheduled Copilot SDK prompt and stores the resulting report
- `orchestrations.py` - coordinates one run, one durable timer, and `continue_as_new()`
- `worker.py` - hosts the Durable Task worker
- `client.py` - starts the eternal orchestration with configurable interval and prompt

## Run locally

Start the Durable Task Scheduler emulator first:

```bash
docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 \
  mcr.microsoft.com/dts/dts-emulator:latest
```

Then install dependencies:

```bash
cd ai-recipes/09-scheduled-agent/copilot-sdk
pip install -r requirements.txt
```

> [!NOTE]
> You also need GitHub Copilot SDK authentication configured in your environment before running this sample.

Start the worker:

```bash
python worker.py
```

In another terminal, start a scheduled run:

```bash
python client.py --interval 60 --prompt "List any TODO comments in Python files" --max-runs 3
```

## How scheduling works

1. The orchestration runs `run_scheduled_review`.
2. The orchestration stores the report with `store_report`.
3. A durable timer waits for the configured interval.
4. `continue_as_new()` resets orchestration history and carries forward the updated run count.

Because the timer is durable, the next scheduled run still happens correctly after worker restarts. Inspect the execution history in the dashboard at `http://localhost:8082`.
