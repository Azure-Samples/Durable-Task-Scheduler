# Monitoring Pattern

## Description of the Sample

This sample demonstrates the monitoring pattern with Azure Durable Functions in Python. The monitoring pattern is used for periodically checking the status of a long-running operation until it completes or times out.

In this sample:
1. The orchestrator starts monitoring a job with a specified ID
2. It periodically calls the `check_job_status` activity at defined intervals
3. The current job status is exposed via custom status, making it available to clients
4. Monitoring continues until either:
   - The job completes successfully
   - The specified timeout period is reached

This pattern is useful for:
- Polling external services or APIs that don't support callbacks
- Checking the status of long-running operations
- Implementing timeout mechanisms for operations with unpredictable durations
- Maintaining state about progress of a workflow

## Sample Architecture

```
HTTP Start → Monitoring Orchestrator
                 ├── Check Job Status (Activity)
                 ├── Wait (Timer)
                 ├── Check Job Status (Activity)
                 ├── Wait (Timer)
                 └── ... (repeat until completion or timeout)
```

## Prerequisites

1. **Azure Storage Emulator** (Azurite) or **Azure Storage Account**
2. **Azure Functions Core Tools** v4.x
3. **Python** 3.8 or higher

## Setup Instructions

### 1. Install Dependencies

```bash
# Navigate to the monitoring sample directory
cd samples/durable-functions/python/monitoring

# Create and activate virtual environment (recommended)
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install required packages
pip install -r requirements.txt
```

### 2. Start Storage Emulator

**Option A: Using Azurite (Recommended)**
```bash
# Install Azurite globally
npm install -g azurite

# Start Azurite
azurite --silent --location ./azurite --debug ./azurite/debug.log
```

**Option B: Using Azure Storage**
Update `local.settings.json` with your storage connection string:
```json
{
  "Values": {
    "AzureWebJobsStorage": "DefaultEndpointsProtocol=https;AccountName=<account>;AccountKey=<key>;EndpointSuffix=core.windows.net"
  }
}
```

### 3. Start the Function App

```bash
# Start the Azure Functions runtime
func start
```

The function app will start on `http://localhost:7071`

## Usage Examples

### 1. Start Job Monitoring (Default Configuration)

**Basic Job Monitoring:**
```bash
curl -X POST http://localhost:7071/api/start_monitoring_job \
  -H "Content-Type: application/json"
```

**Expected Response:**
```json
{
  "id": "abc123def456",
  "statusQueryGetUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abc123def456?taskHub=default",
  "sendEventPostUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abc123def456/raiseEvent/{eventName}?taskHub=default",
  "terminatePostUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abc123def456/terminate?reason={text}&taskHub=default",
  "purgeHistoryDeleteUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abc123def456?taskHub=default"
}
```

### 2. Start Job Monitoring with Custom Parameters

**With Custom Job Configuration:**
```bash
curl -X POST http://localhost:7071/api/start_monitoring_job \
  -H "Content-Type: application/json" \
  -d '{
    "job_id": "my-custom-job-123",
    "polling_interval_seconds": 10,
    "timeout_seconds": 60
  }'
```

### 3. Check Orchestration Status

```bash
curl -X GET "http://localhost:7071/runtime/webhooks/durabletask/instances/{instanceId}?taskHub=default"
```

**Sample Status Response (In Progress):**
```json
{
  "name": "monitoring_job_orchestrator",
  "instanceId": "abc123def456",
  "runtimeStatus": "Running",
  "input": {
    "job_id": "job-uuid-12345",
    "polling_interval_seconds": 5,
    "timeout_seconds": 30
  },
  "customStatus": {
    "job_id": "job-uuid-12345",
    "status": "Running",
    "check_count": 2,
    "last_check_time": "2025-09-19T18:05:15.123Z"
  },
  "output": null,
  "createdTime": "2025-09-19T18:05:00Z",
  "lastUpdatedTime": "2025-09-19T18:05:15Z"
}
```

### 4. Get Job Status Directly

```bash
curl -X GET "http://localhost:7071/api/job_status/{jobId}"
```

**Sample Job Status Response:**
```json
{
  "job_id": "job-uuid-12345",
  "status": "Running",
  "progress_percent": 75,
  "estimated_completion": "2025-09-19T18:15:00Z",
  "last_updated": "2025-09-19T18:05:30.456Z",
  "details": "Processing batch 3 of 4"
}
```

## How the Pattern Works

### 1. Job Status Polling
- The orchestrator periodically calls the `check_job_status` activity
- Each check simulates querying an external service or API
- Job status progresses from "Unknown" → "Running" → "Completed"

### 2. Activity-based Delays
- Uses `wait_for_interval` activity to wait between status checks
- Configurable polling interval (default: 5 seconds)
- Note: Uses activity function instead of `create_timer()` to avoid timer configuration issues in Python SDK

