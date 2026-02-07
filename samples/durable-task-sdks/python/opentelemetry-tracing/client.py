"""Client to schedule and monitor orchestrations."""
import os
import asyncio
import durabletask.client as client


async def main():
    endpoint = os.environ.get("ENDPOINT", "http://localhost:8080")
    taskhub = os.environ.get("TASKHUB", "default")

    async with client.DurableTaskClient(
        host_address=endpoint,
        secure_channel=not endpoint.startswith("http://localhost"),
        taskhub=taskhub,
    ) as c:
        print("Scheduling order processing orchestration...")
        instance_id = await c.schedule_new_orchestration(
            "order_processing_orchestration",
            input="Order-12345",
        )
        print(f"Started orchestration: {instance_id}")
        print("Waiting for completion...")

        result = await c.wait_for_orchestration_completion(
            instance_id, timeout=60
        )
        print(f"Status: {result.runtime_status.name}")
        print(f"Result: {result.serialized_output}")
        print()
        print("View traces in Jaeger UI: http://localhost:16686")
        print("View orchestration in DTS Dashboard: http://localhost:8082")


if __name__ == "__main__":
    asyncio.run(main())
