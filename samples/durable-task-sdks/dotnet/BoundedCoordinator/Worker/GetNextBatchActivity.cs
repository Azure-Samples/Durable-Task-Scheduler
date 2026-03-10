using Microsoft.DurableTask;

namespace BoundedCoordinator;

[DurableTask(nameof(GetNextBatchActivity))]
public class GetNextBatchActivity : TaskActivity<GetBatchInput, WorkBatch>
{
    static int callCount;

    public override Task<WorkBatch> RunAsync(
        TaskActivityContext context,
        GetBatchInput input)
    {
        int call = Interlocked.Increment(ref callCount);

        // Simulate producing 3 batches of work, then finishing.
        if (call > 3)
        {
            return Task.FromResult(new WorkBatch(
                Items: [],
                NextCursor: null,
                HasMore: false));
        }

        var items = Enumerable.Range(1, input.MaxItems > 5 ? 5 : input.MaxItems)
            .Select(i => new WorkItem(
                Id: $"item-{call}-{i}",
                TenantId: $"tenant-{i}",
                Payload: $"data-{call}-{i}"))
            .ToList();

        return Task.FromResult(new WorkBatch(
            Items: items,
            NextCursor: $"cursor-{call}",
            HasMore: call < 3));
    }
}
