"""Client to start travel booking saga orchestrations."""

import asyncio
import json
import logging
import os

from durabletask.azuremanaged.client import DurableTaskSchedulerClient

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
logger = logging.getLogger(__name__)


async def main():
    endpoint = os.getenv("ENDPOINT", "http://localhost:8080")
    taskhub = os.getenv("TASKHUB", "default")

    print(f"Using taskhub: {taskhub}")
    print(f"Using endpoint: {endpoint}")

    c = DurableTaskSchedulerClient(
        host_address=endpoint,
        secure_channel=endpoint != "http://localhost:8080",
        taskhub=taskhub,
        token_credential=None,
    )

    # Scenario 1: Successful booking
    print("\n=== Scenario 1: Successful Booking ===")
    instance_id = c.schedule_new_orchestration(
        "travel_booking_saga",
        input={"destination": "Paris", "nights": 5, "simulate_car_failure": False},
    )
    print(f"Started orchestration: {instance_id}")
    result = c.wait_for_orchestration_completion(instance_id, timeout=30)
    print(f"Result: {json.dumps(json.loads(result.serialized_output), indent=2)}")

    # Scenario 2: Car booking fails — triggers compensation
    print("\n=== Scenario 2: Car Failure + Compensation ===")
    instance_id = c.schedule_new_orchestration(
        "travel_booking_saga",
        input={"destination": "Tokyo", "nights": 3, "simulate_car_failure": True},
    )
    print(f"Started orchestration: {instance_id}")
    result = c.wait_for_orchestration_completion(instance_id, timeout=30)
    print(f"Result: {json.dumps(json.loads(result.serialized_output), indent=2)}")


if __name__ == "__main__":
    asyncio.run(main())
