using Microsoft.DurableTask;

namespace BoundedCoordinator;

[DurableTask(nameof(GetNextBatchActivity))]
public class GetNextBatchActivity : TaskActivity<GetBatchInput, WorkBatch>
{
    const int TotalBatches = 3;

    public override Task<WorkBatch> RunAsync(
        TaskActivityContext context,
        GetBatchInput input)
    {
        // Derive the batch number deterministically from the cursor
        // so the activity is stateless and safe across retries/scale-out.
        int batchNumber = input.Cursor is null
            ? 1
            : int.Parse(input.Cursor.Split('-').Last()) + 1;

        if (batchNumber > TotalBatches)
        {
            return Task.FromResult(new WorkBatch(
                Items: [],
                NextCursor: null,
                HasMore: false));
        }

        var items = Enumerable.Range(1, Math.Min(5, input.MaxItems))
            .Select(i => new WorkItem(
                Id: $"item-{batchNumber}-{i}",
                TenantId: $"tenant-{i}",
                Payload: $"data-{batchNumber}-{i}"))
            .ToList();

        return Task.FromResult(new WorkBatch(
            Items: items,
            NextCursor: $"cursor-{batchNumber}",
            HasMore: batchNumber < TotalBatches));
    }
}
