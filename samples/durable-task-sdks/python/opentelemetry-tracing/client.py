"""Client to schedule and monitor orchestrations with OpenTelemetry tracing.

The SDK automatically captures the current OpenTelemetry span context
and propagates it as W3C trace context to the orchestration, which then
forwards it to all activities and sub-orchestrations.
"""
import os
import asyncio

from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.sdk.resources import Resource
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter

from durabletask.azuremanaged.client import DurableTaskSchedulerClient

# Configure OpenTelemetry (same service name as worker for unified view)
resource = Resource.create({"service.name": "DistributedTracingSample"})
provider = TracerProvider(resource=resource)
otlp_endpoint = os.environ.get("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317")
exporter = OTLPSpanExporter(endpoint=otlp_endpoint, insecure=True)
provider.add_span_processor(BatchSpanProcessor(exporter))
trace.set_tracer_provider(provider)

tracer = trace.get_tracer("durabletask")


async def main():
    endpoint = os.environ.get("ENDPOINT", "http://localhost:8080")
    taskhub = os.environ.get("TASKHUB", "default")

    c = DurableTaskSchedulerClient(
        host_address=endpoint,
        secure_channel=endpoint != "http://localhost:8080",
        taskhub=taskhub,
        token_credential=None,
    )

    # Create a parent span — the SDK automatically captures this context
    # and propagates it to the orchestration and all child activities.
    with tracer.start_as_current_span(
        "create_orchestration:OrderProcessingOrchestration",
        attributes={
            "durabletask.task.name": "OrderProcessingOrchestration",
            "durabletask.type": "orchestration",
        },
    ) as span:
        print("Scheduling order processing orchestration...")
        instance_id = c.schedule_new_orchestration(
            "order_processing_orchestration",
            input="Order-12345",
        )
        span.set_attribute("durabletask.task.instance_id", instance_id)
        print(f"Started orchestration: {instance_id}")
        print("Waiting for completion...")

        result = c.wait_for_orchestration_completion(
            instance_id, timeout=60
        )
        print(f"Status: {result.runtime_status.name}")
        print(f"Result: {result.serialized_output}")

    # Flush remaining spans
    provider.force_flush()

    print()
    print("View traces in Jaeger UI: http://localhost:16686")
    print("  Search for service: DistributedTracingSample")
    print("View orchestration in DTS Dashboard: http://localhost:8082")


if __name__ == "__main__":
    asyncio.run(main())
