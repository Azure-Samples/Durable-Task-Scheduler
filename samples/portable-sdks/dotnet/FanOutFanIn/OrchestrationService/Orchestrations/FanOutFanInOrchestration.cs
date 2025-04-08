using Microsoft.DurableTask;
using OrchestrationService.Models;
using System.Diagnostics;

namespace OrchestrationService.Orchestrations;

public class FanOutFanInOrchestration
{
    public async Task<FanOutFanInTestResult> RunAsync(TaskOrchestrationContext context, FanOutFanInOrchestrationInput input)
    {
        var stopwatch = Stopwatch.StartNew();
        var results = new List<ActivityResult>();
        
        // Run multiple iterations of parallel activities
        for (int i = 0; i < input.Iterations; i++)
        {
            var tasks = new List<Task<ActivityResult>>();
            
            // Create multiple parallel activities
            for (int j = 0; j < input.ParallelActivities; j++)
            {
                var task = context.CallActivityAsync<ActivityResult>(
                    "FanOutFanInActivity",
                    new ActivityInput 
                    { 
                        IterationNumber = i, 
                        ActivityNumber = j 
                    });
                tasks.Add(task);
            }
            
            // Wait for all parallel activities to complete
            await Task.WhenAll(tasks);
            
            // Collect results
            results.AddRange(tasks.Select(t => t.Result));
        }
        
        stopwatch.Stop();
        
        // Return fan-out/fan-in results
        return new FanOutFanInTestResult
        {
            TotalActivities = input.Iterations * input.ParallelActivities,
            ElapsedTimeMs = stopwatch.ElapsedMilliseconds,
            AverageActivityTimeMs = results.Average(r => r.ProcessingTimeMs),
            Results = results
        };
    }
}