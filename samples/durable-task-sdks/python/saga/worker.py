"""Saga pattern worker — Travel booking with compensating transactions."""

import asyncio
import logging
import os
from datetime import datetime

from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
logger = logging.getLogger(__name__)


# --- Activities: Booking ---

def book_flight(ctx, input: dict) -> dict:
    """Book a flight. Simulates success or failure based on destination."""
    destination = input["destination"]
    logger.info(f"Booking flight to {destination}...")

    # Simulate: flights to "Nowhere" fail
    if destination.lower() == "nowhere":
        raise Exception(f"No flights available to {destination}")

    confirmation = f"FL-{datetime.now().strftime('%Y%m%d%H%M%S')}"
    logger.info(f"Flight booked: {confirmation}")
    return {"confirmation": confirmation, "service": "flight", "destination": destination}


def book_hotel(ctx, input: dict) -> dict:
    """Book a hotel. Simulates success or failure based on dates."""
    destination = input["destination"]
    nights = input.get("nights", 3)
    logger.info(f"Booking hotel in {destination} for {nights} nights...")

    # Simulate: 0 nights fails
    if nights <= 0:
        raise Exception("Invalid hotel booking: 0 nights")

    confirmation = f"HT-{datetime.now().strftime('%Y%m%d%H%M%S')}"
    logger.info(f"Hotel booked: {confirmation}")
    return {"confirmation": confirmation, "service": "hotel", "destination": destination}


def book_car(ctx, input: dict) -> dict:
    """Book a rental car. Simulates failure when simulate_failure is True."""
    destination = input["destination"]
    logger.info(f"Booking rental car in {destination}...")

    if input.get("simulate_car_failure", False):
        raise Exception(f"No rental cars available in {destination}")

    confirmation = f"CR-{datetime.now().strftime('%Y%m%d%H%M%S')}"
    logger.info(f"Car booked: {confirmation}")
    return {"confirmation": confirmation, "service": "car", "destination": destination}


# --- Activities: Compensation ---

def cancel_flight(ctx, input: dict) -> str:
    """Compensating action: cancel a flight booking."""
    confirmation = input["confirmation"]
    logger.info(f"COMPENSATING: Cancelling flight {confirmation}")
    return f"Flight {confirmation} cancelled"


def cancel_hotel(ctx, input: dict) -> str:
    """Compensating action: cancel a hotel booking."""
    confirmation = input["confirmation"]
    logger.info(f"COMPENSATING: Cancelling hotel {confirmation}")
    return f"Hotel {confirmation} cancelled"


def cancel_car(ctx, input: dict) -> str:
    """Compensating action: cancel a car booking."""
    confirmation = input["confirmation"]
    logger.info(f"COMPENSATING: Cancelling car {confirmation}")
    return f"Car {confirmation} cancelled"


# --- Orchestration ---

def travel_booking_saga(ctx, input: dict):
    """
    Saga orchestration: book flight -> hotel -> car.
    If any step fails, compensate all previous steps in reverse order.
    """
    destination = input["destination"]
    nights = input.get("nights", 3)
    simulate_car_failure = input.get("simulate_car_failure", False)

    completed_bookings = []  # Stack of (booking_result, cancel_activity_name)

    try:
        # Step 1: Book flight
        flight = yield ctx.call_activity(
            book_flight, input={"destination": destination})
        completed_bookings.append((flight, cancel_flight))

        # Step 2: Book hotel
        hotel = yield ctx.call_activity(
            book_hotel, input={"destination": destination, "nights": nights})
        completed_bookings.append((hotel, cancel_hotel))

        # Step 3: Book car
        car = yield ctx.call_activity(
            book_car, input={"destination": destination, "simulate_car_failure": simulate_car_failure})
        completed_bookings.append((car, cancel_car))

        # All succeeded!
        return {
            "status": "success",
            "bookings": {
                "flight": flight["confirmation"],
                "hotel": hotel["confirmation"],
                "car": car["confirmation"],
            },
            "destination": destination,
        }

    except Exception as e:
        # Compensation: undo completed bookings in reverse order
        logger.info(f"Booking failed: {e}. Starting compensation...")
        compensations = []

        for booking, cancel_activity in reversed(completed_bookings):
            try:
                result = yield ctx.call_activity(cancel_activity, input=booking)
                compensations.append(result)
            except Exception as comp_error:
                compensations.append(f"Compensation failed: {comp_error}")

        return {
            "status": "failed",
            "error": str(e),
            "compensations": compensations,
            "destination": destination,
        }


async def main():
    endpoint = os.getenv("ENDPOINT", "http://localhost:8080")
    taskhub = os.getenv("TASKHUB", "default")

    logger.info("Starting Saga pattern worker...")
    print(f"Using taskhub: {taskhub}")
    print(f"Using endpoint: {endpoint}")

    with DurableTaskSchedulerWorker(
        host_address=endpoint,
        secure_channel=endpoint != "http://localhost:8080",
        taskhub=taskhub,
        token_credential=None,
    ) as w:
        w.add_orchestrator(travel_booking_saga)
        w.add_activity(book_flight)
        w.add_activity(book_hotel)
        w.add_activity(book_car)
        w.add_activity(cancel_flight)
        w.add_activity(cancel_hotel)
        w.add_activity(cancel_car)

        w.start()
        logger.info("Saga worker started. Press Ctrl+C to exit.")

        try:
            while True:
                await asyncio.sleep(1)
        except KeyboardInterrupt:
            logger.info("Worker shutdown initiated")

    logger.info("Worker stopped")


if __name__ == "__main__":
    asyncio.run(main())
