# Durable Functions Patterns Reference

Complete implementations for common Durable Functions workflow patterns.

## Function Chaining

Sequential workflow where each step depends on the previous result:

```csharp
[Function(nameof(ProcessOrderOrchestration))]
public static async Task<OrderResult> ProcessOrderOrchestration(
    [OrchestrationTrigger] TaskOrchestrationContext context)
{
    var order = context.GetInput<Order>()!;
    ILogger logger = context.CreateReplaySafeLogger(nameof(ProcessOrderOrchestration));
    
    logger.LogInformation("Processing order {OrderId}", order.Id);
    
    // Step 1: Validate order
    var validation = await context.CallActivityAsync<ValidationResult>(nameof(ValidateOrder), order);
    if (!validation.IsValid)
    {
        return new OrderResult { Success = false, Error = validation.Error };
    }
    
    // Step 2: Reserve inventory
    var reservation = await context.CallActivityAsync<ReservationResult>(nameof(ReserveInventory), order);
    
    // Step 3: Process payment
    var payment = await context.CallActivityAsync<PaymentResult>(nameof(ProcessPayment), 
        new PaymentRequest { Order = order, ReservationId = reservation.Id });
    
    // Step 4: Ship order
    var shipment = await context.CallActivityAsync<ShipmentResult>(nameof(ShipOrder),
        new ShipmentRequest { Order = order, PaymentId = payment.TransactionId });
    
    // Step 5: Notify customer
    await context.CallActivityAsync(nameof(NotifyCustomer), 
        new Notification { OrderId = order.Id, TrackingNumber = shipment.TrackingNumber });
    
    return new OrderResult { Success = true, TrackingNumber = shipment.TrackingNumber };
}

[Function(nameof(ValidateOrder))]
public static ValidationResult ValidateOrder([ActivityTrigger] Order order) => /* ... */;

[Function(nameof(ReserveInventory))]
public static async Task<ReservationResult> ReserveInventory([ActivityTrigger] Order order) => /* ... */;

[Function(nameof(ProcessPayment))]
public static async Task<PaymentResult> ProcessPayment([ActivityTrigger] PaymentRequest request) => /* ... */;

[Function(nameof(ShipOrder))]
public static async Task<ShipmentResult> ShipOrder([ActivityTrigger] ShipmentRequest request) => /* ... */;

[Function(nameof(NotifyCustomer))]
public static async Task NotifyCustomer([ActivityTrigger] Notification notification) => /* ... */;
```

## Fan-Out/Fan-In

Process items in parallel and aggregate results:

```csharp
[Function(nameof(ParallelProcessingOrchestration))]
public static async Task<BatchResult> ParallelProcessingOrchestration(
    [OrchestrationTrigger] TaskOrchestrationContext context)
{
    var items = context.GetInput<List<WorkItem>>()!;
    ILogger logger = context.CreateReplaySafeLogger(nameof(ParallelProcessingOrchestration));
    
    logger.LogInformation("Processing {Count} items in parallel", items.Count);
    
    // Fan-out: Create tasks for all items
    var tasks = items.Select(item => 
        context.CallActivityAsync<ItemResult>(nameof(ProcessItem), item));
    
    // Fan-in: Wait for all to complete
    ItemResult[] results = await Task.WhenAll(tasks);
    
    // Aggregate results
    return new BatchResult
    {
        TotalProcessed = results.Length,
        Successful = results.Count(r => r.Success),
        Failed = results.Count(r => !r.Success)
    };
}

[Function(nameof(ProcessItem))]
public static async Task<ItemResult> ProcessItem([ActivityTrigger] WorkItem item)
{
    try
    {
        // Process the item
        return new ItemResult { ItemId = item.Id, Success = true };
    }
    catch (Exception ex)
    {
        return new ItemResult { ItemId = item.Id, Success = false, Error = ex.Message };
    }
}
```

### Fan-Out with Batching (Large Scale)

For very large workloads, process in batches to avoid memory issues:

