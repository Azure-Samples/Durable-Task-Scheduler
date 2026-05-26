import azure.functions as func
import azure.durable_functions as df
import logging

app = df.DFApp(http_auth_level=func.AuthLevel.ANONYMOUS)


@app.orchestration_trigger(context_name="context")
def chaining_orchestration(context: df.DurableOrchestrationContext):
    """Function chaining orchestration: calls activities sequentially."""
    result1 = yield context.call_activity("say_hello", "Tokyo")
    result2 = yield context.call_activity("say_hello", "Seattle")
    result3 = yield context.call_activity("say_hello", "London")
    return [result1, result2, result3]


@app.orchestration_trigger(context_name="context")
def fan_out_fan_in_orchestration(context: df.DurableOrchestrationContext):
    """Fan-out/Fan-in orchestration: calls activities in parallel."""
    cities = ["Tokyo", "Seattle", "London", "Paris", "Berlin"]

    # Fan-out: schedule all activities in parallel
    parallel_tasks = []
    for city in cities:
        task = context.call_activity("say_hello", city)
        parallel_tasks.append(task)

    # Fan-in: wait for all to complete
    results = yield context.task_all(parallel_tasks)
    return results


@app.activity_trigger(input_name="city")
def say_hello(city: str) -> str:
    """Activity function that returns a greeting for a city."""
    logging.info(f"Saying hello to {city}.")
    return f"Hello {city}!"


@app.route(route="StartChaining", methods=["POST"])
@app.durable_client_input(client_name="client")
async def start_chaining(req: func.HttpRequest, client) -> func.HttpResponse:
    """HTTP trigger to start the function chaining orchestration."""
    instance_id = await client.start_new("chaining_orchestration")
    logging.info(f"Started chaining orchestration with ID = '{instance_id}'.")
    return client.create_check_status_response(req, instance_id)


@app.route(route="StartFanOutFanIn", methods=["POST"])
@app.durable_client_input(client_name="client")
async def start_fan_out_fan_in(req: func.HttpRequest, client) -> func.HttpResponse:
    """HTTP trigger to start the fan-out/fan-in orchestration."""
    instance_id = await client.start_new("fan_out_fan_in_orchestration")
    logging.info(f"Started fan-out/fan-in orchestration with ID = '{instance_id}'.")
    return client.create_check_status_response(req, instance_id)
