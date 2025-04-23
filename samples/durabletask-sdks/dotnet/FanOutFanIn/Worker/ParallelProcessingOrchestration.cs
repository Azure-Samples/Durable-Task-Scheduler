using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;
using System.Collections.Generic;
using System.Threading.Tasks;

namespace FanOutFanIn;

[DurableTask]
public class ParallelProcessingOrchestration : TaskOrchestrator<List<string>, Dictionary<string, int>>
{
    public override async Task<Dictionary<string, int>> RunAsync(TaskOrchestrationContext context, List<string> workItems)
    {
        // Step 1: Fan-out by creating a task for each work item in parallel
        List<Task<Dictionary<string, int>>> processingTasks = new List<Task<Dictionary<string, int>>>();
        
        foreach (string workItem in workItems)
        {
            // Create a task for each work item (fan-out)
            Task<Dictionary<string, int>> task = context.CallActivityAsync<Dictionary<string, int>>(
                nameof(ProcessWorkItemActivity), workItem);
            processingTasks.Add(task);
        }
        
        // Step 2: Wait for all parallel tasks to complete
        Dictionary<string, int>[] results = await Task.WhenAll(processingTasks);
        
        // Step 3: Fan-in by aggregating all results
        Dictionary<string, int> aggregatedResults = await context.CallActivityAsync<Dictionary<string, int>>(
            nameof(AggregateResultsActivity), results);
        
        return aggregatedResults;
    }
}

[DurableTask]
public class ProcessWorkItemActivity : TaskActivity<string, Dictionary<string, int>>
{
    private readonly ILogger<ProcessWorkItemActivity> _logger;

    public ProcessWorkItemActivity(ILogger<ProcessWorkItemActivity> logger)
    {
        _logger = logger;
    }

    public override Task<Dictionary<string, int>> RunAsync(TaskActivityContext context, string workItem)
    {
        _logger.LogInformation("Processing work item: {WorkItem}", workItem);
        
        // Simulate some work on the item
        // This would be where you do the actual processing for each item
        Dictionary<string, int> result = new Dictionary<string, int>
        {
            { workItem, workItem.Length } // Simple example: Count characters in the work item
        };
        
        _logger.LogInformation("Completed processing work item: {WorkItem} with result: {Result}", 
            workItem, result[workItem]);
        
        return Task.FromResult(result);
    }
}

[DurableTask]
public class AggregateResultsActivity : TaskActivity<Dictionary<string, int>[], Dictionary<string, int>>
{
    private readonly ILogger<AggregateResultsActivity> _logger;

    public AggregateResultsActivity(ILogger<AggregateResultsActivity> logger)
    {
        _logger = logger;
    }

    public override Task<Dictionary<string, int>> RunAsync(TaskActivityContext context, Dictionary<string, int>[] results)
    {
        _logger.LogInformation("Aggregating {Count} results", results.Length);
        
        // Combine all the individual results into one aggregated result
        Dictionary<string, int> aggregatedResult = new Dictionary<string, int>();
        
        foreach (Dictionary<string, int> result in results)
        {
            foreach (KeyValuePair<string, int> kvp in result)
            {
                // Merge each key-value pair into the aggregated result
                aggregatedResult[kvp.Key] = kvp.Value;
            }
        }
        
        _logger.LogInformation("Aggregated {Count} work items into final result", aggregatedResult.Count);
        
        return Task.FromResult(aggregatedResult);
    }
}