```csharp
[Function(nameof(BatchedFanOutOrchestration))]
public static async Task<BatchResult> BatchedFanOutOrchestration(
    [OrchestrationTrigger] TaskOrchestrationContext context)
{
    var input = context.GetInput<LargeWorkload>()!;
    ILogger logger = context.CreateReplaySafeLogger(nameof(BatchedFanOutOrchestration));
    
    const int batchSize = 100;
    var allResults = new List<ItemResult>();
    
    // Process in batches
    for (int i = 0; i < input.Items.Count; i += batchSize)
    {
        var batch = input.Items.Skip(i).Take(batchSize).ToList();
        logger.LogInformation("Processing batch {BatchNumber} ({Count} items)", 
            i / batchSize + 1, batch.Count);
        
        var batchTasks = batch.Select(item =>
            context.CallActivityAsync<ItemResult>(nameof(ProcessItem), item));
        
        var batchResults = await Task.WhenAll(batchTasks);
        allResults.AddRange(batchResults);
    }
    
    return new BatchResult
    {
        TotalProcessed = allResults.Count,
        Successful = allResults.Count(r => r.Success),
        Failed = allResults.Count(r => !r.Success)
    };
}
```

### Fan-Out with Partial Failure Handling

```csharp
[Function(nameof(ResilientFanOutOrchestration))]
public static async Task<ProcessingResult> ResilientFanOutOrchestration(
    [OrchestrationTrigger] TaskOrchestrationContext context)
{
    var items = context.GetInput<List<WorkItem>>()!;
    var results = new List<ItemResult>();
    var errors = new List<string>();
    
    // Create tasks for all items
    var taskItemPairs = items.Select(item => new
    {
        Task = context.CallActivityAsync<ItemResult>(nameof(ProcessItem), item),
        Item = item
    }).ToList();
    
    // Wait for all tasks, catching individual failures
    foreach (var pair in taskItemPairs)
    {
        try
        {
            var result = await pair.Task;
            results.Add(result);
        }
        catch (TaskFailedException ex)
        {
            errors.Add($"Item {pair.Item.Id} failed: {ex.Message}");
            results.Add(new ItemResult { ItemId = pair.Item.Id, Success = false, Error = ex.Message });
        }
    }
    
    return new ProcessingResult
    {
        Results = results,
        Errors = errors,
        AllSuccessful = errors.Count == 0
    };
}
```

## Human Interaction (Approval Workflow)

Wait for external input with timeout:

```csharp
[Function(nameof(ApprovalOrchestration))]
public static async Task<ApprovalResult> ApprovalOrchestration(
    [OrchestrationTrigger] TaskOrchestrationContext context)
{
    var request = context.GetInput<ApprovalRequest>()!;
    ILogger logger = context.CreateReplaySafeLogger(nameof(ApprovalOrchestration));
    
    // Notify approver
    await context.CallActivityAsync(nameof(SendApprovalRequest), new ApprovalNotification
    {
        RequestId = context.InstanceId,
        Requester = request.Requester,
        Amount = request.Amount,
        Approver = request.Approver
    });
    
    logger.LogInformation("Waiting for approval from {Approver}", request.Approver);
    
    // Wait for approval event with 72-hour timeout
    using var cts = new CancellationTokenSource();
    var approvalTask = context.WaitForExternalEvent<ApprovalResponse>("ApprovalResponse");
    var timeoutTask = context.CreateTimer(context.CurrentUtcDateTime.AddHours(72), cts.Token);
    
    var winner = await Task.WhenAny(approvalTask, timeoutTask);
    
    if (winner == approvalTask)
    {
        cts.Cancel();  // Cancel the timer
        var response = await approvalTask;
        
        if (response.Approved)
        {
            await context.CallActivityAsync(nameof(ExecuteApprovedAction), request);
            return new ApprovalResult { Status = "Approved", ApprovedBy = response.ApproverName };
        }
        else
        {
            return new ApprovalResult { Status = "Rejected", Reason = response.RejectionReason };
        }
    }
    else
    {
        // Timeout - escalate
        logger.LogWarning("Approval timed out, escalating");
        await context.CallActivityAsync(nameof(EscalateApproval), request);
        return new ApprovalResult { Status = "Escalated", Reason = "Approval timeout" };
    }
}

// HTTP endpoint to submit approval
[Function("SubmitApproval")]
public static async Task<HttpResponseData> SubmitApproval(
    [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "approval/{instanceId}")] HttpRequestData req,
    [DurableClient] DurableTaskClient client,
    string instanceId)
{
    var response = await req.ReadFromJsonAsync<ApprovalResponse>();
    
    await client.RaiseEventAsync(instanceId, "ApprovalResponse", response);
    
    var httpResponse = req.CreateResponse(HttpStatusCode.Accepted);
    await httpResponse.WriteAsJsonAsync(new { Message = "Approval submitted" });
    return httpResponse;
}

[Function(nameof(SendApprovalRequest))]
public static async Task SendApprovalRequest([ActivityTrigger] ApprovalNotification notification)
{
    // Send email/notification to approver
}

[Function(nameof(ExecuteApprovedAction))]
public static async Task ExecuteApprovedAction([ActivityTrigger] ApprovalRequest request)
{
    // Execute the approved action
}

[Function(nameof(EscalateApproval))]
public static async Task EscalateApproval([ActivityTrigger] ApprovalRequest request)
{
    // Escalate to manager
}
```

