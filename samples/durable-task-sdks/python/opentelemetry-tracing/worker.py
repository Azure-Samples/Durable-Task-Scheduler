"""Worker with OpenTelemetry tracing for Durable Task SDK."""
import asyncio
import os
import time
import logging

from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.sdk.resources import Resource
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter

from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Configure OpenTelemetry with a service name that identifies this application
resource = Resource.create({"service.name": "DistributedTracingSample"})
provider = TracerProvider(resource=resource)
otlp_endpoint = os.environ.get("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317")
exporter = OTLPSpanExporter(endpoint=otlp_endpoint, insecure=True)
provider.add_span_processor(BatchSpanProcessor(exporter))
trace.set_tracer_provider(provider)

tracer = trace.get_tracer("Microsoft.DurableTask")


def _activity_span(activity_name: str, task_id: int = 0):
    """Create an activity span with durable task metadata tags."""
    span = tracer.start_span(
        f"activity:{activity_name}",
        attributes={
            "durabletask.task.name": activity_name,
            "durabletask.type": "activity",
            "durabletask.task.task_id": task_id,
        },
    )
    return span


def validate_order(ctx, order_id: str) -> str:
    span = _activity_span("ValidateOrder", task_id=1)
    with trace.use_span(span, end_on_exit=True):
        logger.info(f"Validating order: {order_id}")
        time.sleep(0.1)
        return f"Validated({order_id})"


def process_payment(ctx, input: str) -> str:
    span = _activity_span("ProcessPayment", task_id=2)
    with trace.use_span(span, end_on_exit=True):
        logger.info(f"Processing payment for: {input}")
        time.sleep(0.2)
        return f"Paid({input})"


def ship_order(ctx, input: str) -> str:
    span = _activity_span("ShipOrder", task_id=3)
    with trace.use_span(span, end_on_exit=True):
        logger.info(f"Shipping: {input}")
        time.sleep(0.15)
        return f"Shipped({input})"


def send_notification(ctx, input: str) -> str:
    span = _activity_span("SendNotification", task_id=4)
    with trace.use_span(span, end_on_exit=True):
        logger.info(f"Notifying customer: {input}")
        time.sleep(0.05)
        return f"Notified({input})"


def order_processing_orchestration(ctx, order_id: str):
    """Process an order through validation, payment, shipping, and notification."""
    validated = yield ctx.call_activity(validate_order, input=order_id)
    paid = yield ctx.call_activity(process_payment, input=validated)
    shipped = yield ctx.call_activity(ship_order, input=paid)
    result = yield ctx.call_activity(send_notification, input=shipped)
    return result


async def main():
    endpoint = os.environ.get("ENDPOINT", "http://localhost:8080")
    taskhub = os.environ.get("TASKHUB", "default")

    with DurableTaskSchedulerWorker(
        host_address=endpoint,
        secure_channel=endpoint != "http://localhost:8080",
        taskhub=taskhub,
        token_credential=None,
    ) as w:
        w.add_orchestrator(order_processing_orchestration)
        w.add_activity(validate_order)
        w.add_activity(process_payment)
        w.add_activity(ship_order)
        w.add_activity(send_notification)

        logger.info("Worker started with OpenTelemetry tracing. Press Ctrl+C to exit.")
        w.start()
        try:
            while True:
                await asyncio.sleep(1)
        except KeyboardInterrupt:
            logger.info("Worker shutdown initiated")


if __name__ == "__main__":
    asyncio.run(main())
