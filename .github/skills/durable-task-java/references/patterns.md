# Durable Task Java SDK Patterns

Detailed implementations of workflow patterns for Java applications.

> **Important API Note**: The examples below use shorthand lambda syntax for clarity. In production code, use `TaskOrchestrationFactory` and `TaskActivityFactory`:
>
> ```java
> // Worker setup with DurableTaskSchedulerWorkerExtensions
> DurableTaskGrpcWorker worker = DurableTaskSchedulerWorkerExtensions.createWorkerBuilder(connectionString)
>     .addOrchestration(new TaskOrchestrationFactory() {
>         @Override public String getName() { return "MyOrchestration"; }
>         @Override public TaskOrchestration create() {
>             return ctx -> {
>                 // Orchestration logic here
>                 ctx.complete(result);  // Use ctx.complete() to return value
>             };
>         }
>     })
>     .addActivity(new TaskActivityFactory() {
>         @Override public String getName() { return "MyActivity"; }
>         @Override public TaskActivity create() {
>             return ctx -> {
>                 // Activity logic here - return value directly
>                 return result;
>             };
>         }
>     })
>     .build();
>
> // Client builder
> DurableTaskClient client = DurableTaskSchedulerClientExtensions.createClientBuilder(connectionString).build();
> ```

## Function Chaining

Sequential activity execution where each step depends on the previous result.

```java
// Orchestration - process order through sequential steps
.addOrchestration("OrderProcessingWorkflow", ctx -> {
    OrderInput input = ctx.getInput(OrderInput.class);
    
    // Step 1: Validate
    ValidationResult validation = ctx.callActivity(
        "ValidateOrder", input, ValidationResult.class).await();
    
    if (!validation.isValid()) {
        return new OrderResult(false, validation.getErrors());
    }
    
    // Step 2: Calculate pricing
    PricingInfo pricing = ctx.callActivity(
        "CalculatePricing", input, PricingInfo.class).await();
    
    // Step 3: Reserve inventory
    ReservationResult reservation = ctx.callActivity(
        "ReserveInventory", input.getItems(), ReservationResult.class).await();
    
    // Step 4: Process payment
    PaymentResult payment = ctx.callActivity(
        "ProcessPayment", 
        new PaymentInput(input.getPaymentInfo(), pricing.getTotal()),
        PaymentResult.class).await();
    
    // Step 5: Complete order
    OrderResult result = ctx.callActivity(
        "CompleteOrder",
        new CompleteOrderInput(input, reservation, payment),
        OrderResult.class).await();
    
    return result;
})

// Activities
.addActivity("ValidateOrder", ctx -> {
    OrderInput input = ctx.getInput(OrderInput.class);
    // Validation logic
    List<String> errors = new ArrayList<>();
    if (input.getItems().isEmpty()) {
        errors.add("Order must contain at least one item");
    }
    if (input.getPaymentInfo() == null) {
        errors.add("Payment information required");
    }
    return new ValidationResult(errors.isEmpty(), errors);
})

.addActivity("CalculatePricing", ctx -> {
    OrderInput input = ctx.getInput(OrderInput.class);
    double subtotal = input.getItems().stream()
        .mapToDouble(item -> item.getPrice() * item.getQuantity())
        .sum();
    double tax = subtotal * 0.08;
    double shipping = subtotal > 100 ? 0 : 9.99;
    return new PricingInfo(subtotal, tax, shipping, subtotal + tax + shipping);
})
```

## Fan-Out/Fan-In (Parallel Processing)

Execute multiple activities in parallel and aggregate results.

### Basic Parallel Execution

```java
.addOrchestration("ParallelProcessing", ctx -> {
    List<String> items = ctx.getInput(new TypeReference<List<String>>() {});
    
    // Create tasks for all items
    List<Task<ProcessResult>> tasks = new ArrayList<>();
    for (String item : items) {
        Task<ProcessResult> task = ctx.callActivity(
            "ProcessItem", item, ProcessResult.class);
        tasks.add(task);
    }
    
    // Wait for all tasks to complete
    List<ProcessResult> results = ctx.allOf(tasks).await();
    
    // Aggregate results
    int successCount = (int) results.stream()
        .filter(ProcessResult::isSuccess)
        .count();
    
    return new AggregateResult(results.size(), successCount);
})
```

### Batched Parallel Processing (Large Scale)