### Multi-Level Approval

```csharp
[Function(nameof(MultiLevelApprovalOrchestration))]
public static async Task<ApprovalResult> MultiLevelApprovalOrchestration(
    [OrchestrationTrigger] TaskOrchestrationContext context)
{
    var request = context.GetInput<MultiLevelApprovalRequest>()!;
    
    foreach (var approver in request.ApprovalChain)
    {
        // Request approval at this level
        await context.CallActivityAsync(nameof(SendApprovalRequest), new ApprovalNotification
        {
            RequestId = context.InstanceId,
            Approver = approver.Email,
            Level = approver.Level
        });
        
        // Wait for this level's approval
        using var cts = new CancellationTokenSource();
        var approvalTask = context.WaitForExternalEvent<ApprovalResponse>($"Approval_{approver.Level}");
        var timeoutTask = context.CreateTimer(context.CurrentUtcDateTime.AddHours(24), cts.Token);
        
        var winner = await Task.WhenAny(approvalTask, timeoutTask);
        
        if (winner == timeoutTask)
        {
            return new ApprovalResult { Status = "TimedOut", Level = approver.Level };
        }
        
        cts.Cancel();
        var response = await approvalTask;
        
        if (!response.Approved)
        {
            return new ApprovalResult 
            { 
                Status = "Rejected", 
                Level = approver.Level,
                Reason = response.RejectionReason 
            };
        }
    }
    
    // All levels approved
    await context.CallActivityAsync(nameof(ExecuteApprovedAction), request);
    return new ApprovalResult { Status = "FullyApproved" };
}
```

## Monitor Pattern

Periodic polling with configurable timeout and exponential backoff:

```csharp
[Function(nameof(ResourceMonitorOrchestration))]
public static async Task<MonitorResult> ResourceMonitorOrchestration(
    [OrchestrationTrigger] TaskOrchestrationContext context)
{
    var config = context.GetInput<MonitorConfig>()!;
    ILogger logger = context.CreateReplaySafeLogger(nameof(ResourceMonitorOrchestration));
    
    DateTime deadline = context.CurrentUtcDateTime.Add(config.MaxDuration);
    TimeSpan pollingInterval = config.InitialPollingInterval;
    int checkCount = 0;
    
    while (context.CurrentUtcDateTime < deadline)
    {
        checkCount++;
        logger.LogInformation("Check #{Count}: Polling resource {ResourceId}", checkCount, config.ResourceId);
        
        var status = await context.CallActivityAsync<ResourceStatus>(
            nameof(CheckResourceStatus), config.ResourceId);
        
        if (status.IsReady)
        {
            logger.LogInformation("Resource {ResourceId} is ready after {Count} checks", 
                config.ResourceId, checkCount);
            return new MonitorResult 
            { 
                Success = true, 
                Status = status,
                CheckCount = checkCount 
            };
        }
        
        if (status.IsFailed)
        {
            logger.LogError("Resource {ResourceId} failed", config.ResourceId);
            return new MonitorResult 
            { 
                Success = false, 
                Error = "Resource provisioning failed",
                Status = status
            };
        }
        
        // Wait before next check (exponential backoff)
        var nextCheck = context.CurrentUtcDateTime.Add(pollingInterval);
        if (nextCheck >= deadline)
        {
            break;  // Don't wait if we'll exceed deadline
        }
        
        await context.CreateTimer(nextCheck, CancellationToken.None);
        
        // Exponential backoff with cap
        pollingInterval = TimeSpan.FromSeconds(
            Math.Min(pollingInterval.TotalSeconds * config.BackoffMultiplier, 
                     config.MaxPollingInterval.TotalSeconds));
    }
    
    // Timeout
    return new MonitorResult 
    { 
        Success = false, 
        Error = "Monitoring timeout exceeded",
        CheckCount = checkCount 
    };
}

[Function(nameof(CheckResourceStatus))]
public static async Task<ResourceStatus> CheckResourceStatus([ActivityTrigger] string resourceId)
{
    // Check resource status via API
    return new ResourceStatus { /* ... */ };
}

public record MonitorConfig
{
    public string ResourceId { get; init; } = "";
    public TimeSpan MaxDuration { get; init; } = TimeSpan.FromHours(2);
    public TimeSpan InitialPollingInterval { get; init; } = TimeSpan.FromSeconds(10);
    public TimeSpan MaxPollingInterval { get; init; } = TimeSpan.FromMinutes(5);
    public double BackoffMultiplier { get; init; } = 1.5;
}
```

