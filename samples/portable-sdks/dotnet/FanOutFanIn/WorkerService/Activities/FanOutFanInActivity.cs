using Microsoft.DurableTask;
using System.Diagnostics;
using WorkerService.Models;

namespace WorkerService.Activities;

public class FanOutFanInActivity
{
    public Task<ActivityResult> RunAsync(TaskActivityContext context, ActivityInput input)
    {
        var stopwatch = Stopwatch.StartNew();
        
        string output = "Hello World";
        Console.WriteLine(output);
        
        stopwatch.Stop();
        
        var result = new ActivityResult
        {
            IterationNumber = input.IterationNumber,
            ActivityNumber = input.ActivityNumber,
            ProcessingTimeMs = stopwatch.ElapsedMilliseconds,
            Output = output
        };
        
        return Task.FromResult(result);
    }
}