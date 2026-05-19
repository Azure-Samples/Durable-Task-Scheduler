# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License.

"""
Work Item Filtering — App B (Python).

App B registers an entirely DIFFERENT set of functions from App A. Both apps
share the same DTS task hub ("default"). Work item filtering ensures each app
only receives work items for the functions IT has registered.

App A owns:  greeting_orchestration, fan_out_orchestration,
             parent_orchestration, counter_orchestration,
             say_hello activity, counter_entity
App B owns:  orders_orchestration, ship_order activity

Either app's HTTP client can SCHEDULE any orchestration name. The scheduler
routes the work item to the app whose filter matches.
"""

import logging
import azure.functions as func
import azure.durable_functions as df

app = func.FunctionApp(http_auth_level=func.AuthLevel.ANONYMOUS)
bp = df.Blueprint()


@bp.orchestration_trigger(context_name="context")
def orders_orchestration(context: df.DurableOrchestrationContext):
    order_id = context.get_input() or f"order-{context.new_guid()}"
    result = yield context.call_activity("ship_order", order_id)
    return result


@bp.activity_trigger(input_name="orderId")
def ship_order(orderId: str) -> str:
    logging.info("App B shipping %s", orderId)
    return f"Shipped {orderId} from App B"


@app.route(route="orchestrators/orders", methods=["POST"])
@app.durable_client_input(client_name="client")
async def start_orders(req: func.HttpRequest, client) -> func.HttpResponse:
    instance_id = await client.start_new("orders_orchestration", client_input="order-42")
    return client.create_check_status_response(req, instance_id)


@app.route(route="start/{name}", methods=["POST"])
@app.durable_client_input(client_name="client")
async def start_any(req: func.HttpRequest, client) -> func.HttpResponse:
    """Generic starter: schedule ANY orchestration by name from App B."""
    name = req.route_params.get("name")
    instance_id = await client.start_new(name)
    return client.create_check_status_response(req, instance_id)


app.register_functions(bp)
