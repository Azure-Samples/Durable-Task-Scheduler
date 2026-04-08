# In-Memory Testing

Python | Durable Task SDK

## Description of the Sample

This sample demonstrates how to unit test Durable Task orchestrations and activities using the in-memory backend (`InMemoryOrchestrationBackend`) — no Docker, no emulator, and no Azure resources required.

In this sample:
1. `workflows.py` defines a pure `order_processing_orchestrator` with `validate_order`, `charge_payment`, and `ship_order` activities
2. `test_workflows.py` uses `create_test_backend()` from `durabletask.testing` to run orchestrations entirely in-process
3. Tests verify both happy paths (successful orders) and failure paths (validation errors)

This pattern is useful for:
- Fast feedback loops during development — tests run in seconds
- CI/CD pipelines where Docker is unavailable or adds overhead
- Verifying orchestration logic in isolation before deploying to Azure
- Test-driven development of durable workflows

## Prerequisites

1. [Python 3.10+](https://www.python.org/downloads/)
2. No Docker or emulator required!

## How to Run the Sample

1. First, activate your Python virtual environment (if you're using one):
   ```bash
   python -m venv venv
   source venv/bin/activate  # On Windows, use: venv\Scripts\activate
   ```

2. Install the required packages:
   ```bash
   pip install -r requirements.txt
   ```

3. Run the tests:
   ```bash
   pytest test_workflows.py -v
   ```

## Expected Output

```
test_workflows.py::TestOrderProcessing::test_single_item_order PASSED
test_workflows.py::TestOrderProcessing::test_multi_item_order PASSED
test_workflows.py::TestOrderValidationFailures::test_empty_items_fails PASSED
test_workflows.py::TestOrderValidationFailures::test_invalid_quantity_fails PASSED

========================= 4 passed =========================
```

## Code Walkthrough

### Separating Workflow Logic

Keep orchestrators and activities in a separate module (`workflows.py`) with no infrastructure dependencies:

```python
from durabletask import task

def validate_order(ctx: task.ActivityContext, order) -> None:
    if not order.items:
        raise ValueError("Order must contain at least one item")

def order_processing_orchestrator(ctx: task.OrchestrationContext, order):
    yield ctx.call_activity(validate_order, input=order)
    # ... more activities
```

### Writing Tests with the In-Memory Backend

Use `create_test_backend()` to spin up a lightweight gRPC server, then connect a standard `TaskHubGrpcWorker` and `TaskHubGrpcClient`:

```python
from durabletask import client, worker
from durabletask.testing import create_test_backend

@pytest.fixture(autouse=True)
def backend():
    b = create_test_backend(port=50061)
    yield b
    b.stop()
    b.reset()

def test_my_orchestration(backend):
    w = worker.TaskHubGrpcWorker(host_address="localhost:50061")
    w.add_orchestrator(my_orchestrator)
    w.add_activity(my_activity)

    with w:
        w.start()
        c = client.TaskHubGrpcClient(host_address="localhost:50061")
        instance_id = c.schedule_new_orchestration(my_orchestrator, input="data")
        state = c.wait_for_orchestration_completion(instance_id, timeout=30)

    assert state.runtime_status == client.OrchestrationStatus.COMPLETED
```

Key points:
- Use `TaskHubGrpcWorker` and `TaskHubGrpcClient` (not the DTS-specific classes) since the in-memory backend implements the same gRPC interface
- Call `backend.stop()` and `backend.reset()` after each test to clean up state
- The `timeout` parameter on `wait_for_orchestration_completion` prevents tests from hanging

## Related Samples

- [Function Chaining](../function-chaining/) - Basic sequential workflow pattern
- [Human Interaction](../human-interaction/) - Approval workflows with external events

## Learn More

- [Durable Task Scheduler documentation](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/develop-with-durable-task-scheduler)
