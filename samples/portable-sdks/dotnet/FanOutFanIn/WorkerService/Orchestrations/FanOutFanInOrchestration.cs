using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;
using WorkerService.Models;
using System.Diagnostics;

namespace WorkerService.Orchestrations;

public class HelloWorld
{
    private readonly ILogger<HelloWorld>? _logger;

    // Support both constructor injection and direct instantiation
    public HelloWorld(ILogger<HelloWorld>? logger = null)
    {
        _logger = logger;
    }

    public async Task<FanOutFanInTestResult> RunAsync(TaskOrchestrationContext context, FanOutFanInOrchestrationInput input)
    {
        var stopwatch = Stopwatch.StartNew();
        var results = new List<ActivityResult>();
        
        _logger?.LogInformation("Orchestration started. Instance: {InstanceId}, Iterations: {Iterations}, ParallelActivities: {ParallelActivities}", 
            context.InstanceId, 
            input.Iterations,
            input.ParallelActivities);
        
        // Run multiple iterations of parallel activities
        for (int i = 0; i < input.Iterations; i++)
        {
            _logger?.LogDebug("Starting iteration {Iteration} of {TotalIterations}", i + 1, input.Iterations);
            
            var tasks = new List<Task<ActivityResult>>();
            
            // Create multiple parallel activities
            for (int j = 0; j < input.ParallelActivities; j++)
            {
                var task = context.CallActivityAsync<ActivityResult>(
                    "SayHello",
                    new ActivityInput 
                    { 
                        IterationNumber = i, 
                        ActivityNumber = j 
                    });
                tasks.Add(task);
                
                _logger?.LogDebug("Activity {ActivityNumber} in iteration {Iteration} scheduled", j + 1, i + 1);
            }
            
            _logger?.LogDebug("Waiting for all {Count} activities in iteration {Iteration} to complete", tasks.Count, i + 1);
            
            // Wait for all parallel activities to complete
            await Task.WhenAll(tasks);
            
            // Collect results
            results.AddRange(tasks.Select(t => t.Result));
            
            _logger?.LogDebug("All activities in iteration {Iteration} completed successfully", i + 1);
        }
        
        stopwatch.Stop();
        
        // Return fan-out/fan-in results
        var finalResult = new FanOutFanInTestResult
        {
            TotalActivities = input.Iterations * input.ParallelActivities,
            ElapsedTimeMs = stopwatch.ElapsedMilliseconds,
            AverageActivityTimeMs = results.Average(r => r.ProcessingTimeMs),
            Results = results
        };
        
        _logger?.LogInformation("Orchestration completed. Instance: {InstanceId}, TotalActivities: {TotalActivities}, ElapsedTime: {ElapsedTimeMs}ms, AvgActivityTime: {AverageActivityTimeMs}ms",
            context.InstanceId,
            finalResult.TotalActivities,
            finalResult.ElapsedTimeMs,
            finalResult.AverageActivityTimeMs);
            
        return finalResult;
    }
}