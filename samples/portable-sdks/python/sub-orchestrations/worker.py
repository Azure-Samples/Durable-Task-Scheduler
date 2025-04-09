import os
import random
import time
from azure.identity import DefaultAzureCredential
from durabletask import task
from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

def get_orders(ctx, _) -> list[str]:
    """Activity function that returns a list of work items"""
    # return a random number of work items
    count = random.randint(2, 10)
    print(f'generating {count} orders...')
    return [f'order {i}' for i in range(count)]

def check_and_update_inventory(ctx, order: str) -> str:
    """Activity function that checks inventory for a given order"""
    print(f'checking inventory for order: {order}')

    # simulate inventory check
    time.sleep(random.random() * 2)

    # return a random boolean indicating if the item is in stock
    return random.choices([True, False], weights=[9, 1])

def charge_payment(ctx, order: str) -> bool:
    """Activity function that charges payment for a given order"""
    print(f'charging payment for order: {order}')

    # simulate payment processing
    time.sleep(random.random() * 2)

    # return a random boolean indicating if the payment was successful
    return random.choices([True, False], weights=[9, 1])

def ship_order(ctx, order: str) -> bool:
    """Activity function that ships a given order"""
    print(f'shipping order: {order}')

    # simulate shipping process
    time.sleep(random.random() * 2)

    # return a random boolean indicating if the shipping was successful
    return random.choices([True, False], weights=[9, 1])

def notify_customer(ctx, order: str) -> bool:
    """Activity function that notifies the customer about the order status"""
    print(f'notifying customer about order: {order}')

    # simulate customer notification
    time.sleep(random.random() * 2)

    # return a random boolean indicating if the notification was successful
    return random.choices([True, False], weights=[9, 1])

def process_order(ctx, order: str) -> dict:
    """Sub-orchestration function that processes a given order by performing all steps"""
    print(f'processing order: {order}')

     # Check inventory
    inventory_checked = yield ctx.call_activity('check_and_update_inventory', input=order)

    if not inventory_checked:
        return {'order': order, 'status': 'failed', 'reason': 'out of stock'}

    # Charge payment
    payment_charged = yield ctx.call_activity('charge_payment', input=order)

    if not payment_charged:
        return {'order': order, 'status': 'failed', 'reason': 'payment failed'}

    # Ship order
    order_shipped = yield ctx.call_activity('ship_order', input=order)

    if not order_shipped:
        return {'order': order, 'status': 'failed', 'reason': 'shipping failed'}

    # Notify customer
    customer_notified = yield ctx.call_activity('notify_customer', input=order)

    if not customer_notified:
        return {'order': order, 'status': 'failed', 'reason': 'customer notification failed'}

    # Return success status
    return {'order': order, 'status': 'completed'}

def orchestrator(ctx, _):
    """Orchestrator function that calls the 'get_orders' and 'process_order'
    sub-orchestration functions in parallel, waits for them all to complete, and prints
    an aggregate summary of the outputs"""

    orders: list[str] = yield ctx.call_activity('get_orders')

    # Execute the orders in parallel and wait for them all to return
    tasks = [ctx.call_sub_orchestrator(process_order, input=order) for order in orders]
    results: list[dict] = yield task.when_all(tasks)

    # Return an aggregate summary of the results
    return {
        'orders': orders,
        'results': results,
        'total_completed': sum(1 for result in results if result['status'] == 'completed'),
        'total_failed': sum(1 for result in results if result['status'] == 'failed'),
        'details': results,
    }

# Read the environment variable
taskhub_name = os.getenv("TASKHUB")

# Check if the variable exists
if taskhub_name:
    print(f"The value of TASKHUB is: {taskhub_name}")
else:
    print("TASKHUB is not set. Please set the TASKHUB environment variable to the name of the taskhub you wish to use")
    print("If you are using windows powershell, run the following: $env:TASKHUB=\"<taskhubname>\"")
    print("If you are using bash, run the following: export TASKHUB=\"<taskhubname>\"")
    exit()

# Read the environment variable
endpoint = os.getenv("ENDPOINT")

# Check if the variable exists
if endpoint:
    print(f"The value of ENDPOINT is: {endpoint}")
else:
    print("ENDPOINT is not set. Please set the ENDPOINT environment variable to the endpoint of the scheduler")
    print("If you are using windows powershell, run the following: $env:ENDPOINT=\"<schedulerEndpoint>\"")
    print("If you are using bash, run the following: export ENDPOINT=\"<schedulerEndpoint>\"")
    exit()

credential = DefaultAzureCredential()

# Configure and start the worker
with DurableTaskSchedulerWorker(host_address=endpoint, secure_channel=True,
                                taskhub=taskhub_name, token_credential=credential) as w:
    
    w.add_orchestrator(orchestrator)
    w.add_orchestrator(process_order)
    w.add_activity(get_orders)
    w.add_activity(check_and_update_inventory)
    w.add_activity(charge_payment)
    w.add_activity(ship_order)
    w.add_activity(notify_customer)

    w.start()

    while True:
        time.sleep(1)
