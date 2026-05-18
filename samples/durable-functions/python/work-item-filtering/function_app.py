# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License.

"""
Work Item Filtering sample for Durable Functions (Python).

With workItemFilteringEnabled: true in host.json, this app advertises its
registered orchestrations, activities, and entities to the Durable Task
Scheduler (DTS). DTS then only dispatches matching work items to this worker.

Orchestrations scheduled for functions NOT registered in this app will stay
in Pending state until a matching worker connects — proving filter isolation.
"""

import logging
import json
import azure.functions as func
import azure.durable_functions as df

app = func.FunctionApp(http_auth_level=func.AuthLevel.ANONYMOUS)
bp = df.Blueprint()


# =============================================================================
# Orchestrations
# =============================================================================

@bp.orchestration_trigger(context_name="context")
def greeting_orchestration(context: df.DurableOrchestrationContext):
    """Simple orchestration that calls an activity."""
    result = yield context.call_activity("say_hello", "World")
    return result


@bp.orchestration_trigger(context_name="context")
def fan_out_orchestration(context: df.DurableOrchestrationContext):
    """Fan-out/fan-in: calls the same activity in parallel with different inputs."""
    cities = ["Tokyo", "London", "Seattle"]
    parallel_tasks = [context.call_activity("say_hello", city) for city in cities]
    results = yield context.task_all(parallel_tasks)
    return results


@bp.orchestration_trigger(context_name="context")
def parent_orchestration(context: df.DurableOrchestrationContext):
    """Parent orchestration that calls a child orchestration."""
    result = yield context.call_sub_orchestrator("greeting_orchestration")
    return f"Parent received: {result}"


@bp.orchestration_trigger(context_name="context")
def counter_orchestration(context: df.DurableOrchestrationContext):
    """Orchestration that interacts with a durable entity."""
    entity_id = df.EntityId("counter_entity", "sample-counter")

    yield context.call_entity(entity_id, "add", 10)
    yield context.call_entity(entity_id, "add", 20)
    value = yield context.call_entity(entity_id, "get")

    return value


# =============================================================================
# Activities
# =============================================================================

@bp.activity_trigger(input_name="name")
def say_hello(name: str) -> str:
    """Simple activity that returns a greeting."""
    logging.info(f"say_hello called with: {name}")
    return f"Hello, {name}!"


# =============================================================================
# Entities
# =============================================================================

@bp.entity_trigger(context_name="context")
def counter_entity(context: df.DurableEntityContext):
    """Simple counter entity with add, reset, and get operations."""
    state = context.get_state(lambda: 0)
    operation = context.operation_name

    if operation == "add":
        amount = context.get_input()
        state += amount
    elif operation == "reset":
        state = 0
    elif operation == "get":
        context.set_result(state)

    context.set_state(state)


# =============================================================================
# HTTP triggers
# =============================================================================

@bp.durable_client_input(client_name="client")
@app.route(route="orchestrators/greeting", methods=["POST"])
async def start_greeting(req: func.HttpRequest, client) -> func.HttpResponse:
    """Start the greeting orchestration."""
    client = df.DurableOrchestrationClient(client)
    instance_id = await client.start_new("greeting_orchestration")
    return client.create_check_status_response(req, instance_id)


@bp.durable_client_input(client_name="client")
@app.route(route="orchestrators/fanout", methods=["POST"])
async def start_fan_out(req: func.HttpRequest, client) -> func.HttpResponse:
    """Start the fan-out orchestration."""
    client = df.DurableOrchestrationClient(client)
    instance_id = await client.start_new("fan_out_orchestration")
    return client.create_check_status_response(req, instance_id)


@bp.durable_client_input(client_name="client")
@app.route(route="orchestrators/parent", methods=["POST"])
async def start_parent(req: func.HttpRequest, client) -> func.HttpResponse:
    """Start the parent orchestration."""
    client = df.DurableOrchestrationClient(client)
    instance_id = await client.start_new("parent_orchestration")
    return client.create_check_status_response(req, instance_id)


@bp.durable_client_input(client_name="client")
@app.route(route="orchestrators/counter", methods=["POST"])
async def start_counter(req: func.HttpRequest, client) -> func.HttpResponse:
    """Start the counter orchestration (entity interaction)."""
    client = df.DurableOrchestrationClient(client)
    instance_id = await client.start_new("counter_orchestration")
    return client.create_check_status_response(req, instance_id)


@bp.durable_client_input(client_name="client")
@app.route(route="start/{name}", methods=["POST"])
async def start_any(req: func.HttpRequest, client) -> func.HttpResponse:
    """Generic starter — can schedule any orchestration by name.
    Useful for testing filter isolation: schedule an orchestration this app
    does NOT have and observe it stays Pending.
    """
    name = req.route_params.get("name")
    client = df.DurableOrchestrationClient(client)
    instance_id = await client.start_new(name)
    return client.create_check_status_response(req, instance_id)


app.register_functions(bp)
