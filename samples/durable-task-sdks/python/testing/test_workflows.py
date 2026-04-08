"""
Unit tests for the order-processing workflows.

These tests use the in-memory backend so they run entirely in-process
with no external dependencies — no sidecar, no emulator, no Azure.

Run with:
    pytest test_workflows.py
"""

import json

import pytest

from durabletask import client, worker
from durabletask.testing import create_test_backend

from workflows import (
    charge_payment,
    order_processing_orchestrator,
    ship_order,
    validate_order,
)

HOST = "localhost:50061"


@pytest.fixture(autouse=True)
def backend():
    """Start and stop the in-memory backend for each test."""
    b = create_test_backend(port=50061)
    yield b
    b.stop()
    b.reset()


def _create_worker() -> worker.TaskHubGrpcWorker:
    """Create a worker with all orchestrators and activities registered."""
    w = worker.TaskHubGrpcWorker(host_address=HOST)
    w.add_orchestrator(order_processing_orchestrator)
    w.add_activity(validate_order)
    w.add_activity(charge_payment)
    w.add_activity(ship_order)
    return w


# ---------------------------------------------------------------------------
# Happy path tests
# ---------------------------------------------------------------------------

class TestOrderProcessing:
    """Tests for the order_processing_orchestrator."""

    def test_single_item_order(self):
        """A single-item order should complete with the correct total."""
        order = {
            "customer": "Alice",
            "items": [{"name": "Widget", "quantity": 2, "unit_price": 10.00}],
        }

        with _create_worker() as w:
            w.start()
            c = client.TaskHubGrpcClient(host_address=HOST)
            instance_id = c.schedule_new_orchestration(
                order_processing_orchestrator, input=order
            )
            state = c.wait_for_orchestration_completion(instance_id, timeout=30)

        assert state is not None
        assert state.runtime_status == client.OrchestrationStatus.COMPLETED
        assert state.serialized_output is not None

        result = json.loads(state.serialized_output)
        assert result["total"] == 20.0
        assert result["status"] == "completed"
        assert result["payment_id"] == "PAY-2000"
        assert result["tracking_id"] == "TRACK-ALICE-1"

    def test_multi_item_order(self):
        """An order with multiple items should calculate the correct total."""
        order = {
            "customer": "Bob",
            "items": [
                {"name": "Widget", "quantity": 3, "unit_price": 25.00},
                {"name": "Gadget", "quantity": 1, "unit_price": 99.99},
            ],
        }

        with _create_worker() as w:
            w.start()
            c = client.TaskHubGrpcClient(host_address=HOST)
            instance_id = c.schedule_new_orchestration(
                order_processing_orchestrator, input=order
            )
            state = c.wait_for_orchestration_completion(instance_id, timeout=30)

        assert state is not None
        assert state.runtime_status == client.OrchestrationStatus.COMPLETED

        result = json.loads(state.serialized_output)
        expected_total = 3 * 25.00 + 1 * 99.99  # 174.99
        assert result["total"] == expected_total
        assert result["tracking_id"] == "TRACK-BOB-2"


# ---------------------------------------------------------------------------
# Failure path tests
# ---------------------------------------------------------------------------

class TestOrderValidationFailures:
    """Tests that verify validation errors are surfaced correctly."""

    def test_empty_items_fails(self):
        """An order with no items should fail validation."""
        order = {"customer": "Eve", "items": []}

        with _create_worker() as w:
            w.start()
            c = client.TaskHubGrpcClient(host_address=HOST)
            instance_id = c.schedule_new_orchestration(
                order_processing_orchestrator, input=order
            )
            state = c.wait_for_orchestration_completion(instance_id, timeout=30)

        assert state is not None
        assert state.runtime_status == client.OrchestrationStatus.FAILED
        assert state.failure_details is not None
        assert "at least one item" in state.failure_details.message

    def test_invalid_quantity_fails(self):
        """An item with zero quantity should fail validation."""
        order = {
            "customer": "Mallory",
            "items": [{"name": "Widget", "quantity": 0, "unit_price": 10.00}],
        }

        with _create_worker() as w:
            w.start()
            c = client.TaskHubGrpcClient(host_address=HOST)
            instance_id = c.schedule_new_orchestration(
                order_processing_orchestrator, input=order
            )
            state = c.wait_for_orchestration_completion(instance_id, timeout=30)

        assert state is not None
        assert state.runtime_status == client.OrchestrationStatus.FAILED
        assert state.failure_details is not None
        assert "Invalid quantity" in state.failure_details.message
