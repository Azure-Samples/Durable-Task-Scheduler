# Scheduled Tasks

Python | Durable Task SDK

## Description of the Sample

This sample demonstrates the **recurring schedule** feature of the Azure Durable Task Scheduler using the Python SDK. A schedule periodically starts a target orchestration at a fixed interval, and can be paused, resumed, listed, updated, and deleted at runtime.

This is the Python counterpart to the .NET [ScheduleWebApp](../../dotnet/ScheduleWebApp/) sample. It uses the `durabletask.scheduled` package, which builds the schedule feature on top of durable entities and a helper orchestrator.

In this sample:
1. A schedule is created that runs the `report_orchestrator` every 5 seconds
2. The schedule fires several times, each run executing the target orchestration
3. The schedule is paused, then resumed
4. Schedules are listed by ID prefix
5. The schedule is deleted

This pattern is useful for:
- Periodic background jobs (cache clearing, report generation, cleanup)
- Cron-like recurring workflows without an external scheduler
- Workflows that must be paused/resumed or reconfigured without redeployment

## Prerequisites

1. [Python 3.10+](https://www.python.org/downloads/)
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator)
3. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli) (if using a deployed Durable Task Scheduler)

## Configuring Durable Task Scheduler

There are two ways to run this sample locally:

### Using the Emulator (Recommended)

The emulator simulates a scheduler and taskhub in a Docker container, making it ideal for development and learning.

1. Pull the Docker Image for the Emulator:
   ```bash
   docker pull mcr.microsoft.com/dts/dts-emulator:latest
   ```

2. Run the Emulator:
   ```bash
   docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
   ```

Wait a few seconds for the container to be ready.

Note: The example code automatically uses the default emulator settings (endpoint: `http://localhost:8080`, taskhub: `default`). You don't need to set any environment variables.

### Using a Deployed Scheduler and Taskhub in Azure

Local development with a deployed scheduler:

1. Install the durable task scheduler CLI extension:

    ```bash
    az upgrade
    az extension add --name durabletask --allow-preview true
    ```

2. Create a resource group in a region where the Durable Task Scheduler is available:

    ```bash
    az provider show --namespace Microsoft.DurableTask --query "resourceTypes[?resourceType=='schedulers'].locations | [0]" --out table
    ```

    ```bash
    az group create --name my-resource-group --location <location>
    ```

3. Create a durable task scheduler resource:

    ```bash
    az durabletask scheduler create \
        --resource-group my-resource-group \
        --name my-scheduler \
        --ip-allowlist '["0.0.0.0/0"]' \
        --sku-name "Dedicated" \
        --sku-capacity 1 \
        --tags "{'myattribute':'myvalue'}"
    ```

4. Create a task hub within the scheduler resource:

    ```bash
    az durabletask taskhub create \
        --resource-group my-resource-group \
        --scheduler-name my-scheduler \
        --name "my-taskhub"
    ```

5. Grant the current user permission to connect to the `my-taskhub` task hub:

    ```bash
    subscriptionId=$(az account show --query "id" -o tsv)
    loggedInUser=$(az account show --query "user.name" -o tsv)

    az role assignment create \
        --assignee $loggedInUser \
        --role "Durable Task Data Contributor" \
        --scope "/subscriptions/$subscriptionId/resourceGroups/my-resource-group/providers/Microsoft.DurableTask/schedulers/my-scheduler/taskHubs/my-taskhub"
    ```

## How to Run the Sample

Once you have set up either the emulator or deployed scheduler, follow these steps to run the sample:

1. First, activate your Python virtual environment (if you're using one):
   ```bash
   python -m venv venv
   source venv/bin/activate  # On Windows, use: venv\Scripts\activate
   ```

2. If you're using a deployed scheduler, set environment variables:
   ```bash
   export ENDPOINT=$(az durabletask scheduler show \
       --resource-group my-resource-group \
       --name my-scheduler \
       --query "properties.endpoint" \
       --output tsv)

   export TASKHUB="my-taskhub"
   ```

3. Install the required packages:
   ```bash
   pip install -r requirements.txt
   ```

4. Start the worker in a terminal:
   ```bash
   python worker.py
   ```

5. In a new terminal (with the virtual environment activated if applicable), run the client:
   > **Note:** Remember to set the environment variables again if you're using a deployed scheduler.

   ```bash
   python client.py
   ```

## Expected Output

### Client Output
```
=== Step 1: Create a schedule ===
  Created schedule 'report-every-5s'
  Description: ScheduleDescription(schedule_id='report-every-5s', ...)

  Letting the schedule run for ~12 seconds...

=== Step 2: Pause and resume ===
  Schedule paused
  Schedule resumed

=== Step 3: List schedules ===
  Found 1 schedule(s) with prefix 'report-':
    - report-every-5s: status=ScheduleStatus.ACTIVE, next_run_at=2025-01-01T00:00:10+00:00

=== Step 4: Delete the schedule ===
  Deleted schedule 'report-every-5s'
  get_schedule now returns: None

Done!
```

The worker terminal logs a "Generating report for region 'westus'" line each time the schedule fires.

## Code Walkthrough

### Registering the schedule feature on the worker

Every worker that participates in scheduled tasks must register the schedule
entity and operation orchestrator via `worker.configure_scheduled_tasks()`:

```python
worker.add_orchestrator(report_orchestrator)
worker.configure_scheduled_tasks()
```

### Creating a schedule

The `ScheduledTaskClient` wraps the base client and exposes schedule management:

```python
from durabletask.scheduled import ScheduledTaskClient, ScheduleCreationOptions

scheduled_tasks = ScheduledTaskClient(client)

schedule = scheduled_tasks.create_schedule(ScheduleCreationOptions(
    schedule_id="report-every-5s",
    orchestration_name="report_orchestrator",
    interval=timedelta(seconds=5),
    orchestration_input="westus",
    start_at=datetime.now(timezone.utc),
    start_immediately_if_late=True,
))
```

### Managing a schedule

`create_schedule` returns a `ScheduleClient` for the lifecycle operations:

```python
schedule.pause()
schedule.resume()
schedule.update(ScheduleUpdateOptions(interval=timedelta(seconds=10)))
schedule.delete()
```

### Listing schedules

```python
from durabletask.scheduled import ScheduleQuery

descriptions = scheduled_tasks.list_schedules(
    ScheduleQuery(schedule_id_prefix="report-")
)
```

## Viewing in the Dashboard

- **Emulator:** Navigate to http://localhost:8082 → select the "default" task hub
- **Azure:** Navigate to your Scheduler resource in the Azure Portal → Task Hub → Dashboard URL

Each schedule is backed by a durable entity, and each run appears as an
orchestration instance.

## Related Samples

- [Entities](../entities/) - The durable entity feature that powers schedules
- [Eternal Orchestrations](../eternal-orchestrations/) - Recurring work via `continue_as_new`
- [Orchestration Management](../orchestration-management/) - Restart, query, and purge orchestrations

## Learn More

- [Durable Task Scheduler documentation](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/develop-with-durable-task-scheduler)