## Durable Entities (Aggregator Pattern)

Stateful objects that maintain state across operations:

### Counter Entity

```csharp
// Entity function
[Function(nameof(Counter))]
public static Task Counter([EntityTrigger] TaskEntityDispatcher dispatcher)
    => dispatcher.DispatchAsync<CounterEntity>();

// Entity class
public class CounterEntity
{
    public int Value { get; set; }
    
    public void Add(int amount) => Value += amount;
    public void Subtract(int amount) => Value -= amount;
    public void Reset() => Value = 0;
    public int Get() => Value;
}

// Using entity from orchestration
[Function(nameof(CounterOrchestration))]
public static async Task<int> CounterOrchestration([OrchestrationTrigger] TaskOrchestrationContext context)
{
    var entityId = new EntityInstanceId(nameof(Counter), "myCounter");
    
    // Signal (fire-and-forget)
    await context.Entities.SignalEntityAsync(entityId, "Add", 5);
    await context.Entities.SignalEntityAsync(entityId, "Add", 10);
    
    // Call and get result
    int value = await context.Entities.CallEntityAsync<int>(entityId, "Get");
    
    return value;  // Returns 15
}

// HTTP endpoint to interact with entity
[Function("CounterOperation")]
public static async Task<HttpResponseData> CounterOperation(
    [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "counter/{entityKey}/{operation}")] HttpRequestData req,
    [DurableClient] DurableTaskClient client,
    string entityKey,
    string operation)
{
    var entityId = new EntityInstanceId(nameof(Counter), entityKey);
    var amount = await req.ReadFromJsonAsync<int>();
    
    // Signal the entity
    await client.Entities.SignalEntityAsync(entityId, operation, amount);
    
    var response = req.CreateResponse(HttpStatusCode.Accepted);
    return response;
}

[Function("GetCounter")]
public static async Task<HttpResponseData> GetCounter(
    [HttpTrigger(AuthorizationLevel.Anonymous, "get", Route = "counter/{entityKey}")] HttpRequestData req,
    [DurableClient] DurableTaskClient client,
    string entityKey)
{
    var entityId = new EntityInstanceId(nameof(Counter), entityKey);
    var state = await client.Entities.GetEntityAsync<CounterEntity>(entityId);
    
    var response = req.CreateResponse(HttpStatusCode.OK);
    await response.WriteAsJsonAsync(state?.State ?? new CounterEntity());
    return response;
}
```

### Account Entity with Locking

