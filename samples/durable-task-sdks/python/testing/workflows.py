"""
Orchestrators and activities for a simple order-processing workflow.

This module defines pure workflow logic with no infrastructure dependencies,
making it easy to test with the in-memory backend.
"""

from durabletask import task


# ---------------------------------------------------------------------------
# Activities
# ---------------------------------------------------------------------------

def validate_order(ctx: task.ActivityContext, order) -> None:
    """Validate that the order has items and a customer name.

    Raises ``ValueError`` on invalid input.
    """
    if not order.customer:
        raise ValueError("Order must have a customer name")
    if not order.items:
        raise ValueError("Order must contain at least one item")
    for item in order.items:
        if item["quantity"] <= 0:
            raise ValueError(f"Invalid quantity for '{item['name']}': {item['quantity']}")


def charge_payment(ctx: task.ActivityContext, amount: float) -> str:
    """Process a payment and return a confirmation ID.

    Raises ``ValueError`` if the amount is not positive.
    """
    if amount <= 0:
        raise ValueError("Payment amount must be positive")
    # In a real app this would call a payment gateway
    return f"PAY-{int(amount * 100)}"


def ship_order(ctx: task.ActivityContext, data: dict) -> str:
    """Ship an order and return a tracking ID."""
    customer = data["customer"]
    item_count = data["item_count"]
    # In a real app this would call a shipping service
    return f"TRACK-{customer.upper()}-{item_count}"


# ---------------------------------------------------------------------------
# Orchestrator
# ---------------------------------------------------------------------------

def order_processing_orchestrator(ctx: task.OrchestrationContext, order):
    """Process an order: validate, calculate total, charge, and ship.

    Demonstrates a sequential activity chain that is easy to unit test
    with the in-memory backend.
    """
    # 1. Validate the order
    yield ctx.call_activity(validate_order, input=order)

    # 2. Calculate total
    total = sum(item["quantity"] * item["unit_price"] for item in order.items)

    # 3. Charge payment
    payment_id = yield ctx.call_activity(charge_payment, input=total)

    # 4. Ship the order
    tracking_id = yield ctx.call_activity(ship_order, input={
        "customer": order.customer,
        "item_count": len(order.items),
    })

    return {
        "payment_id": payment_id,
        "tracking_id": tracking_id,
        "total": total,
        "status": "completed",
    }
