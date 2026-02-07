"""Worker with OpenTelemetry tracing for Durable Task SDK."""
import os
import time
import logging

from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.sdk.resources import Resource
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter

import durabletask.worker as worker
import durabletask.task as task

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Configure OpenTelemetry
resource = Resource.create({"service.name": "durable-worker"})
provider = TracerProvider(resource=resource)
otlp_endpoint = os.environ.get("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317")
exporter = OTLPSpanExporter(endpoint=otlp_endpoint, insecure=True)
provider.add_span_processor(BatchSpanProcessor(exporter))
trace.set_tracer_provider(provider)

tracer = trace.get_tracer(__name__)


def validate_order(ctx: task.ActivityContext, order_id: str) -> str:
    with tracer.start_as_current_span("validate_order"):
        logger.info(f"Validating order: {order_id}")
        time.sleep(0.1)
        return f"Validated({order_id})"


def process_payment(ctx: task.ActivityContext, input: str) -> str:
    with tracer.start_as_current_span("process_payment"):
        logger.info(f"Processing payment for: {input}")
        time.sleep(0.2)
        return f"Paid({input})"


def ship_order(ctx: task.ActivityContext, input: str) -> str:
    with tracer.start_as_current_span("ship_order"):
        logger.info(f"Shipping: {input}")
        time.sleep(0.15)
        return f"Shipped({input})"


def send_notification(ctx: task.ActivityContext, input: str) -> str:
    with tracer.start_as_current_span("send_notification"):
        logger.info(f"Notifying customer: {input}")
        time.sleep(0.05)
        return f"Notified({input})"


def order_processing_orchestration(ctx: task.OrchestrationContext, order_id: str):
    """Process an order through validation, payment, shipping, and notification."""
    validated = yield ctx.call_activity(validate_order, input=order_id)
    paid = yield ctx.call_activity(process_payment, input=validated)
    shipped = yield ctx.call_activity(ship_order, input=paid)
    result = yield ctx.call_activity(send_notification, input=shipped)
    return result


if __name__ == "__main__":
    endpoint = os.environ.get("ENDPOINT", "http://localhost:8080")
    taskhub = os.environ.get("TASKHUB", "default")

    with worker.DurableTaskWorker(
        host_address=endpoint,
        secure_channel=not endpoint.startswith("http://localhost"),
        taskhub=taskhub,
    ) as w:
        w.add_orchestrator(order_processing_orchestration)
        w.add_activity(validate_order)
        w.add_activity(process_payment)
        w.add_activity(ship_order)
        w.add_activity(send_notification)

        logger.info("Worker started with OpenTelemetry tracing. Press Ctrl+C to exit.")
        w.start()
        try:
            import signal
            signal.pause()
        except (KeyboardInterrupt, AttributeError):
            import threading
            threading.Event().wait()
