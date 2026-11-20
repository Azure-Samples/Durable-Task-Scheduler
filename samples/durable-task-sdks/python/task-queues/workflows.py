from durabletask import task


def record_order(_ctx: task.ActivityContext, order_id: str) -> str:
    return f"Recorded {order_id}"


def call_partner(_ctx: task.ActivityContext, order_id: str) -> str:
    return f"Partner accepted {order_id}"


def validate_order(ctx: task.OrchestrationContext, order_id: str):
    yield ctx.call_activity(record_order, input=order_id)
    return f"Validated {order_id}"


def process_order(ctx: task.OrchestrationContext, order_id: str):
    recorded = yield ctx.call_activity(record_order, input=order_id)
    partner = yield ctx.call_activity(
        call_partner, input=order_id, tags={"DTS_TaskQueue": "partner-api"}
    )
    # Python has no explicit sub-orchestration tags; this child inherits orders.
    validated = yield ctx.call_sub_orchestrator(validate_order, input=order_id)
    return f"{recorded}; {partner}; {validated}"
