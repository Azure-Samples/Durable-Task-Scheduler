# Task queues (Python SDK)

`process_order` starts with `DTS_TaskQueue=orders`. When the queues exist, its
`record_order` activity and `validate_order` child inherit `orders`, while
`call_partner` explicitly uses `partner-api`. Python supports start and
activity tags but has no explicit sub-orchestration tags. Activities return
simulated results without external API calls.

## Prerequisites

- Python 3.10 or later.
- An Azure scheduler and a dedicated task hub with `orders` and `partner-api`.
  Complete the [shared task queue setup](../../../../docs/task-queues.md).
- An identity with Durable Task Data Contributor access.

The worker never opts in to work-item filtering and registers every task.
Use only this sample's worker (or identical replicas) in its hub.

## Run

From this directory:

```bash
python -m venv venv
source venv/bin/activate
pip install -r requirements.txt
export ENDPOINT="https://<scheduler-host>"
export TASKHUB="<hub>"
python main.py order-001
```

Azure connections use `DefaultAzureCredential`. Without `ENDPOINT` and
`TASKHUB`, the sample uses the emulator on `localhost:8080` and hub `default`.
Emulator execution exercises the workflow, not Azure queue provisioning or
rate enforcement.

## Expected output

```text
Started <instance-id> with DTS_TaskQueue=orders
Recorded order-001; Partner accepted order-001; Validated order-001
```

The client waits at most 120 seconds and rejects unsuccessful completion or
unexpected output. A timeout does not terminate the persisted instance.
The worker, client, and credential are closed when the process exits.

`main.py` configures connections and the tagged start; `workflows.py` contains
the inherited and overridden activity calls.

## Tests

```bash
python -m unittest test_workflows.py
```

These offline tests check the options passed to the SDK, workflow output,
and failure propagation. They do not test scheduler routing or throttling.