```java
.addOrchestration("BatchedParallelProcessing", ctx -> {
    List<WorkItem> allItems = ctx.getInput(new TypeReference<List<WorkItem>>() {});
    int batchSize = 10;  // Process 10 at a time to avoid overload
    
    List<ProcessResult> allResults = new ArrayList<>();
    
    // Process in batches
    for (int i = 0; i < allItems.size(); i += batchSize) {
        List<WorkItem> batch = allItems.subList(
            i, Math.min(i + batchSize, allItems.size()));
        
        // Create tasks for this batch
        List<Task<ProcessResult>> batchTasks = new ArrayList<>();
        for (WorkItem item : batch) {
            batchTasks.add(ctx.callActivity(
                "ProcessWorkItem", item, ProcessResult.class));
        }
        
        // Wait for batch to complete
        List<ProcessResult> batchResults = ctx.allOf(batchTasks).await();
        allResults.addAll(batchResults);
        
        // Update status after each batch
        ctx.setCustomStatus(Map.of(
            "processed", i + batch.size(),
            "total", allItems.size()
        ));
    }
    
    return allResults;
})
```

### Fan-Out with Different Activities

```java
.addOrchestration("MultiSourceAggregation", ctx -> {
    String query = ctx.getInput(String.class);
    
    // Fan out to multiple data sources in parallel
    Task<List<Product>> catalogTask = ctx.callActivity(
        "SearchCatalog", query, new TypeReference<List<Product>>() {});
    Task<List<Product>> inventoryTask = ctx.callActivity(
        "SearchInventory", query, new TypeReference<List<Product>>() {});
    Task<List<Product>> warehouseTask = ctx.callActivity(
        "SearchWarehouse", query, new TypeReference<List<Product>>() {});
    
    // Wait for all searches to complete
    ctx.allOf(List.of(catalogTask, inventoryTask, warehouseTask)).await();
    
    // Combine and deduplicate results
    List<Product> combined = ctx.callActivity(
        "MergeResults", 
        new MergeInput(
            catalogTask.await(), 
            inventoryTask.await(), 
            warehouseTask.await()
        ),
        new TypeReference<List<Product>>() {}).await();
    
    return combined;
})
```

## Human Interaction (Approval Workflow)

Workflow that pauses to wait for human input with timeout support.

```java
.addOrchestration("ApprovalWorkflow", ctx -> {
    ApprovalRequest request = ctx.getInput(ApprovalRequest.class);
    Duration approvalTimeout = Duration.ofHours(72);  // 3 days
    
    // Send approval request notification
    ctx.callActivity("SendApprovalRequest", request, Void.class).await();
    ctx.setCustomStatus(Map.of("status", "WaitingForApproval", "requestedAt", ctx.getCurrentInstant().toString()));
    
    // Wait for approval event or timeout
    try {
        ApprovalResponse response = ctx.waitForExternalEvent(
            "ApprovalResponse", approvalTimeout, ApprovalResponse.class).await();
        
        if (response.isApproved()) {
            // Process approved request
            ProcessResult result = ctx.callActivity(
                "ProcessApprovedRequest", request, ProcessResult.class).await();
            
            ctx.callActivity("SendApprovalNotification", 
                new NotificationInput(request, "Approved and processed"), Void.class).await();
            
            return new WorkflowResult("Approved", result);
        } else {
            // Handle rejection
            ctx.callActivity("SendRejectionNotification",
                new RejectionInput(request, response.getReason()), Void.class).await();
            
            return new WorkflowResult("Rejected", response.getReason());
        }
        
    } catch (TaskCanceledException e) {
        // Timeout occurred - escalate
        ctx.callActivity("EscalateApproval", request, Void.class).await();
        
        // Wait for escalation response
        try {
            ApprovalResponse escalatedResponse = ctx.waitForExternalEvent(
                "EscalatedApprovalResponse", Duration.ofHours(24), ApprovalResponse.class).await();
            
            if (escalatedResponse.isApproved()) {
                return new WorkflowResult("ApprovedAfterEscalation", null);
            } else {
                return new WorkflowResult("RejectedAfterEscalation", escalatedResponse.getReason());
            }
        } catch (TaskCanceledException e2) {
            // Final timeout - auto-reject
            ctx.callActivity("SendTimeoutNotification", request, Void.class).await();
            return new WorkflowResult("TimedOut", "No response after escalation");
        }
    }
})

// Activity to send approval request
.addActivity("SendApprovalRequest", ctx -> {
    ApprovalRequest request = ctx.getInput(ApprovalRequest.class);
    // Send email, Teams message, etc.
    // Include link: /api/approval?instanceId=xxx&action=approve
    System.out.println("Approval request sent for: " + request.getDescription());
    return null;
})
```

