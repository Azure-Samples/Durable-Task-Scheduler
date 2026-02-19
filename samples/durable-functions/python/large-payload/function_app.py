"""
Large Payload Sample - Python Durable Functions with Durable Task Scheduler

Demonstrates how to use the large payload storage feature to handle payloads
that exceed the Durable Task Scheduler's message size limit. When enabled,
payloads larger than the configured threshold are automatically offloaded to
Azure Blob Storage (compressed via gzip), keeping orchestration history lean
while supporting arbitrarily large data.

This sample uses a fan-out/fan-in pattern: the orchestrator fans out to multiple
activity functions, each of which generates a large payload (configurable size).
The orchestrator then aggregates the results.
"""

import json
import logging
import os
import azure.functions as func
import azure.durable_functions as df

app = df.DFApp(http_auth_level=func.AuthLevel.ANONYMOUS)

# Default payload size in KB (override via PAYLOAD_SIZE_KB app setting)
DEFAULT_PAYLOAD_SIZE_KB = 100

# Default number of parallel activities (override via ACTIVITY_COUNT app setting)
DEFAULT_ACTIVITY_COUNT = 5


def generate_large_payload(size_kb: int) -> str:
    """Generate a JSON payload of approximately the specified size in KB."""
    # Each character in the string is roughly 1 byte
    target_bytes = size_kb * 1024
    filler = "x" * max(0, target_bytes - 50)  # Reserve space for JSON envelope
    return json.dumps({"size_kb": size_kb, "data": filler})


# ---------------------------------------------------------------------------
# HTTP Trigger – starts the orchestration
# ---------------------------------------------------------------------------
@app.route(route="startlargepayload", methods=["POST", "GET"])
@app.durable_client_input(client_name="client")
async def start_large_payload(req: func.HttpRequest, client):
    """HTTP trigger that starts the large-payload orchestration."""
    try:
        body = req.get_json()
    except ValueError:
        body = {}

    activity_count = int(
        req.params.get("activity_count")
        or body.get("activity_count")
        or DEFAULT_ACTIVITY_COUNT
    )
    payload_size_kb = int(
        req.params.get("payload_size_kb")
        or body.get("payload_size_kb")
        or DEFAULT_PAYLOAD_SIZE_KB
    )

    config = {"activity_count": activity_count, "payload_size_kb": payload_size_kb}
    instance_id = await client.start_new("large_payload_orchestrator", client_input=config)
    logging.info("Started orchestration with ID = '%s'.", instance_id)
    return client.create_check_status_response(req, instance_id)


# ---------------------------------------------------------------------------
# Orchestrator – fans out to N parallel activities, each producing a large payload
# ---------------------------------------------------------------------------
@app.orchestration_trigger(context_name="context")
def large_payload_orchestrator(context: df.DurableOrchestrationContext):
    """Fan-out/fan-in orchestrator that exercises large payload externalization."""
    # Read config from orchestration input (set by the HTTP trigger)
    # to avoid non-deterministic environment variable access in the orchestrator.
    config = context.get_input() or {}
    activity_count = config.get("activity_count", DEFAULT_ACTIVITY_COUNT)
    payload_size_kb = config.get("payload_size_kb", DEFAULT_PAYLOAD_SIZE_KB)

    # Fan-out: schedule N activities in parallel
    tasks = []
    for i in range(activity_count):
        tasks.append(
            context.call_activity(
                "process_large_data",
                {"task_id": i, "payload_size_kb": payload_size_kb},
            )
        )

    # Fan-in: wait for all activities to complete
    results = yield context.task_all(tasks)

    # Aggregate results
    total_size = sum(r["size_kb"] for r in results)
    summary = {
        "items_processed": len(results),
        "total_size_kb": total_size,
        "individual_sizes": [r["size_kb"] for r in results],
    }
    return summary


# ---------------------------------------------------------------------------
# Activity – generates and returns a large payload
# ---------------------------------------------------------------------------
@app.activity_trigger(input_name="input")
def process_large_data(input: dict) -> dict:
    """Activity that generates a large payload of configurable size."""
    task_id = input["task_id"]
    payload_size_kb = input["payload_size_kb"]

    logging.info("Task %d: generating %d KB payload...", task_id, payload_size_kb)
    payload = generate_large_payload(payload_size_kb)
    actual_size = len(payload.encode("utf-8"))
    logging.info("Task %d: payload size = %d bytes", task_id, actual_size)

    return {"task_id": task_id, "size_kb": payload_size_kb, "payload": payload}


# ---------------------------------------------------------------------------
# Health-check endpoint
# ---------------------------------------------------------------------------
@app.route(route="hello", methods=["GET"])
def hello(req: func.HttpRequest) -> func.HttpResponse:
    """Simple health-check endpoint."""
    return func.HttpResponse("Hello from Large Payload Sample!")
