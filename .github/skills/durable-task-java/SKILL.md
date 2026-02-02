---
name: durable-task-java
description: Build durable, fault-tolerant workflows in Java using the Durable Task SDK with Azure Durable Task Scheduler. Use when creating orchestrations, activities, or implementing patterns like function chaining, fan-out/fan-in, human interaction, or monitoring. Applies to any Java application requiring durable execution, state persistence, or distributed transactions without Azure Functions dependency.
---

# Durable Task Java SDK with Durable Task Scheduler

Build fault-tolerant, stateful workflows in Java applications using the Durable Task SDK connected to Azure Durable Task Scheduler.

## Quick Start

### Gradle Dependencies

```groovy
dependencies {
    implementation 'com.microsoft.durabletask:durabletask-client:1.6.2'
    implementation 'com.microsoft.durabletask:durabletask-azuremanaged:1.6.2'
    implementation 'com.azure:azure-identity:1.11.0'
}
```

### Minimal Worker + Client Setup

```java
import com.microsoft.durabletask.*;
import com.microsoft.durabletask.azuremanaged.*;
import java.time.Duration;

public class DurableTaskApp {
    public static void main(String[] args) throws Exception {
        // Connection string - defaults to local emulator
        String connectionString = System.getenv("DURABLE_TASK_CONNECTION_STRING");
        if (connectionString == null) {
            connectionString = "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";
        }

        // Build and start the worker
        DurableTaskGrpcWorker worker = new DurableTaskGrpcWorkerBuilder()
            .connectionString(connectionString)
            .addOrchestration("MyOrchestration", ctx -> {
                String input = ctx.getInput(String.class);
                String result = ctx.callActivity("SayHello", input, String.class).await();
                return result;
            })
            .addActivity("SayHello", ctx -> {
                String name = ctx.getInput(String.class);
                return "Hello " + name + "!";
            })
            .build();

        worker.start();

        // Build the client
        DurableTaskClient client = new DurableTaskGrpcClientBuilder()
            .connectionString(connectionString)
            .build();

        // Schedule an orchestration
        String instanceId = client.scheduleNewOrchestrationInstance("MyOrchestration", "World");
        System.out.println("Started orchestration: " + instanceId);

        // Wait for completion
        OrchestrationMetadata result = client.waitForInstanceCompletion(
            instanceId, Duration.ofSeconds(60), true);
        
        System.out.println("Result: " + result.readOutputAs(String.class));
        
        worker.close();
        client.close();
    }
}
```

## Pattern Selection Guide

| Pattern | Use When |
|---------|----------|
| **Function Chaining** | Sequential steps where each depends on the previous |
| **Fan-Out/Fan-In** | Parallel processing with aggregated results |
| **Human Interaction** | Workflow pauses for external input/approval |
| **Sub-Orchestrations** | Reusable workflow components or version isolation |
| **Eternal Orchestrations** | Long-running background processes with `continueAsNew` |
| **Monitoring** | Periodic polling with configurable timeouts |

See [references/patterns.md](references/patterns.md) for detailed implementations.

## Orchestration Structure

### Basic Orchestrator

```java
// Orchestrator function - MUST be deterministic
.addOrchestration("OrderWorkflow", ctx -> {
    OrderInfo order = ctx.getInput(OrderInfo.class);
    
    // Call activities sequentially
    boolean valid = ctx.callActivity("ValidateOrder", order, Boolean.class).await();
    if (!valid) {
        return "Order invalid";
    }
    
    String result = ctx.callActivity("ProcessOrder", order, String.class).await();
    return result;
})
```

### Basic Activity

```java
// Activity function - can have side effects, I/O, non-determinism
.addActivity("ProcessOrder", ctx -> {
    OrderInfo order = ctx.getInput(OrderInfo.class);
    
    // Perform actual work here - HTTP calls, database, etc.
    System.out.println("Processing order: " + order.getOrderId());
    
    return "Order " + order.getOrderId() + " processed";
})
```

