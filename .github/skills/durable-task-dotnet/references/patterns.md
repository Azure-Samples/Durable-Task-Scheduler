# Durable Task Workflow Patterns

## Table of Contents
- [Function Chaining](#function-chaining)
- [Fan-Out/Fan-In](#fan-outfan-in)
- [Human Interaction](#human-interaction)
- [Durable Entities](#durable-entities)
- [Sub-Orchestrations](#sub-orchestrations)
- [Scheduling/Timers](#schedulingtimers)

---

## Function Chaining

Sequential execution where output of one activity feeds into the next.

### Implementation

```csharp
[DurableTask(nameof(OrderProcessingOrchestration))]
public class OrderProcessingOrchestration : TaskOrchestrator<OrderRequest, OrderResult>
{
    public override async Task<OrderResult> RunAsync(TaskOrchestrationContext context, OrderRequest order)
    {
        // Step 1: Validate order
        var validationResult = await context.CallActivityAsync<ValidationResult>(
            nameof(ValidateOrderActivity), order);
        
        if (!validationResult.IsValid)
            return new OrderResult { Status = "ValidationFailed", Message = validationResult.Error };
        
        // Step 2: Process payment
        var paymentResult = await context.CallActivityAsync<PaymentResult>(
            nameof(ProcessPaymentActivity), order);
        
        // Step 3: Reserve inventory
        var inventoryResult = await context.CallActivityAsync<InventoryResult>(
            nameof(ReserveInventoryActivity), new ReservationRequest 
            { 
                OrderId = order.Id, 
                PaymentId = paymentResult.TransactionId 
            });
        
        // Step 4: Ship order
        var shippingResult = await context.CallActivityAsync<ShippingResult>(
            nameof(ShipOrderActivity), new ShipRequest 
            { 
                OrderId = order.Id, 
                InventoryReservationId = inventoryResult.ReservationId 
            });
        
        return new OrderResult 
        { 
            Status = "Completed",
            TrackingNumber = shippingResult.TrackingNumber 
        };
    }
}

[DurableTask(nameof(ValidateOrderActivity))]
public class ValidateOrderActivity : TaskActivity<OrderRequest, ValidationResult>
{
    public override Task<ValidationResult> RunAsync(TaskActivityContext context, OrderRequest order)
    {
        // Validation logic
        if (order.Items == null || !order.Items.Any())
            return Task.FromResult(new ValidationResult { IsValid = false, Error = "No items in order" });
        
        return Task.FromResult(new ValidationResult { IsValid = true });
    }
}
```

---

## Fan-Out/Fan-In

Execute multiple activities in parallel, then aggregate results.

### Implementation

```csharp
[DurableTask(nameof(BatchProcessingOrchestration))]
public class BatchProcessingOrchestration : TaskOrchestrator<List<string>, ProcessingResults>
{
    public override async Task<ProcessingResults> RunAsync(
        TaskOrchestrationContext context, List<string> workItems)
    {
        // Fan-out: create parallel tasks
        var parallelTasks = new List<Task<ItemResult>>();
        
        foreach (var item in workItems)
        {
            var task = context.CallActivityAsync<ItemResult>(
                nameof(ProcessItemActivity), item);
            parallelTasks.Add(task);
        }
        
        // Wait for all parallel tasks
        var results = await Task.WhenAll(parallelTasks);
        
        // Fan-in: aggregate results
        var aggregatedResult = await context.CallActivityAsync<ProcessingResults>(
            nameof(AggregateResultsActivity), results);
        
        return aggregatedResult;
    }
}

[DurableTask(nameof(ProcessItemActivity))]
public class ProcessItemActivity : TaskActivity<string, ItemResult>
{
    private readonly ILogger<ProcessItemActivity> _logger;
    
    public ProcessItemActivity(ILoggerFactory loggerFactory)
    {
        _logger = loggerFactory.CreateLogger<ProcessItemActivity>();
    }

    public override async Task<ItemResult> RunAsync(TaskActivityContext context, string item)
    {
        _logger.LogInformation("Processing item: {Item}", item);
        
        // Simulate processing
        await Task.Delay(TimeSpan.FromMilliseconds(100));
        
        return new ItemResult 
        { 
            ItemId = item, 
            ProcessedValue = item.Length,
            ProcessedAt = DateTime.UtcNow 
        };
    }
}

[DurableTask(nameof(AggregateResultsActivity))]
public class AggregateResultsActivity : TaskActivity<ItemResult[], ProcessingResults>
{
    public override Task<ProcessingResults> RunAsync(
        TaskActivityContext context, ItemResult[] results)
    {
        return Task.FromResult(new ProcessingResults
        {
            TotalItems = results.Length,
            TotalValue = results.Sum(r => r.ProcessedValue),
            ItemResults = results.ToList()
        });
    }
}
```

### With Batching for Large Workloads

```csharp
public override async Task<ProcessingResults> RunAsync(
    TaskOrchestrationContext context, List<string> workItems)
{
    const int batchSize = 10;
    var allResults = new List<ItemResult>();
    
    // Process in batches to avoid overwhelming resources
    for (int i = 0; i < workItems.Count; i += batchSize)
    {
        var batch = workItems.Skip(i).Take(batchSize).ToList();
        
        var batchTasks = batch.Select(item => 
            context.CallActivityAsync<ItemResult>(nameof(ProcessItemActivity), item));
        
        var batchResults = await Task.WhenAll(batchTasks);
        allResults.AddRange(batchResults);
        
        // Update progress
        context.SetCustomStatus(new { 
            Processed = Math.Min(i + batchSize, workItems.Count), 
            Total = workItems.Count 
        });
    }
    
    return await context.CallActivityAsync<ProcessingResults>(
        nameof(AggregateResultsActivity), allResults.ToArray());
}
```

---

## Human Interaction

Pause workflow for external approval or input.

### Implementation

```csharp
[DurableTask(nameof(ApprovalOrchestration))]
public class ApprovalOrchestration : TaskOrchestrator<ApprovalRequest, ApprovalResult>
{
    private const string ApprovalEventName = "approval_response";
    
    public override async Task<ApprovalResult> RunAsync(
        TaskOrchestrationContext context, ApprovalRequest request)
    {
        // Step 1: Submit approval request
        var submission = await context.CallActivityAsync<SubmissionResult>(
            nameof(SubmitApprovalRequestActivity), request);
        
        // Make request details available via custom status
        context.SetCustomStatus(new { 
            RequestId = request.Id,
            Status = "PendingApproval",
            SubmittedAt = submission.SubmittedAt,
            ApprovalUrl = submission.ApprovalUrl
        });
        
        // Step 2: Wait for approval or timeout
        var timeout = context.CurrentUtcDateTime.Add(request.TimeoutDuration);
        
        using var timeoutCts = new CancellationTokenSource();
        var timeoutTask = context.CreateTimer(timeout, timeoutCts.Token);
        var approvalTask = context.WaitForExternalEvent<ApprovalResponse>(ApprovalEventName);
        
        var completedTask = await Task.WhenAny(approvalTask, timeoutTask);
        
        ApprovalResult result;
        
        if (completedTask == approvalTask)
        {
            // Approval received before timeout
            timeoutCts.Cancel();
            var response = approvalTask.Result;
            
            result = await context.CallActivityAsync<ApprovalResult>(
                nameof(ProcessApprovalActivity), new ProcessApprovalInput
                {
                    Request = request,
                    Response = response
                });
        }
        else
        {
            // Timeout occurred
            result = new ApprovalResult
            {
                RequestId = request.Id,
                Status = ApprovalStatus.TimedOut,
                ProcessedAt = context.CurrentUtcDateTime
            };
        }
        
        // Notify requester of outcome
        await context.CallActivityAsync(nameof(SendNotificationActivity), new NotificationInput
        {
            RequestId = request.Id,
            Result = result
        });
        
        return result;
    }
}

// Client code to send approval
public async Task ApproveRequest(DurableTaskClient client, string instanceId, bool approved)
{
    var response = new ApprovalResponse
    {
        IsApproved = approved,
        ApprovedBy = "user@example.com",
        Comments = approved ? "Looks good!" : "Rejected due to policy violation"
    };
    
    await client.RaiseEventAsync(instanceId, "approval_response", response);
}
```

### Multi-Step Approval

```csharp
public override async Task<ApprovalResult> RunAsync(
    TaskOrchestrationContext context, MultiLevelApprovalRequest request)
{
    var approvers = new[] { "manager", "director", "vp" };
    
    foreach (var level in approvers)
    {
        if (request.Amount < GetThresholdForLevel(level))
            continue;
        
        context.SetCustomStatus(new { 
            CurrentLevel = level, 
            PendingApproval = true 
        });
        
        // Wait for approval at this level
        var response = await context.WaitForExternalEvent<ApprovalResponse>($"approval_{level}");
        
        if (!response.IsApproved)
        {
            return new ApprovalResult { Status = ApprovalStatus.Rejected, RejectedBy = level };
        }
    }
    
    return new ApprovalResult { Status = ApprovalStatus.Approved };
}
```

---

## Durable Entities

Stateful objects for managing state with atomic operations.

### Entity Definition

```csharp
public interface IAccountEntity
{
    void Deposit(decimal amount);
    void Withdraw(decimal amount);
    decimal GetBalance();
    void Reset();
}

[DurableTask(nameof(AccountEntity))]
public class AccountEntity : TaskEntity<AccountState>, IAccountEntity
{
    public void Deposit(decimal amount)
    {
        if (amount <= 0)
            throw new ArgumentException("Amount must be positive");
        
        State.Balance += amount;
        State.LastModified = DateTime.UtcNow;
        State.TransactionHistory.Add(new Transaction 
        { 
            Type = "Deposit", 
            Amount = amount, 
            Timestamp = State.LastModified 
        });
    }

    public void Withdraw(decimal amount)
    {
        if (amount <= 0)
            throw new ArgumentException("Amount must be positive");
        
        if (State.Balance < amount)
            throw new InvalidOperationException("Insufficient funds");
        
        State.Balance -= amount;
        State.LastModified = DateTime.UtcNow;
        State.TransactionHistory.Add(new Transaction 
        { 
            Type = "Withdrawal", 
            Amount = amount, 
            Timestamp = State.LastModified 
        });
    }

    public decimal GetBalance() => State.Balance;

    public void Reset()
    {
        State = new AccountState();
    }
    
    protected override AccountState InitializeState() => new AccountState();
}

public class AccountState
{
    public decimal Balance { get; set; }
    public DateTime LastModified { get; set; }
    public List<Transaction> TransactionHistory { get; set; } = new();
}
```

### Using Entities from Orchestrations

```csharp
[DurableTask(nameof(TransferFundsOrchestration))]
public class TransferFundsOrchestration : TaskOrchestrator<TransferRequest, TransferResult>
{
    public override async Task<TransferResult> RunAsync(
        TaskOrchestrationContext context, TransferRequest request)
    {
        var sourceEntity = new EntityInstanceId(nameof(AccountEntity), request.SourceAccountId);
        var destEntity = new EntityInstanceId(nameof(AccountEntity), request.DestinationAccountId);
        
        // Check source balance
        var sourceBalance = await context.Entities.CallEntityAsync<decimal>(
            sourceEntity, nameof(IAccountEntity.GetBalance));
        
        if (sourceBalance < request.Amount)
        {
            return new TransferResult 
            { 
                Success = false, 
                Error = "Insufficient funds" 
            };
        }
        
        // Perform transfer atomically using critical section
        using (await context.Entities.LockEntitiesAsync(sourceEntity, destEntity))
        {
            try
            {
                // Withdraw from source
                await context.Entities.CallEntityAsync(
                    sourceEntity, nameof(IAccountEntity.Withdraw), request.Amount);
                
                // Deposit to destination
                await context.Entities.CallEntityAsync(
                    destEntity, nameof(IAccountEntity.Deposit), request.Amount);
                
                return new TransferResult 
                { 
                    Success = true, 
                    TransactionId = context.NewGuid().ToString() 
                };
            }
            catch (Exception ex)
            {
                // Compensation logic if needed
                return new TransferResult { Success = false, Error = ex.Message };
            }
        }
    }
}
```

### Signaling Entities from Client

```csharp
// Fire-and-forget signal (one-way)
await client.Entities.SignalEntityAsync(
    new EntityInstanceId(nameof(AccountEntity), "account-123"),
    nameof(IAccountEntity.Deposit),
    100.00m);

// Query entity state
var balance = await client.Entities.GetEntityAsync<AccountState>(
    new EntityInstanceId(nameof(AccountEntity), "account-123"));
```

---

## Sub-Orchestrations

Compose orchestrations for modularity and version management.

### Implementation

```csharp
[DurableTask(nameof(MainOrchestration))]
public class MainOrchestration : TaskOrchestrator<MainRequest, MainResult>
{
    public override async Task<MainResult> RunAsync(
        TaskOrchestrationContext context, MainRequest request)
    {
        // Call sub-orchestration
        var orderResult = await context.CallSubOrchestrationAsync<OrderResult>(
            nameof(OrderProcessingOrchestration),
            new OrderRequest { CustomerId = request.CustomerId, Items = request.Items });
        
        // Call another sub-orchestration with custom instance ID
        var notificationResult = await context.CallSubOrchestrationAsync<NotificationResult>(
            nameof(NotificationOrchestration),
            new NotificationRequest { OrderId = orderResult.OrderId },
            new SubOrchestrationOptions 
            { 
                InstanceId = $"notification-{orderResult.OrderId}" 
            });
        
        return new MainResult
        {
            OrderResult = orderResult,
            NotificationResult = notificationResult
        };
    }
}
```

### Parallel Sub-Orchestrations

```csharp
public override async Task<BatchResult> RunAsync(
    TaskOrchestrationContext context, List<string> customerIds)
{
    var tasks = customerIds.Select(customerId => 
        context.CallSubOrchestrationAsync<CustomerResult>(
            nameof(ProcessCustomerOrchestration),
            new CustomerRequest { CustomerId = customerId },
            new SubOrchestrationOptions { InstanceId = $"customer-{customerId}" }));
    
    var results = await Task.WhenAll(tasks);
    
    return new BatchResult { CustomerResults = results.ToList() };
}
```

---

## Scheduling/Timers

Implement delays, schedules, and recurring workflows.

### Delayed Execution

```csharp
public override async Task<ReminderResult> RunAsync(
    TaskOrchestrationContext context, ReminderRequest request)
{
    // Wait until specified time
    var reminderTime = request.ReminderDateTime;
    await context.CreateTimer(reminderTime, CancellationToken.None);
    
    // Send reminder
    return await context.CallActivityAsync<ReminderResult>(
        nameof(SendReminderActivity), request);
}
```

### Recurring Schedule (Eternal Orchestration)

```csharp
[DurableTask(nameof(RecurringJobOrchestration))]
public class RecurringJobOrchestration : TaskOrchestrator<RecurringJobConfig, object?>
{
    public override async Task<object?> RunAsync(
        TaskOrchestrationContext context, RecurringJobConfig config)
    {
        // Execute the job
        await context.CallActivityAsync(nameof(ExecuteJobActivity), config.JobData);
        
        // Calculate next run time
        var nextRun = context.CurrentUtcDateTime.Add(config.Interval);
        
        // Wait for next scheduled time
        await context.CreateTimer(nextRun, CancellationToken.None);
        
        // Continue as new to avoid history buildup
        context.ContinueAsNew(config);
        
        return null;
    }
}
```

### Scheduled with Early Termination

```csharp
public override async Task<object?> RunAsync(
    TaskOrchestrationContext context, ScheduledJobConfig config)
{
    while (context.CurrentUtcDateTime < config.EndDate)
    {
        // Check for cancellation signal
        var cancelTask = context.WaitForExternalEvent<bool>("cancel");
        var timerTask = context.CreateTimer(
            context.CurrentUtcDateTime.Add(config.Interval), 
            CancellationToken.None);
        
        var completedTask = await Task.WhenAny(cancelTask, timerTask);
        
        if (completedTask == cancelTask && cancelTask.Result)
        {
            // Cancelled
            return new { Status = "Cancelled" };
        }
        
        // Execute job
        await context.CallActivityAsync(nameof(ExecuteJobActivity), config.JobData);
    }
    
    return new { Status = "Completed", EndDate = config.EndDate };
}
```
