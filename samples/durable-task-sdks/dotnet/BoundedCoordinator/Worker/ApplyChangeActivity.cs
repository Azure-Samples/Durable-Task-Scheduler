using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;

namespace BoundedCoordinator;

[DurableTask(nameof(ApplyChangeActivity))]
public class ApplyChangeActivity : TaskActivity<WorkItem, string>
{
    readonly ILogger<ApplyChangeActivity> logger;

    public ApplyChangeActivity(ILoggerFactory loggerFactory)
    {
        this.logger = loggerFactory.CreateLogger<ApplyChangeActivity>();
    }

    public override Task<string> RunAsync(
        TaskActivityContext context,
        WorkItem input)
    {
        this.logger.LogInformation("Applying change for item {ItemId}, tenant {TenantId}",
            input.Id, input.TenantId);

        return Task.FromResult($"processed:{input.Id}");
    }
}