## Critical Rules

### Orchestration Determinism

Orchestrations replay from history - all code MUST be deterministic. When an orchestration resumes, it replays all previous code to rebuild state. Non-deterministic code produces different results on replay, causing failures.

**NEVER do inside orchestrations:**
- `Instant.now()`, `LocalDateTime.now()`, `new Date()` → Use `ctx.getCurrentInstant()`
- `UUID.randomUUID()` → Use `ctx.newUUID()`
- `new Random()` → Pass random values from activities
- Direct I/O, HTTP calls, database access → Move to activities
- `Thread.sleep()` → Use `ctx.createTimer()`
- `System.getenv()` that may change → Pass as input or use activities
- HashMap/HashSet iteration (non-deterministic order) → Use TreeMap/TreeSet

**ALWAYS use:**
- `ctx.callActivity("Name", input, Type.class).await()` - Call activities
- `ctx.callSubOrchestrator("Name", input, Type.class).await()` - Sub-orchestrations
- `ctx.createTimer(Duration).await()` - Durable delays
- `ctx.waitForExternalEvent("EventName", timeout, Type.class).await()` - External events
- `ctx.getCurrentInstant()` - Current time (deterministic)
- `ctx.newUUID()` - Generate UUIDs (deterministic)
- `ctx.setCustomStatus(status)` - Set status

### Non-Determinism Patterns (WRONG vs CORRECT)

#### Getting Current Time

```java
// WRONG - Instant.now() returns different value on replay
.addOrchestration("BadOrchestration", ctx -> {
    Instant currentTime = Instant.now();  // Non-deterministic!
    if (currentTime.isBefore(deadline)) {
        ctx.callActivity("ProcessNow", null, Void.class).await();
    }
    return null;
})

// CORRECT - ctx.getCurrentInstant() replays consistently
.addOrchestration("GoodOrchestration", ctx -> {
    Instant currentTime = ctx.getCurrentInstant();  // Deterministic
    if (currentTime.isBefore(deadline)) {
        ctx.callActivity("ProcessNow", null, Void.class).await();
    }
    return null;
})
```

#### Generating UUIDs

```java
// WRONG - UUID.randomUUID() generates different value on replay
.addOrchestration("BadOrchestration", ctx -> {
    String orderId = UUID.randomUUID().toString();  // Non-deterministic!
    ctx.callActivity("CreateOrder", orderId, Void.class).await();
    return orderId;
})

// CORRECT - ctx.newUUID() replays the same value
.addOrchestration("GoodOrchestration", ctx -> {
    String orderId = ctx.newUUID().toString();  // Deterministic
    ctx.callActivity("CreateOrder", orderId, Void.class).await();
    return orderId;
})
```

#### Random Numbers

```java
// WRONG - Random produces different values on replay
.addOrchestration("BadOrchestration", ctx -> {
    int delay = new Random().nextInt(10);  // Non-deterministic!
    ctx.createTimer(Duration.ofSeconds(delay)).await();
    return null;
})

// CORRECT - generate random in activity, pass to orchestrator
.addActivity("GetRandomDelay", ctx -> {
    return new Random().nextInt(10);  // OK in activity
})

.addOrchestration("GoodOrchestration", ctx -> {
    int delay = ctx.callActivity("GetRandomDelay", null, Integer.class).await();
    ctx.createTimer(Duration.ofSeconds(delay)).await();  // Deterministic
    return null;
})
```

#### Sleeping/Delays

```java
// WRONG - Thread.sleep blocks and doesn't persist
.addOrchestration("BadOrchestration", ctx -> {
    ctx.callActivity("Step1", null, Void.class).await();
    Thread.sleep(60000);  // Non-durable! Lost on restart
    ctx.callActivity("Step2", null, Void.class).await();
    return null;
})

// CORRECT - ctx.createTimer is durable
.addOrchestration("GoodOrchestration", ctx -> {
    ctx.callActivity("Step1", null, Void.class).await();
    ctx.createTimer(Duration.ofMinutes(1)).await();  // Durable timer
    ctx.callActivity("Step2", null, Void.class).await();
    return null;
})
```