### Raising Approval Event (from external API)

```java
// In your REST controller or message handler
@PostMapping("/api/approval/{instanceId}")
public ResponseEntity<String> handleApproval(
    @PathVariable String instanceId,
    @RequestBody ApprovalResponse response) {
    
    client.raiseEvent(instanceId, "ApprovalResponse", response);
    return ResponseEntity.ok("Approval recorded");
}
```

## Sub-Orchestrations

Compose workflows from reusable orchestration components.

```java
// Parent orchestration
.addOrchestration("OrderFulfillment", ctx -> {
    OrderRequest order = ctx.getInput(OrderRequest.class);
    
    // Sub-orchestration for payment processing
    PaymentResult payment = ctx.callSubOrchestrator(
        "PaymentProcessingWorkflow",
        order.getPaymentInfo(),
        PaymentResult.class).await();
    
    if (!payment.isSuccessful()) {
        return new FulfillmentResult(false, "Payment failed: " + payment.getError());
    }
    
    // Sub-orchestration for each shipment (parallel)
    List<Task<ShipmentResult>> shipmentTasks = new ArrayList<>();
    for (ShipmentRequest shipment : order.getShipments()) {
        shipmentTasks.add(ctx.callSubOrchestrator(
            "ShipmentWorkflow", shipment, ShipmentResult.class));
    }
    
    List<ShipmentResult> shipmentResults = ctx.allOf(shipmentTasks).await();
    
    // Sub-orchestration for notification
    ctx.callSubOrchestrator(
        "NotificationWorkflow",
        new NotificationRequest(order, shipmentResults),
        Void.class).await();
    
    return new FulfillmentResult(true, shipmentResults);
})

// Child orchestration - payment processing
.addOrchestration("PaymentProcessingWorkflow", ctx -> {
    PaymentInfo payment = ctx.getInput(PaymentInfo.class);
    
    // Validate card
    boolean isValid = ctx.callActivity(
        "ValidatePaymentMethod", payment, Boolean.class).await();
    
    if (!isValid) {
        return new PaymentResult(false, "Invalid payment method");
    }
    
    // Attempt charge with retry
    TaskOptions retryOptions = new TaskOptions(new RetryPolicy(
        3, Duration.ofSeconds(5), 2.0, Duration.ofMinutes(1), null));
    
    ChargeResult charge = ctx.callActivity(
        "ChargePayment", payment, ChargeResult.class, retryOptions).await();
    
    return new PaymentResult(charge.isSuccessful(), charge.getTransactionId());
})

// Child orchestration - shipment
.addOrchestration("ShipmentWorkflow", ctx -> {
    ShipmentRequest shipment = ctx.getInput(ShipmentRequest.class);
    
    // Reserve inventory
    ctx.callActivity("ReserveInventory", shipment.getItems(), Void.class).await();
    
    // Create shipping label
    ShippingLabel label = ctx.callActivity(
        "CreateShippingLabel", shipment, ShippingLabel.class).await();
    
    // Schedule pickup
    PickupConfirmation pickup = ctx.callActivity(
        "SchedulePickup", label, PickupConfirmation.class).await();
    
    return new ShipmentResult(label.getTrackingNumber(), pickup.getScheduledTime());
})
```

## Eternal Orchestrations

Long-running orchestrations that periodically perform work using `continueAsNew`.

```java
.addOrchestration("PeriodicCleanupWorkflow", ctx -> {
    CleanupState state = ctx.getInput(CleanupState.class);
    if (state == null) {
        state = new CleanupState(0, Instant.EPOCH);
    }
    
    // Perform cleanup work
    CleanupResult result = ctx.callActivity(
        "PerformCleanup", state, CleanupResult.class).await();
    
    // Log completion
    ctx.callActivity("LogCleanupResult", result, Void.class).await();
    
    // Update status
    ctx.setCustomStatus(Map.of(
        "lastRun", ctx.getCurrentInstant().toString(),
        "totalRuns", state.getRunCount() + 1,
        "itemsCleaned", result.getItemsCleaned()
    ));
    
    // Wait until next scheduled time
    ctx.createTimer(Duration.ofHours(1)).await();
    
    // Continue as new to prevent history buildup
    // Pass updated state for the next iteration
    CleanupState nextState = new CleanupState(
        state.getRunCount() + 1,
        ctx.getCurrentInstant()
    );
    ctx.continueAsNew(nextState);
    
    return null;  // Never reached due to continueAsNew
})
```

