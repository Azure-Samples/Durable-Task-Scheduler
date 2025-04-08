using Microsoft.DurableTask;
using System.Diagnostics;
using WorkerService.Models;
using Microsoft.Extensions.Logging;

namespace WorkerService.Activities;

public class SayHello
{
    private readonly ILogger<SayHello>? _logger;

    // Support both constructor injection and direct instantiation
    public SayHello(ILogger<SayHello>? logger = null)
    {
        _logger = logger;
    }

    public Task<ActivityResult> RunAsync(TaskActivityContext context, ActivityInput input)
    {
        _logger?.LogInformation("Starting activity execution. Iteration: {IterationNumber}, Activity: {ActivityNumber}, Instance: {InstanceId}", 
            input.IterationNumber, 
            input.ActivityNumber,
            context.InstanceId);
            
        var stopwatch = Stopwatch.StartNew();
        
        try
        {
            string output = "Hello World";
            _logger?.LogInformation("Activity processing complete. Output: {Output}", output);
            
            stopwatch.Stop();
            
            var result = new ActivityResult
            {
                IterationNumber = input.IterationNumber,
                ActivityNumber = input.ActivityNumber,
                ProcessingTimeMs = stopwatch.ElapsedMilliseconds,
                Output = output
            };
            
            _logger?.LogInformation("Activity completed successfully. Iteration: {Iteration}, Activity: {ActivityNumber}, Time: {ProcessingTime}ms", 
                input.IterationNumber, 
                input.ActivityNumber, 
                stopwatch.ElapsedMilliseconds);
                
            return Task.FromResult(result);
        }
        catch (Exception ex)
        {
            stopwatch.Stop();
            _logger?.LogError(ex, "Activity failed. Iteration: {Iteration}, Activity: {ActivityNumber}, Error: {ErrorMessage}", 
                input.IterationNumber, 
                input.ActivityNumber, 
                ex.Message);
            throw; // Re-throw to let the Durable Task Framework handle it
        }
    }
}