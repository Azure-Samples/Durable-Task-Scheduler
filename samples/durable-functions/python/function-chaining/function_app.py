import logging
import json
import azure.functions as func
import azure.durable_functions as df

# Create the Durable Functions app with HTTP auth level set to ANONYMOUS for easier testing
app = df.DFApp(http_auth_level=func.AuthLevel.ANONYMOUS)

@app.route(route="function_chaining", methods=["POST"])
@app.durable_client_input(client_name="client")
async def http_start_function_chaining(req: func.HttpRequest, client):
    """HTTP trigger that starts the function chaining orchestration."""
    try:
        # Get input from request body or use default
        req_body = req.get_json()
        name = req_body.get("name", "World") if req_body else "World"
        
        logging.info(f"Starting function chaining orchestration for: {name}")
        
        # Start the orchestration
        instance_id = await client.start_new("function_chaining_orchestrator", client_input=name)
        
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
def function_chaining_orchestrator(context: df.DurableOrchestrationContext):
    """
    Orchestrator that demonstrates the function chaining pattern.
    Calls activities sequentially, passing the output of each activity to the next.
    """
    name = context.get_input()
    logging.info(f"Function chaining orchestration started for: {name}")
    
    # Call first activity - passing input directly
    greeting = yield context.call_activity("say_hello", name)
    
    # Call second activity with the result from first activity
    processed_greeting = yield context.call_activity("process_greeting", greeting)
    
    # Call third activity with the result from second activity
    final_response = yield context.call_activity("finalize_response", processed_greeting)
    
    logging.info(f"Function chaining orchestration completed: {final_response}")
    return final_response

@app.activity_trigger(input_name="name")
def say_hello(name: str) -> str:
    """First activity that greets the user."""
    logging.info(f"Activity say_hello called with name: {name}")
    return f"Hello {name}!"

@app.activity_trigger(input_name="greeting")
def process_greeting(greeting: str) -> str:
    """Second activity that processes the greeting."""
    logging.info(f"Activity process_greeting called with greeting: {greeting}")
    return f"{greeting} How are you today?"

@app.activity_trigger(input_name="response")
def finalize_response(response: str) -> str:
    """Third activity that finalizes the response."""
    logging.info(f"Activity finalize_response called with response: {response}")
    return f"{response} I hope you're doing well!"

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