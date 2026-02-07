import azure.functions as func
import azure.durable_functions as df
import logging
import json

app = func.FunctionApp(http_auth_level=func.AuthLevel.ANONYMOUS)
bp = df.Blueprint()


@bp.orchestration_trigger(context_name="context")
def fan_out_fan_in_orchestration(context: df.DurableOrchestrationContext):
    """Fan-out/Fan-in orchestration that processes items in parallel."""
    # Generate work items
    work_items = [f"item-{i}" for i in range(5)]

    # Fan-out: schedule all activities in parallel
    parallel_tasks = []
    for item in work_items:
        task = context.call_activity("process_item", item)
        parallel_tasks.append(task)

    # Fan-in: wait for all to complete
    results = yield context.task_all(parallel_tasks)

    # Aggregate results
    total = sum(results)
    return {"items_processed": len(results), "total_score": total, "results": results}


@bp.activity_trigger(input_name="item")
def process_item(item: str) -> int:
    """Process a single work item and return a score."""
    logging.info(f"Processing: {item}")
    # Simulate processing - return length of item name as "score"
    score = len(item) * 10
    logging.info(f"Processed {item} with score: {score}")
    return score


@bp.durable_client_input(client_name="client")
@app.route(route="StartFanOutFanIn", methods=["POST"])
async def start_fan_out_fan_in(req: func.HttpRequest, client) -> func.HttpResponse:
    """HTTP trigger to start the fan-out/fan-in orchestration."""
    client = df.DurableOrchestrationClient(client)
    instance_id = await client.start_new("fan_out_fan_in_orchestration")
    logging.info(f"Started orchestration: {instance_id}")
    return client.create_check_status_response(req, instance_id)


app.register_functions(bp)