#### HTTP Calls and I/O

```java
// WRONG - HTTP call in orchestrator is non-deterministic
.addOrchestration("BadOrchestration", ctx -> {
    HttpClient client = HttpClient.newHttpClient();
    HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create("https://api.example.com/data"))
        .build();
    HttpResponse<String> response = client.send(request, 
        HttpResponse.BodyHandlers.ofString());  // Non-deterministic!
    return response.body();
})

// CORRECT - move I/O to activity
.addActivity("FetchData", ctx -> {
    String url = ctx.getInput(String.class);
    HttpClient client = HttpClient.newHttpClient();
    HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(url))
        .build();
    HttpResponse<String> response = client.send(request,
        HttpResponse.BodyHandlers.ofString());  // OK in activity
    return response.body();
})

.addOrchestration("GoodOrchestration", ctx -> {
    String data = ctx.callActivity("FetchData", 
        "https://api.example.com/data", String.class).await();  // Deterministic
    return data;
})
```

#### Database Access

```java
// WRONG - database query in orchestrator
.addOrchestration("BadOrchestration", ctx -> {
    Connection conn = DriverManager.getConnection(dbUrl);  // Non-deterministic!
    PreparedStatement stmt = conn.prepareStatement("SELECT * FROM users WHERE id=?");
    // ...
    return null;
})

// CORRECT - database access in activity
.addActivity("GetUser", ctx -> {
    String userId = ctx.getInput(String.class);
    Connection conn = DriverManager.getConnection(dbUrl);  // OK in activity
    PreparedStatement stmt = conn.prepareStatement("SELECT * FROM users WHERE id=?");
    stmt.setString(1, userId);
    ResultSet rs = stmt.executeQuery();
    // ...
    return user;
})

.addOrchestration("GoodOrchestration", ctx -> {
    String userId = ctx.getInput(String.class);
    User user = ctx.callActivity("GetUser", userId, User.class).await();
    return user;
})
```

#### Environment Variables

```java
// WRONG - env var might change between replays
.addOrchestration("BadOrchestration", ctx -> {
    String apiEndpoint = System.getenv("API_ENDPOINT");  // Could change!
    ctx.callActivity("CallApi", apiEndpoint, Void.class).await();
    return null;
})

// CORRECT - pass config as input or read in activity
.addOrchestration("GoodOrchestration", ctx -> {
    Config config = ctx.getInput(Config.class);
    String apiEndpoint = config.getApiEndpoint();  // From input, deterministic
    ctx.callActivity("CallApi", apiEndpoint, Void.class).await();
    return null;
})

// ALSO CORRECT - read env var in activity
.addActivity("CallApi", ctx -> {
    String apiEndpoint = System.getenv("API_ENDPOINT");  // OK in activity
    // make the call...
    return null;
})
```

#### Collection Iteration Order

```java
// WRONG - HashMap iteration order is non-deterministic
.addOrchestration("BadOrchestration", ctx -> {
    Map<String, Object> items = ctx.getInput(HashMap.class);
    for (String key : items.keySet()) {  // Order not guaranteed!
        ctx.callActivity("Process", key, Void.class).await();
    }
    return null;
})

// CORRECT - use TreeMap or sorted keys for deterministic order
.addOrchestration("GoodOrchestration", ctx -> {
    Map<String, Object> items = ctx.getInput(HashMap.class);
    List<String> sortedKeys = new ArrayList<>(items.keySet());
    Collections.sort(sortedKeys);  // Guaranteed order
    for (String key : sortedKeys) {
        ctx.callActivity("Process", key, Void.class).await();
    }
    return null;
})
```