```csharp
[Function(nameof(BankAccount))]
public static Task BankAccount([EntityTrigger] TaskEntityDispatcher dispatcher)
    => dispatcher.DispatchAsync<BankAccountEntity>();

public class BankAccountEntity
{
    public decimal Balance { get; set; }
    public List<Transaction> History { get; set; } = new();
    
    public bool Deposit(decimal amount)
    {
        if (amount <= 0) return false;
        
        Balance += amount;
        History.Add(new Transaction { Type = "Deposit", Amount = amount, Timestamp = DateTime.UtcNow });
        return true;
    }
    
    public bool Withdraw(decimal amount)
    {
        if (amount <= 0 || Balance < amount) return false;
        
        Balance -= amount;
        History.Add(new Transaction { Type = "Withdrawal", Amount = amount, Timestamp = DateTime.UtcNow });
        return true;
    }
    
    public decimal GetBalance() => Balance;
    public List<Transaction> GetHistory() => History;
}

// Transfer between accounts (uses locking)
[Function(nameof(TransferOrchestration))]
public static async Task<TransferResult> TransferOrchestration([OrchestrationTrigger] TaskOrchestrationContext context)
{
    var request = context.GetInput<TransferRequest>()!;
    
    var fromEntity = new EntityInstanceId(nameof(BankAccount), request.FromAccount);
    var toEntity = new EntityInstanceId(nameof(BankAccount), request.ToAccount);
    
    // Lock both accounts in a consistent order to prevent deadlocks
    var entities = new[] { fromEntity.ToString(), toEntity.ToString() }.OrderBy(x => x).ToArray();
    
    using (await context.Entities.LockEntitiesAsync(
        entities.Select(e => EntityInstanceId.Parse(e)).ToArray()))
    {
        // Check balance
        decimal balance = await context.Entities.CallEntityAsync<decimal>(fromEntity, "GetBalance");
        if (balance < request.Amount)
        {
            return new TransferResult { Success = false, Error = "Insufficient funds" };
        }
        
        // Perform transfer
        await context.Entities.CallEntityAsync<bool>(fromEntity, "Withdraw", request.Amount);
        await context.Entities.CallEntityAsync<bool>(toEntity, "Deposit", request.Amount);
        
        return new TransferResult { Success = true };
    }
}
```

## Sub-Orchestrations

Compose workflows by calling other orchestrations:

```csharp
[Function(nameof(OrderFulfillmentOrchestration))]
public static async Task<FulfillmentResult> OrderFulfillmentOrchestration(
    [OrchestrationTrigger] TaskOrchestrationContext context)
{
    var order = context.GetInput<Order>()!;
    ILogger logger = context.CreateReplaySafeLogger(nameof(OrderFulfillmentOrchestration));
    
    logger.LogInformation("Fulfilling order {OrderId}", order.Id);
    
    // Call sub-orchestration for payment processing
    var paymentResult = await context.CallSubOrchestrationAsync<PaymentResult>(
        nameof(PaymentOrchestration),
        new PaymentRequest { OrderId = order.Id, Amount = order.Total });
    
    if (!paymentResult.Success)
    {
        return new FulfillmentResult { Success = false, Error = "Payment failed" };
    }
    
    // Call sub-orchestration for shipping (with custom instance ID)
    var shipmentResult = await context.CallSubOrchestrationAsync<ShipmentResult>(
        nameof(ShippingOrchestration),
        new ShipmentRequest { OrderId = order.Id, Items = order.Items },
        new SubOrchestrationOptions { InstanceId = $"ship-{order.Id}" });
    
    return new FulfillmentResult 
    { 
        Success = true, 
        PaymentId = paymentResult.TransactionId,
        TrackingNumber = shipmentResult.TrackingNumber 
    };
}

[Function(nameof(PaymentOrchestration))]
public static async Task<PaymentResult> PaymentOrchestration([OrchestrationTrigger] TaskOrchestrationContext context)
{
    var request = context.GetInput<PaymentRequest>()!;
    
    var authResult = await context.CallActivityAsync<AuthResult>(nameof(AuthorizePayment), request);
    if (!authResult.Authorized)
    {
        return new PaymentResult { Success = false, Error = authResult.DeclineReason };
    }
    
    var captureResult = await context.CallActivityAsync<CaptureResult>(nameof(CapturePayment), authResult.AuthorizationId);
    
    return new PaymentResult { Success = true, TransactionId = captureResult.TransactionId };
}

[Function(nameof(ShippingOrchestration))]
public static async Task<ShipmentResult> ShippingOrchestration([OrchestrationTrigger] TaskOrchestrationContext context)
{
    var request = context.GetInput<ShipmentRequest>()!;
    
    var label = await context.CallActivityAsync<ShippingLabel>(nameof(CreateShippingLabel), request);
    await context.CallActivityAsync(nameof(NotifyWarehouse), label);
    
    return new ShipmentResult { TrackingNumber = label.TrackingNumber };
}
```

## Saga Pattern (Distributed Transactions)

Implement compensating transactions for handling partial failures:

