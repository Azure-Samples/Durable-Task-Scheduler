using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;

namespace BoundedCoordinator;

/// <summary>
/// Short-lived child orchestration that processes a single work item.
/// Completes quickly and does not accumulate history.
/// </summary>
[DurableTask(nameof(ProcessItemOrchestration))]
public class ProcessItemOrchestration : TaskOrchestrator<WorkItem, string>
{
    public override async Task<string> RunAsync(
        TaskOrchestrationContext context,
        WorkItem input)
    {
        ILogger logger = context.CreateReplaySafeLogger<ProcessItemOrchestration>();
        logger.LogInformation("Processing item {ItemId} for tenant {TenantId}", input.Id, input.TenantId);

        string result = await context.CallActivityAsync<string>(
            nameof(ApplyChangeActivity),
            input);

        return result;
    }
}