### 3. Custom Status Updates
- Current job status is exposed via `set_custom_status()`
- Clients can query orchestration status to see job progress
- Real-time visibility into monitoring state

### 4. Timeout Handling
- Monitoring stops if timeout period is reached
- Job status is set to "Timeout" if not completed in time
- Prevents infinite monitoring loops

## Sample Outputs

### Completed Job Monitoring Result

```json
{
  "job_id": "job-uuid-12345",
  "final_status": "Completed",
  "checks_performed": 4,
  "monitoring_duration_seconds": 15.6
}
```

### Timed Out Job Monitoring Result

```json
{
  "job_id": "job-uuid-67890",
  "final_status": "Timeout",
  "checks_performed": 6,
  "monitoring_duration_seconds": 30.0
}
```

### Custom Status During Monitoring

```json
{
  "job_id": "job-uuid-12345",
  "status": "Running", 
  "check_count": 3,
  "last_check_time": "2025-09-19T18:05:15.123Z"
}
```

## Configuration Options

### Workflow Configuration
- **workflow_type**: Type of workflow being monitored (string)
- **batch_size**: Total number of items to process (integer, default: 100)
- **failure_rate**: Simulated failure rate for demo (float, 0.0-1.0, default: 0.1)
- **enable_monitoring**: Enable/disable monitoring features (boolean, default: true)

### Alert Thresholds
- **Success Rate Alert**: Triggered when success rate < 90%
- **Performance Alert**: Based on processing time deviations
- **Error Rate Alert**: When error rate exceeds configured threshold

## Monitoring Best Practices

### 1. **Use Custom Status Effectively**
```python
# Update custom status with meaningful progress information
context.set_custom_status({
    "current_phase": "processing",
    "progress": {"completed": 50, "total": 100},
    "metrics": {"success_rate": 95.5}
})
```

### 2. **Implement Structured Logging**
```python
logging.info(f"Batch {batch_num} completed", extra={
    "workflow_id": workflow_id,
    "batch_number": batch_num,
    "items_processed": processed_count,
    "processing_time": processing_time
})
```

### 3. **Track Key Metrics**
- Processing throughput (items/second)
- Success/failure rates
- Resource utilization
- Duration and timing metrics

### 4. **Configure Appropriate Alerts**
- Set meaningful thresholds
- Include actionable information
- Route to appropriate teams
- Provide context for quick resolution

## Integration Examples

### Application Insights Integration
```python
# Add Application Insights logging
import logging
from opencensus.ext.azure.log_exporter import AzureLogHandler

# Configure Application Insights
logging.getLogger().addHandler(AzureLogHandler(
    connection_string="InstrumentationKey=your-key"
))
```

### Event Grid Integration
Configure in `host.json` to publish orchestration events:
```json
{
  "extensions": {
    "durableTask": {
      "notifications": {
        "eventGrid": {
          "topicEndpoint": "https://your-topic.eventgrid.azure.net/",
          "keySettingName": "EventGridKey"
        }
      }
    }
  }
}
```

## Troubleshooting

### Common Issues

1. **Timer Configuration Errors**
   - Error: "replay schema version >= V3 is being used, but timer properties are not defined"
   - Solution: This sample uses activity functions (`wait_for_interval`) instead of `create_timer()` to avoid known timer issues in the Python SDK
   - This is a temporary workaround until the SDK timer configuration is fixed

2. **Storage Connection Issues**
   - Ensure Durable Task Scheduler is running on http://localhost:8080
   - Check `local.settings.json` has correct `DURABLE_TASK_SCHEDULER_CONNECTION_STRING`
   - Verify `TASKHUB_NAME` is set to "default"

3. **Job Status Not Updating**
   - Verify `set_custom_status()` calls in orchestrator
   - Check that `check_job_status` activity is being called successfully
   - Monitor logs for activity execution

### Debug Logs

Enable detailed logging in `host.json`:
```json
{
  "logging": {
    "logLevel": {
      "DurableTask.Core": "Information",
      "DurableTask.AzureStorage": "Information"
    }
  }
}
```

## Related Samples

- **[Function Chaining](../function-chaining/)**: Basic orchestration pattern
- **[Fan-out/Fan-in](../fan-out-fan-in/)**: Parallel processing with aggregation
- **[Human Interaction](../human-interaction/)**: External event handling
- **[Eternal Orchestrations](../eternal-orchestrations/)**: Long-running workflows

## Additional Resources

- [Durable Functions Monitoring Documentation](https://docs.microsoft.com/azure/azure-functions/durable/durable-functions-monitor)
- [Azure Functions Python Developer Guide](https://docs.microsoft.com/azure/azure-functions/functions-reference-python)
- [Application Insights for Azure Functions](https://docs.microsoft.com/azure/azure-functions/functions-monitoring)