```csharp
[Function(nameof(BookTravelOrchestration))]
public static async Task<TravelBookingResult> BookTravelOrchestration(
    [OrchestrationTrigger] TaskOrchestrationContext context)
{
    var request = context.GetInput<TravelBookingRequest>()!;
    ILogger logger = context.CreateReplaySafeLogger(nameof(BookTravelOrchestration));
    
    var completedSteps = new Stack<string>();
    
    try
    {
        // Step 1: Book flight
        var flight = await context.CallActivityAsync<FlightBooking>(nameof(BookFlight), request.Flight);
        completedSteps.Push("flight");
        logger.LogInformation("Flight booked: {ConfirmationNumber}", flight.ConfirmationNumber);
        
        // Step 2: Book hotel
        var hotel = await context.CallActivityAsync<HotelBooking>(nameof(BookHotel), request.Hotel);
        completedSteps.Push("hotel");
        logger.LogInformation("Hotel booked: {ConfirmationNumber}", hotel.ConfirmationNumber);
        
        // Step 3: Book car
        var car = await context.CallActivityAsync<CarBooking>(nameof(BookCar), request.Car);
        completedSteps.Push("car");
        logger.LogInformation("Car booked: {ConfirmationNumber}", car.ConfirmationNumber);
        
        return new TravelBookingResult
        {
            Success = true,
            FlightConfirmation = flight.ConfirmationNumber,
            HotelConfirmation = hotel.ConfirmationNumber,
            CarConfirmation = car.ConfirmationNumber
        };
    }
    catch (TaskFailedException ex)
    {
        logger.LogError(ex, "Booking failed, initiating compensation");
        
        // Compensate in reverse order
        var compensationErrors = new List<string>();
        
        while (completedSteps.Count > 0)
        {
            var step = completedSteps.Pop();
            try
            {
                switch (step)
                {
                    case "car":
                        await context.CallActivityAsync(nameof(CancelCar), request.Car);
                        break;
                    case "hotel":
                        await context.CallActivityAsync(nameof(CancelHotel), request.Hotel);
                        break;
                    case "flight":
                        await context.CallActivityAsync(nameof(CancelFlight), request.Flight);
                        break;
                }
                logger.LogInformation("Compensated: {Step}", step);
            }
            catch (TaskFailedException compEx)
            {
                compensationErrors.Add($"Failed to compensate {step}: {compEx.Message}");
            }
        }
        
        return new TravelBookingResult
        {
            Success = false,
            Error = ex.Message,
            CompensationErrors = compensationErrors
        };
    }
}

// Activity functions
[Function(nameof(BookFlight))]
public static async Task<FlightBooking> BookFlight([ActivityTrigger] FlightRequest request) => /* ... */;

[Function(nameof(BookHotel))]
public static async Task<HotelBooking> BookHotel([ActivityTrigger] HotelRequest request) => /* ... */;

[Function(nameof(BookCar))]
public static async Task<CarBooking> BookCar([ActivityTrigger] CarRequest request) => /* ... */;

[Function(nameof(CancelFlight))]
public static async Task CancelFlight([ActivityTrigger] FlightRequest request) => /* ... */;

[Function(nameof(CancelHotel))]
public static async Task CancelHotel([ActivityTrigger] HotelRequest request) => /* ... */;

[Function(nameof(CancelCar))]
public static async Task CancelCar([ActivityTrigger] CarRequest request) => /* ... */;
```

## Eternal Orchestration (Continue-As-New)

Long-running workflows that periodically restart to manage history size:

```csharp
[Function(nameof(EternalProcessorOrchestration))]
public static async Task EternalProcessorOrchestration([OrchestrationTrigger] TaskOrchestrationContext context)
{
    var state = context.GetInput<ProcessorState>() ?? new ProcessorState();
    ILogger logger = context.CreateReplaySafeLogger(nameof(EternalProcessorOrchestration));
    
    // Check for external stop signal
    bool stopRequested = false;
    try
    {
        stopRequested = await context.WaitForExternalEvent<bool>(
            "StopRequested",
            TimeSpan.Zero);  // Non-blocking check
    }
    catch (TaskCanceledException)
    {
        stopRequested = false;
    }
    
    if (stopRequested)
    {
        logger.LogInformation("Stop requested, exiting eternal orchestration");
        return;
    }
    
    // Do work
    logger.LogInformation("Processing iteration {Iteration}", state.IterationCount);
    
    var newItems = await context.CallActivityAsync<List<WorkItem>>(nameof(GetNewItems), state.LastProcessedId);
    
    if (newItems.Any())
    {
        var tasks = newItems.Select(item => 
            context.CallActivityAsync(nameof(ProcessItem), item));
        await Task.WhenAll(tasks);
        
        state.LastProcessedId = newItems.Max(i => i.Id);
        state.TotalProcessed += newItems.Count;
    }
    
    state.IterationCount++;
    
    // Wait before next iteration
    await context.CreateTimer(context.CurrentUtcDateTime.AddMinutes(1), CancellationToken.None);
    
    // Continue-as-new to prevent unbounded history growth
    context.ContinueAsNew(state);
}

public class ProcessorState
{
    public int IterationCount { get; set; }
    public long LastProcessedId { get; set; }
    public long TotalProcessed { get; set; }
}

// HTTP endpoint to stop the eternal orchestration
[Function("StopEternalOrchestration")]
public static async Task<HttpResponseData> StopEternalOrchestration(
    [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "eternal/{instanceId}/stop")] HttpRequestData req,
    [DurableClient] DurableTaskClient client,
    string instanceId)
{
    await client.RaiseEventAsync(instanceId, "StopRequested", true);
    return req.CreateResponse(HttpStatusCode.Accepted);
}
```

## Scheduled/Timer-Based Workflows

### Delayed Execution

```csharp
[Function(nameof(ScheduledReminderOrchestration))]
public static async Task ScheduledReminderOrchestration([OrchestrationTrigger] TaskOrchestrationContext context)
{
    var reminder = context.GetInput<Reminder>()!;
    
    // Wait until scheduled time
    await context.CreateTimer(reminder.ScheduledTime, CancellationToken.None);
    
    // Send the reminder
    await context.CallActivityAsync(nameof(SendReminder), reminder);
}
```

### Recurring Execution with Cancellation

```csharp
[Function(nameof(RecurringJobOrchestration))]
public static async Task RecurringJobOrchestration([OrchestrationTrigger] TaskOrchestrationContext context)
{
    var config = context.GetInput<RecurringJobConfig>()!;
    ILogger logger = context.CreateReplaySafeLogger(nameof(RecurringJobOrchestration));
    
    DateTime endTime = context.CurrentUtcDateTime.Add(config.TotalDuration);
    
    while (context.CurrentUtcDateTime < endTime)
    {
        logger.LogInformation("Executing scheduled job");
        
        // Execute job
        await context.CallActivityAsync(nameof(ExecuteJob), config.JobParameters);
        
        // Wait for cancel OR next interval
        using var cts = new CancellationTokenSource();
        var cancelTask = context.WaitForExternalEvent<bool>("Cancel");
        var timerTask = context.CreateTimer(
            context.CurrentUtcDateTime.Add(config.Interval),
            cts.Token);
        
        var winner = await Task.WhenAny(cancelTask, timerTask);
        if (winner == cancelTask && await cancelTask)
        {
            logger.LogInformation("Job cancelled");
            return;
        }
        
        cts.Cancel();
    }
    
    logger.LogInformation("Recurring job completed");
}
```

## Version-Aware Orchestrations

Handle orchestration versioning with running instances:

```csharp
[Function(nameof(VersionedOrchestration))]
public static async Task<string> VersionedOrchestration([OrchestrationTrigger] TaskOrchestrationContext context)
{
    var input = context.GetInput<VersionedInput>()!;
    const int CurrentVersion = 2;
    
    // Handle different versions
    if (input.Version < CurrentVersion)
    {
        // Legacy behavior for in-flight instances
        return await LegacyWorkflow(context, input);
    }
    
    // Current version behavior
    return await CurrentWorkflow(context, input);
}

private static async Task<string> LegacyWorkflow(TaskOrchestrationContext context, VersionedInput input)
{
    // Original implementation
    var result = await context.CallActivityAsync<string>(nameof(OldActivity), input.Data);
    return result;
}

private static async Task<string> CurrentWorkflow(TaskOrchestrationContext context, VersionedInput input)
{
    // New improved implementation
    var step1 = await context.CallActivityAsync<string>(nameof(NewActivityStep1), input.Data);
    var step2 = await context.CallActivityAsync<string>(nameof(NewActivityStep2), step1);
    return step2;
}

public record VersionedInput
{
    public int Version { get; init; } = 2;  // Default to current version
    public string Data { get; init; } = "";
}
```
