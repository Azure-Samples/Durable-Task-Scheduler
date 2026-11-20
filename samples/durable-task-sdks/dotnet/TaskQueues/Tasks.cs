using Microsoft.DurableTask;

namespace TaskQueues;

public sealed class ProcessOrder : TaskOrchestrator<string, string>
{
    public override async Task<string> RunAsync(TaskOrchestrationContext context, string orderId)
    {
        string recorded = await context.CallActivityAsync<string>(nameof(RecordOrder), orderId);
        string partner = await context.CallActivityAsync<string>(
            nameof(CallPartner), orderId,
            new TaskOptions(tags: new Dictionary<string, string> { ["DTS_TaskQueue"] = "partner-api" }));
        string validated = await context.CallSubOrchestratorAsync<string>(
            nameof(ValidateOrder), orderId,
            new SubOrchestrationOptions { Tags = new Dictionary<string, string> { ["DTS_TaskQueue"] = "partner-api" } });
        return $"{recorded}; {partner}; {validated}";
    }
}

public sealed class ValidateOrder : TaskOrchestrator<string, string>
{
    public override async Task<string> RunAsync(TaskOrchestrationContext context, string orderId)
    {
        await context.CallActivityAsync<string>(nameof(RecordOrder), orderId);
        return $"Validated {orderId}";
    }
}

public sealed class RecordOrder : TaskActivity<string, string>
{
    public override Task<string> RunAsync(TaskActivityContext context, string orderId)
        => Task.FromResult($"Recorded {orderId}");
}

public sealed class CallPartner : TaskActivity<string, string>
{
    public override Task<string> RunAsync(TaskActivityContext context, string orderId)
        => Task.FromResult($"Partner accepted {orderId}");
}