### Graceful Termination for Eternal Orchestrations

```java
.addOrchestration("EternalWorkflowWithGracefulStop", ctx -> {
    WorkflowState state = ctx.getInput(WorkflowState.class);
    if (state == null) {
        state = new WorkflowState(0, false);
    }
    
    // Check for stop signal
    if (state.shouldStop()) {
        ctx.callActivity("PerformFinalCleanup", null, Void.class).await();
        return "Workflow stopped gracefully";
    }
    
    // Do work
    ctx.callActivity("PerformPeriodicWork", state, Void.class).await();
    
    // Wait for either timer or stop event
    Task<Void> timerTask = ctx.createTimer(Duration.ofMinutes(5));
    Task<Boolean> stopTask = ctx.waitForExternalEvent(
        "StopWorkflow", Duration.ofMinutes(5), Boolean.class);
    
    Task<?> winner = ctx.anyOf(List.of(timerTask, stopTask)).await();
    
    // Check if stop event was raised
    boolean shouldStop = false;
    try {
        // If stopTask completed, get its value
        if (stopTask.isDone()) {
            shouldStop = stopTask.await();
        }
    } catch (Exception e) {
        // Timer won or event wasn't raised
    }
    
    // Continue as new
    ctx.continueAsNew(new WorkflowState(state.getIteration() + 1, shouldStop));
    return null;
})
```

## Monitoring Pattern

Periodic polling with configurable timeouts and backoff.

```java
.addOrchestration("MonitorDeployment", ctx -> {
    MonitorConfig config = ctx.getInput(MonitorConfig.class);
    
    Instant startTime = ctx.getCurrentInstant();
    Duration maxDuration = Duration.ofMinutes(config.getTimeoutMinutes());
    Duration pollingInterval = Duration.ofSeconds(config.getInitialPollingSeconds());
    Duration maxPollingInterval = Duration.ofMinutes(5);
    int attempts = 0;
    
    while (true) {
        attempts++;
        
        // Check deployment status
        DeploymentStatus status = ctx.callActivity(
            "CheckDeploymentStatus", config.getDeploymentId(), DeploymentStatus.class).await();
        
        ctx.setCustomStatus(Map.of(
            "status", status.getState(),
            "attempts", attempts,
            "lastCheck", ctx.getCurrentInstant().toString()
        ));
        
        // Check for terminal states
        if (status.isSuccessful()) {
            ctx.callActivity("NotifyDeploymentSuccess", config, Void.class).await();
            return new MonitorResult(true, "Deployment succeeded", attempts);
        }
        
        if (status.isFailed()) {
            ctx.callActivity("NotifyDeploymentFailure", 
                new FailureInfo(config, status.getError()), Void.class).await();
            return new MonitorResult(false, "Deployment failed: " + status.getError(), attempts);
        }
        
        // Check for timeout
        Duration elapsed = Duration.between(startTime, ctx.getCurrentInstant());
        if (elapsed.compareTo(maxDuration) > 0) {
            ctx.callActivity("NotifyDeploymentTimeout", config, Void.class).await();
            return new MonitorResult(false, "Monitoring timed out", attempts);
        }
        
        // Wait before next poll with exponential backoff
        ctx.createTimer(pollingInterval).await();
        
        // Increase polling interval (exponential backoff with cap)
        pollingInterval = pollingInterval.multipliedBy(2);
        if (pollingInterval.compareTo(maxPollingInterval) > 0) {
            pollingInterval = maxPollingInterval;
        }
    }
})
```

## Durable Timers and Scheduled Execution

### Delayed Execution

```java
.addOrchestration("ScheduledReminder", ctx -> {
    ReminderInput input = ctx.getInput(ReminderInput.class);
    
    // Calculate delay until reminder time
    Duration delay = Duration.between(ctx.getCurrentInstant(), input.getReminderTime());
    
    if (!delay.isNegative()) {
        // Wait until reminder time
        ctx.createTimer(delay).await();
    }
    
    // Send the reminder
    ctx.callActivity("SendReminder", input, Void.class).await();
    
    return "Reminder sent";
})
```

### Recurring Schedule

