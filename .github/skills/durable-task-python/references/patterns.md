# Durable Task Python Patterns

Detailed implementation patterns for the Durable Task Python SDK.

## Function Chaining

Sequential execution where each step depends on the previous:

```python
def hello(ctx: task.ActivityContext, name: str) -> str:
    """Activity function that returns a greeting"""
    return f'Hello {name}!'


def chained_workflow(ctx: task.OrchestrationContext, _):
    """Orchestrator that chains activity calls sequentially"""
    result1 = yield ctx.call_activity(hello, input='Tokyo')
    result2 = yield ctx.call_activity(hello, input='Seattle')
    result3 = yield ctx.call_activity(hello, input='London')
    return [result1, result2, result3]


# Registration
worker.add_orchestrator(chained_workflow)
worker.add_activity(hello)
```

## Fan-Out/Fan-In

Parallel processing with aggregated results:

```python
import random
from durabletask import task


def get_work_items(ctx: task.ActivityContext, _) -> list[str]:
    """Activity that returns a list of work items"""
    count = random.randint(2, 10)
    return [f'work item {i}' for i in range(count)]


def process_work_item(ctx: task.ActivityContext, item: str) -> int:
    """Activity that processes a single work item"""
    return random.randint(0, 10)


def fanout_fanin_workflow(ctx: task.OrchestrationContext, _):
    """Orchestrator that fans out work and fans in results"""
    # Get work items
    work_items: list[str] = yield ctx.call_activity(get_work_items)
    
    # Fan-out: schedule all work items in parallel
    tasks = [ctx.call_activity(process_work_item, input=item) for item in work_items]
    
    # Fan-in: wait for all to complete
    results: list[int] = yield task.when_all(tasks)
    
    # Return aggregated results
    return {
        'work_items': work_items,
        'results': results,
        'total': sum(results)
    }


# Registration
worker.add_orchestrator(fanout_fanin_workflow)
worker.add_activity(get_work_items)
worker.add_activity(process_work_item)
```

### Batched Fan-Out (Large Scale)

For large numbers of items, process in batches:

```python
def batched_fanout_workflow(ctx: task.OrchestrationContext, _):
    """Process work items in batches to control parallelism"""
    work_items = yield ctx.call_activity(get_work_items)
    
    batch_size = 10
    all_results = []
    
    for i in range(0, len(work_items), batch_size):
        batch = work_items[i:i + batch_size]
        tasks = [ctx.call_activity(process_work_item, input=item) for item in batch]
        batch_results = yield task.when_all(tasks)
        all_results.extend(batch_results)
    
    return {'total': sum(all_results)}
```

## Human Interaction

Workflow that waits for external approval with timeout:

```python
from collections import namedtuple
from dataclasses import dataclass
from datetime import timedelta
from durabletask import task


@dataclass
class Order:
    cost: float
    product: str
    quantity: int


def send_approval_request(ctx: task.ActivityContext, order: Order) -> None:
    """Send notification requesting approval"""
    print(f"Approval needed for {order.product} (${order.cost})")


def place_order(ctx: task.ActivityContext, order: Order) -> str:
    """Place the order after approval"""
    return f"Order placed: {order.quantity}x {order.product}"


def purchase_order_workflow(ctx: task.OrchestrationContext, order: Order):
    """Orchestrator that implements human approval pattern"""
    # Auto-approve small orders
    if order.cost < 1000:
        return "Auto-approved"
    
    # Request approval for larger orders
    yield ctx.call_activity(send_approval_request, input=order)
    
    # Wait for approval OR timeout (whichever comes first)
    approval_event = ctx.wait_for_external_event("approval_received")
    timeout_event = ctx.create_timer(timedelta(hours=24))
    
    winner = yield task.when_any([approval_event, timeout_event])
    
    if winner == timeout_event:
        return "Canceled - approval timeout"
    
    # Order was approved
    yield ctx.call_activity(place_order, input=order)
    approval_details = approval_event.get_result()
    return f"Approved by '{approval_details.approver}'"


# Raising the approval event from client
Approval = namedtuple("Approval", ["approver"])
approval_data = Approval("manager@company.com")
client.raise_orchestration_event(instance_id, "approval_received", data=approval_data)
```

