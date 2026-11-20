import unittest
from types import SimpleNamespace
from unittest.mock import Mock

from durabletask.client import OrchestrationStatus

from main import run_order
from workflows import call_partner, process_order, record_order, validate_order


class TaskQueueTests(unittest.TestCase):
    def test_routing_and_output(self):
        ctx = Mock()
        workflow = process_order(ctx, "order-001")
        next(workflow)
        ctx.call_activity.assert_called_once_with(record_order, input="order-001")
        workflow.send("Recorded order-001")
        ctx.call_activity.assert_called_with(
            call_partner, input="order-001", tags={"DTS_TaskQueue": "partner-api"}
        )
        workflow.send("Partner accepted order-001")
        ctx.call_sub_orchestrator.assert_called_once_with(validate_order, input="order-001")
        with self.assertRaises(StopIteration) as completed:
            workflow.send("Validated order-001")
        self.assertEqual(
            completed.exception.value,
            "Recorded order-001; Partner accepted order-001; Validated order-001",
        )

    def test_tagged_start_and_failed_completion(self):
        client = Mock()
        client.schedule_new_orchestration.return_value = "instance-001"
        client.wait_for_orchestration_completion.return_value = SimpleNamespace(
            runtime_status=OrchestrationStatus.FAILED, failure_details="activity failed"
        )
        with self.assertRaisesRegex(RuntimeError, "activity failed"):
            run_order(client, "order-001")
        client.schedule_new_orchestration.assert_called_once_with(
            process_order, input="order-001", tags={"DTS_TaskQueue": "orders"}
        )

    def test_activity_failure_propagates(self):
        workflow = process_order(Mock(), "order-001")
        next(workflow)
        with self.assertRaisesRegex(RuntimeError, "recording failed"):
            workflow.throw(RuntimeError("recording failed"))


if __name__ == "__main__":
    unittest.main()
