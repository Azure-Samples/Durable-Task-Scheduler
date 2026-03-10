using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;

namespace BoundedCoordinator;

/// <summary>
/// Demonstrates a bounded coordinator that fans out child work, waits for all
/// children to complete, then resets via ContinueAsNew. This prevents unbounded
/// history growth that can occur with long-lived message-pump orchestrations.
/// </summary>
[DurableTask(nameof(CoordinatorOrchestration))]
public class CoordinatorOrchestration : TaskOrchestrator<CoordinatorState?, CoordinatorResult>
{
    public override async Task<CoordinatorResult> RunAsync(
        TaskOrchestrationContext context,
        CoordinatorState? input)
    {
        ILogger logger = context.CreateReplaySafeLogger<CoordinatorOrchestration>();

        CoordinatorState state = input ?? new CoordinatorState(Cursor: null, BatchNumber: 0);

        logger.LogInformation("Coordinator batch {BatchNumber}, cursor={Cursor}",
            state.BatchNumber, state.Cursor ?? "(start)");

        // Step 1: Read a bounded batch of work items.
        WorkBatch batch = await context.CallActivityAsync<WorkBatch>(
            nameof(GetNextBatchActivity),
            new GetBatchInput(state.Cursor, MaxItems: 50));

        if (batch.Items.Count == 0)
        {
            logger.LogInformation("No more items to process. Coordinator completing.");
            return new CoordinatorResult(state.BatchNumber, Completed: true);
        }

        // Step 2: Fan out child orchestrations for this batch.
        logger.LogInformation("Processing batch of {Count} items", batch.Items.Count);

        var childTasks = new Task[batch.Items.Count];
        for (int i = 0; i < batch.Items.Count; i++)
        {
            childTasks[i] = context.CallSubOrchestratorAsync<string>(
                nameof(ProcessItemOrchestration),
                batch.Items[i]);
        }

        // Step 3: Wait for ALL children to complete before resetting.
        // This is critical — never call ContinueAsNew while children are still running.
        await Task.WhenAll(childTasks);

        logger.LogInformation("Batch {BatchNumber} complete. {Count} items processed.",
            state.BatchNumber + 1, batch.Items.Count);

        // Step 4: ContinueAsNew with compact carry-forward state.
        // This resets the orchestration history, preventing unbounded growth.
        if (batch.HasMore)
        {
            var nextState = new CoordinatorState(
                Cursor: batch.NextCursor,
                BatchNumber: state.BatchNumber + 1);

            context.ContinueAsNew(nextState);
            return default!; // unreachable after ContinueAsNew
        }

        return new CoordinatorResult(state.BatchNumber + 1, Completed: true);
    }
}

public record CoordinatorState(string? Cursor, int BatchNumber);
public record CoordinatorResult(int TotalBatches, bool Completed);
public record GetBatchInput(string? Cursor, int MaxItems);
public record WorkItem(string Id, string TenantId, string Payload);
public record WorkBatch(IReadOnlyList<WorkItem> Items, string? NextCursor, bool HasMore);