## Durable Timers

Schedule delayed execution:

```python
from datetime import timedelta


def delayed_workflow(ctx: task.OrchestrationContext, _):
    """Orchestrator that waits before continuing"""
    yield ctx.call_activity(start_activity)
    
    # Wait for 5 minutes (survives restarts)
    yield ctx.create_timer(timedelta(minutes=5))
    
    yield ctx.call_activity(continue_activity)
    return "Done"
```

### Scheduled Execution

Execute at a specific time:

```python
from datetime import datetime, timedelta


def scheduled_workflow(ctx: task.OrchestrationContext, scheduled_time: datetime):
    """Execute activity at a specific scheduled time"""
    # Calculate delay from current orchestration time
    delay = scheduled_time - ctx.current_utc_datetime
    
    if delay > timedelta(0):
        yield ctx.create_timer(delay)
    
    result = yield ctx.call_activity(scheduled_activity)
    return result
```

## Sub-Orchestrations

Compose orchestrations from smaller pieces:

```python
def child_orchestration(ctx: task.OrchestrationContext, data: str):
    """Child orchestration that can be called from parents"""
    result1 = yield ctx.call_activity(activity_a, input=data)
    result2 = yield ctx.call_activity(activity_b, input=result1)
    return result2


def parent_orchestration(ctx: task.OrchestrationContext, items: list[str]):
    """Parent orchestration that calls child orchestrations"""
    # Call child orchestrations in parallel
    tasks = [
        ctx.call_sub_orchestrator(child_orchestration, input=item) 
        for item in items
    ]
    results = yield task.when_all(tasks)
    return results


# Registration - both must be registered
worker.add_orchestrator(parent_orchestration)
worker.add_orchestrator(child_orchestration)
worker.add_activity(activity_a)
worker.add_activity(activity_b)
```

## Durable Entities

Stateful objects with operations:

### Function-Based Entity

```python
from durabletask import entities


def counter(ctx: entities.EntityContext, input: int):
    """Function-based entity for a counter"""
    state = ctx.get_state(int, 0)  # Get state with default 0
    
    if ctx.operation == "add":
        state += input
        ctx.set_state(state)
    elif ctx.operation == "subtract":
        state -= input
        ctx.set_state(state)
    elif ctx.operation == "get":
        return state
    elif ctx.operation == "reset":
        ctx.set_state(0)
```

### Class-Based Entity

```python
from durabletask import entities


class Counter(entities.DurableEntity):
    """Class-based entity for a counter"""
    
    def __init__(self):
        self.set_state(0)
    
    def add(self, amount: int):
        current = self.get_state(int, 0)
        self.set_state(current + amount)
    
    def subtract(self, amount: int):
        current = self.get_state(int, 0)
        self.set_state(current - amount)
    
    def get(self) -> int:
        return self.get_state(int, 0)
    
    def reset(self):
        self.set_state(0)
```

### Using Entities from Orchestrations

```python
def workflow_with_entity(ctx: task.OrchestrationContext, _):
    """Orchestration that interacts with entities"""
    entity_id = entities.EntityInstanceId("counter", "my-counter")
    
    # Signal entity (fire-and-forget) - uses entity_id and operation_name params
    ctx.signal_entity(entity_id=entity_id, operation_name="add", input=5)
    
    # Call entity and wait for result - uses entity and operation params (different from signal!)
    value = yield ctx.call_entity(entity=entity_id, operation="get")
    
    return f"Counter value: {value}"
```

### Using Entities from Client

```python
entity_id = entities.EntityInstanceId("counter", "my-counter")
client.signal_entity(entity_id, "add", input=10)
```

### Entity Locking

Ensure exclusive access to multiple entities:

```python
def workflow_with_locks(ctx: task.OrchestrationContext, _):
    """Lock multiple entities for atomic operations"""
    entity_id_1 = entities.EntityInstanceId("account", "account-1")
    entity_id_2 = entities.EntityInstanceId("account", "account-2")
    
    # Lock entities for exclusive access
    with (yield ctx.lock_entities([entity_id_1, entity_id_2])):
        # Perform atomic operations on both entities
        # Note: call_entity uses 'entity' and 'operation' params
        balance1 = yield ctx.call_entity(entity=entity_id_1, operation="get_balance")
        balance2 = yield ctx.call_entity(entity=entity_id_2, operation="get_balance")
        
        # Transfer between accounts
        yield ctx.call_entity(entity=entity_id_1, operation="withdraw", input=100)
        yield ctx.call_entity(entity=entity_id_2, operation="deposit", input=100)
    
    return "Transfer complete"
```

## Eternal Orchestrations (Continue-As-New)

Long-running processes that periodically restart:

```python
from datetime import timedelta


def eternal_orchestration(ctx: task.OrchestrationContext, iteration: int):
    """Orchestration that runs forever, restarting periodically"""
    # Do periodic work
    yield ctx.call_activity(periodic_work, input=iteration)
    
    # Wait before next iteration
    yield ctx.create_timer(timedelta(minutes=5))
    
    # Restart with new iteration count
    # This prevents history from growing unbounded
    ctx.continue_as_new(iteration + 1)


# Start with iteration 0
client.schedule_new_orchestration(eternal_orchestration, input=0)
```

## Monitoring Pattern

Periodic polling with flexible exit conditions:

```python
from datetime import timedelta


def monitoring_workflow(ctx: task.OrchestrationContext, job_id: str):
    """Monitor a job until completion or timeout"""
    max_attempts = 10
    polling_interval = timedelta(seconds=30)
    
    for attempt in range(max_attempts):
        status = yield ctx.call_activity(check_job_status, input=job_id)
        
        if status == "completed":
            return {"status": "success", "attempts": attempt + 1}
        
        if status == "failed":
            return {"status": "failed", "attempts": attempt + 1}
        
        # Wait before next poll
        yield ctx.create_timer(polling_interval)
    
    return {"status": "timeout", "attempts": max_attempts}
```

## Version-Aware Orchestration

Handle breaking changes gracefully using orchestration versioning. Version is set when scheduling the orchestration and read via `ctx.version`.

### Setting Version When Scheduling

```python
# Schedule orchestration with a specific version
instance_id = client.schedule_new_orchestration(
    "versioned_orchestration",
    input="data",
    version="2.0.0"  # Version is set here
)
```

### Version Comparison Helper

Use the `packaging` module for semantic version comparison:

```python
from packaging import version

def compare_version(v1: str | None, v2: str) -> int:
    """Compare two version strings.
    
    Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
    """
    if v1 is None:
        return -1
    try:
        ver1 = version.parse(v1)
        ver2 = version.parse(v2)
        if ver1 < ver2:
            return -1
        elif ver1 > ver2:
            return 1
        return 0
    except Exception:
        # Fall back to string comparison
        return (v1 > v2) - (v1 < v2)
```

### Versioned Orchestration Example

```python
def versioned_orchestration(ctx: task.OrchestrationContext, name: str):
    """Orchestrator that handles multiple versions.
    
    Version history:
    - v1.0.0: Basic hello greeting
    - v2.0.0: Added goodbye greeting
    """
    results = []
    orch_version = ctx.version  # Read version set during scheduling
    
    # v1.0.0+: Always run this
    hello = yield ctx.call_activity(say_hello, input=name)
    results.append(hello)
    
    # v2.0.0+: Added in version 2
    if compare_version(orch_version, "2.0.0") >= 0:
        goodbye = yield ctx.call_activity(say_goodbye, input=name)
        results.append(goodbye)
    
    return {"version": orch_version, "results": results}
```

### Why Versioning Matters

Without versioning, changing orchestration logic causes **non-deterministic errors**:
1. v1 orchestration starts (calls activity A, then B)
2. You deploy v2 code (calls A, C, then B)
3. v1 orchestration replays but hits new code path
4. **ERROR**: History doesn't match - expected B, got C

With versioning:
- v1 orchestrations continue using v1 code path
- v2 orchestrations use v2 code path
- Both run on the same worker without conflict
