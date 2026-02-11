using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;

namespace EternalOrchestrations;

[DurableTask(nameof(PerformCleanupActivity))]
public class PerformCleanupActivity : TaskActivity<CleanupInput, CleanupResult>
{
    readonly ILogger<PerformCleanupActivity> logger;

    public PerformCleanupActivity(ILogger<PerformCleanupActivity> logger)
    {
        this.logger = logger;
    }

    public override Task<CleanupResult> RunAsync(TaskActivityContext context, CleanupInput input)
    {
        // Simulate cleanup work
        var random = new Random(input.Iteration);
        int itemsRemoved = random.Next(5, 50);
        long bytesFreed = random.Next(1024, 1024 * 1024);

        this.logger.LogInformation("Performing cleanup (iteration {Iteration}): removed {Items} expired items, freed {Bytes} bytes.",
            input.Iteration, itemsRemoved, bytesFreed);

        return Task.FromResult(new CleanupResult(itemsRemoved, bytesFreed));
    }
}
