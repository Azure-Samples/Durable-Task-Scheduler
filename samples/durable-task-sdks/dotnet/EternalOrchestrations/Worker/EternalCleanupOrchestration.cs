using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;

namespace EternalOrchestrations;

[DurableTask(nameof(EternalCleanupOrchestration))]
public class EternalCleanupOrchestration : TaskOrchestrator<EternalState, string>
{
    public override async Task<string> RunAsync(TaskOrchestrationContext context, EternalState state)
    {
        ILogger logger = context.CreateReplaySafeLogger<EternalCleanupOrchestration>();

        int iteration = state.Iteration + 1;
        int intervalSeconds = state.IntervalSeconds > 0 ? state.IntervalSeconds : 10;

        logger.LogInformation("=== Eternal orchestration iteration {Iteration} ===", iteration);

        // Perform the periodic work
        var result = await context.CallActivityAsync<CleanupResult>(
            nameof(PerformCleanupActivity),
            new CleanupInput(iteration));

        logger.LogInformation("Cleanup iteration {Iteration}: removed {Count} items, {Size} bytes freed.",
            iteration, result.ItemsRemoved, result.BytesFreed);

        // Wait before next iteration
        logger.LogInformation("Sleeping for {Interval} seconds before next iteration...", intervalSeconds);
        await context.CreateTimer(TimeSpan.FromSeconds(intervalSeconds), CancellationToken.None);

        // Restart with a clean history — this is what makes it "eternal"
        context.ContinueAsNew(new EternalState(iteration, intervalSeconds));

        return string.Empty; // Won't be reached
    }
}

public record EternalState(int Iteration = 0, int IntervalSeconds = 10);
public record CleanupInput(int Iteration);
public record CleanupResult(int ItemsRemoved, long BytesFreed);