### Using await()

In Java, orchestrator functions use `.await()` to wait for durable operations:

```java
// CORRECT - use await() to get result
String result = ctx.callActivity("MyActivity", input, String.class).await();

// WRONG - forgetting await() returns Task, not result
Task<String> task = ctx.callActivity("MyActivity", input, String.class);  // Returns Task!
```

### Error Handling

```java
.addOrchestration("OrchestrationWithErrorHandling", ctx -> {
    String input = ctx.getInput(String.class);
    try {
        String result = ctx.callActivity("RiskyActivity", input, String.class).await();
        return result;
    } catch (TaskFailedException ex) {
        // Activity failed - implement compensation
        ctx.setCustomStatus(Map.of("error", ex.getMessage()));
        ctx.callActivity("CompensationActivity", input, Void.class).await();
        return "Compensated";
    }
})
```

### Retry Policies

```java
TaskOptions options = new TaskOptions(new RetryPolicy(
    3,                              // maxNumberOfAttempts
    Duration.ofSeconds(5),          // firstRetryInterval
    2.0,                            // backoffCoefficient
    Duration.ofMinutes(1),          // maxRetryInterval
    Duration.ofMinutes(5)           // retryTimeout
));

ctx.callActivity("UnreliableActivity", input, String.class, options).await();
```

## Connection & Authentication

### Connection String Formats

```java
// Local emulator (no auth)
"Endpoint=http://localhost:8080;TaskHub=default;Authentication=None"

// Azure with DefaultAzureCredential
"Endpoint=https://my-scheduler.region.durabletask.io;TaskHub=my-hub;Authentication=DefaultAzure"

// Azure with Managed Identity
"Endpoint=https://my-scheduler.region.durabletask.io;TaskHub=my-hub;Authentication=ManagedIdentity"
```

### Connection Helper

```java
public static String getConnectionString() {
    String endpoint = System.getenv("ENDPOINT");
    String taskHub = System.getenv("TASKHUB");
    
    if (endpoint == null) endpoint = "http://localhost:8080";
    if (taskHub == null) taskHub = "default";
    
    String authType = endpoint.startsWith("http://localhost") ? "None" : "DefaultAzure";
    return String.format("Endpoint=%s;TaskHub=%s;Authentication=%s", 
        endpoint, taskHub, authType);
}
```

## Local Development with Emulator

```bash
# Pull and run the emulator
docker pull mcr.microsoft.com/dts/dts-emulator:latest
docker run -d -p 8080:8080 -p 8082:8082 --name dts-emulator mcr.microsoft.com/dts/dts-emulator:latest

# Dashboard available at http://localhost:8082
```

## Client Operations

```java
DurableTaskClient client = new DurableTaskGrpcClientBuilder()
    .connectionString(connectionString)
    .build();

// Schedule new orchestration
String instanceId = client.scheduleNewOrchestrationInstance("MyOrchestration", input);

// Schedule with custom instance ID
String instanceId = client.scheduleNewOrchestrationInstance(
    "MyOrchestration", input, "my-custom-id");

// Wait for completion
OrchestrationMetadata result = client.waitForInstanceCompletion(
    instanceId, Duration.ofSeconds(60), true);

// Get current status
OrchestrationMetadata state = client.getInstanceMetadata(instanceId, true);

// Raise external event
client.raiseEvent(instanceId, "ApprovalEvent", approvalData);

// Terminate orchestration
client.terminate(instanceId, "User cancelled");

// Suspend/Resume
client.suspend(instanceId, "Pausing for maintenance");
client.resume(instanceId, "Resuming operation");
```

## References

- **[patterns.md](references/patterns.md)** - Detailed pattern implementations (Fan-Out/Fan-In, Human Interaction, Sub-Orchestrations)
- **[setup.md](references/setup.md)** - Azure Durable Task Scheduler provisioning and deployment