```java
.addOrchestration("RecurringReport", ctx -> {
    ReportConfig config = ctx.getInput(ReportConfig.class);
    
    // Generate and send report
    ReportData report = ctx.callActivity(
        "GenerateReport", config, ReportData.class).await();
    ctx.callActivity("SendReport", report, Void.class).await();
    
    // Calculate time until next run (e.g., next Monday 9 AM)
    Instant nextRun = calculateNextRunTime(ctx.getCurrentInstant(), config.getSchedule());
    Duration delay = Duration.between(ctx.getCurrentInstant(), nextRun);
    
    // Wait until next scheduled time
    ctx.createTimer(delay).await();
    
    // Continue as new for the next iteration
    ctx.continueAsNew(config);
    return null;
})

// Helper method (must be deterministic - no external calls)
private static Instant calculateNextRunTime(Instant current, String schedule) {
    // Parse cron-like schedule and calculate next run
    // This logic must be deterministic
    ZonedDateTime now = current.atZone(ZoneOffset.UTC);
    ZonedDateTime next = now.plusWeeks(1)
        .with(DayOfWeek.MONDAY)
        .withHour(9)
        .withMinute(0)
        .withSecond(0);
    return next.toInstant();
}
```

## Saga Pattern (Distributed Transactions)

Implement compensating transactions for distributed operations.

```java
.addOrchestration("BookingWorkflow", ctx -> {
    BookingRequest request = ctx.getInput(BookingRequest.class);
    
    String flightReservation = null;
    String hotelReservation = null;
    String carReservation = null;
    
    try {
        // Step 1: Reserve flight
        flightReservation = ctx.callActivity(
            "ReserveFlight", request.getFlight(), String.class).await();
        
        // Step 2: Reserve hotel
        hotelReservation = ctx.callActivity(
            "ReserveHotel", request.getHotel(), String.class).await();
        
        // Step 3: Reserve car
        carReservation = ctx.callActivity(
            "ReserveCar", request.getCar(), String.class).await();
        
        // All reservations successful
        return new BookingResult(true, flightReservation, hotelReservation, carReservation);
        
    } catch (TaskFailedException e) {
        // Compensate in reverse order
        List<Task<Void>> compensations = new ArrayList<>();
        
        if (carReservation != null) {
            compensations.add(ctx.callActivity(
                "CancelCarReservation", carReservation, Void.class));
        }
        if (hotelReservation != null) {
            compensations.add(ctx.callActivity(
                "CancelHotelReservation", hotelReservation, Void.class));
        }
        if (flightReservation != null) {
            compensations.add(ctx.callActivity(
                "CancelFlightReservation", flightReservation, Void.class));
        }
        
        if (!compensations.isEmpty()) {
            ctx.allOf(compensations).await();
        }
        
        return new BookingResult(false, "Booking failed: " + e.getMessage());
    }
})
```

## Version-Aware Orchestrations

Handle orchestration versioning for long-running workflows.

```java
.addOrchestration("VersionedWorkflow", ctx -> {
    VersionedInput input = ctx.getInput(VersionedInput.class);
    int version = input.getVersion();
    
    // Route based on version
    if (version == 1) {
        return executeV1(ctx, input);
    } else if (version == 2) {
        return executeV2(ctx, input);
    } else {
        return executeLatest(ctx, input);
    }
})

private static Object executeV1(TaskOrchestrationContext ctx, VersionedInput input) {
    // Original workflow logic
    return ctx.callActivity("ProcessV1", input, Object.class).await();
}

private static Object executeV2(TaskOrchestrationContext ctx, VersionedInput input) {
    // Updated workflow logic with new step
    Object intermediate = ctx.callActivity("ProcessV2Step1", input, Object.class).await();
    return ctx.callActivity("ProcessV2Step2", intermediate, Object.class).await();
}
```

## Type References for Generic Types

When working with generic types like `List<T>` or `Map<K,V>`:

```java
import com.microsoft.durabletask.TypeReference;

// For List types
List<String> items = ctx.callActivity(
    "GetItems", null, new TypeReference<List<String>>() {}).await();

// For Map types
Map<String, Integer> counts = ctx.callActivity(
    "GetCounts", null, new TypeReference<Map<String, Integer>>() {}).await();

// For custom generic types
PagedResult<Customer> customers = ctx.callActivity(
    "GetCustomers", page, new TypeReference<PagedResult<Customer>>() {}).await();
```
