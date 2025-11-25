import logging
import json
import asyncio
import azure.functions as func
import azure.durable_functions as df

# Create the Durable Functions app with HTTP auth level set to ANONYMOUS for easier testing
app = df.DFApp(http_auth_level=func.AuthLevel.ANONYMOUS)

@app.route(route="fan_out_fan_in", methods=["POST"])
@app.durable_client_input(client_name="client")
async def http_start_fan_out_fan_in(req: func.HttpRequest, client):
    """HTTP trigger that starts the fan-out/fan-in orchestration."""
    try:
        # Get input from request body or use default
        req_body = req.get_json()
        work_items = req_body.get("workItems", ["Item1", "Item2", "Item3", "Item4", "Item5"]) if req_body else ["Item1", "Item2", "Item3", "Item4", "Item5"]
        
        logging.info(f"Starting fan-out/fan-in orchestration with {len(work_items)} work items")
        
        # Start the orchestration
        instance_id = await client.start_new("fan_out_fan_in_orchestrator", client_input=work_items)
        
        # Return management URLs for the orchestration
        return client.create_check_status_response(req, instance_id)
        
    except Exception as e:
        logging.error(f"Error starting orchestration: {str(e)}")
        return func.HttpResponse(
            json.dumps({"error": f"Failed to start orchestration: {str(e)}"}),
            status_code=500,
            mimetype="application/json"
        )

@app.orchestration_trigger(context_name="context")
def fan_out_fan_in_orchestrator(context: df.DurableOrchestrationContext):
    """
    Orchestrator that demonstrates the fan-out/fan-in pattern.
    Processes multiple work items in parallel and aggregates the results.
    """
    work_items = context.get_input()
    logging.info(f"Fan-out/fan-in orchestration started with {len(work_items)} work items")
    
    # Fan-out: Start all work items in parallel
    parallel_tasks = []
    for item in work_items:
        task = context.call_activity("process_work_item", item)
        parallel_tasks.append(task)
    
    # Fan-in: Wait for all tasks to complete and collect results
    results = yield context.task_all(parallel_tasks)
    
    # Aggregate the results
    aggregated_result = yield context.call_activity("aggregate_results", results)
    
    logging.info(f"Fan-out/fan-in orchestration completed with aggregated result")
    return aggregated_result

@app.activity_trigger(input_name="workItem")
def process_work_item(workItem: str) -> dict:
    """Process a single work item."""
    import time
    import random
    
    logging.info(f"Processing work item: {workItem}")
    
    # Simulate processing time
    processing_time = random.uniform(0.5, 2.0)
    time.sleep(processing_time)
    
    # Simulate different types of processing results
    result = {
        "item": workItem,
        "processed": True,
        "processing_time": round(processing_time, 2),
        "result": f"Processed_{workItem}",
        "value": random.randint(10, 100)
    }
    
    logging.info(f"Completed processing work item: {workItem}")
    return result

@app.activity_trigger(input_name="results")
def aggregate_results(results: list) -> dict:
    """Aggregate the results from all parallel processing tasks."""
    logging.info(f"Aggregating {len(results)} results")
    
    total_value = sum(result["value"] for result in results)
    total_processing_time = sum(result["processing_time"] for result in results)
    processed_items = [result["result"] for result in results]
    
    aggregated = {
        "total_items_processed": len(results),
        "total_value": total_value,
        "average_value": round(total_value / len(results), 2),
        "total_processing_time": round(total_processing_time, 2),
        "processed_items": processed_items,
        "success": True
    }
    
    logging.info(f"Aggregation completed: {aggregated}")
    return aggregated

@app.route(route="status/{instanceId}", methods=["GET"])
@app.durable_client_input(client_name="client")
async def get_orchestration_status(req: func.HttpRequest, client):
    """Get the status of a running orchestration."""
    instance_id = req.route_params.get('instanceId')
    
    try:
        status = await client.get_status(instance_id)
        
        if status:
            return func.HttpResponse(
                json.dumps({
                    "instanceId": status.instance_id,
                    "name": status.name,
                    "runtimeStatus": status.runtime_status,
                    "input": status.input_,
                    "output": status.output,
                    "createdTime": status.created_time.isoformat() if status.created_time else None,
                    "lastUpdatedTime": status.last_updated_time.isoformat() if status.last_updated_time else None
                }, default=str),
                mimetype="application/json"
            )
        else:
            return func.HttpResponse(
                json.dumps({"error": "Orchestration not found"}),
                status_code=404,
                mimetype="application/json"
            )
            
    except Exception as e:
        logging.error(f"Error getting status: {str(e)}")
        return func.HttpResponse(
            json.dumps({"error": f"Failed to get status: {str(e)}"}),
            status_code=500,
            mimetype="application/json"
        )