using Microsoft.DurableTask;
using Microsoft.DurableTask.Worker;
using Microsoft.DurableTask.Worker.AzureManaged;
using System.Diagnostics;

var builder = WebApplication.CreateBuilder(args);

// Add services to the container.
builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen();

// Get connection string from environment variable or configuration
var connectionString = Environment.GetEnvironmentVariable("DURABLE_TASK_CONNECTION_STRING") 
    ?? builder.Configuration["DurableTaskScheduler:ConnectionString"];
if (string.IsNullOrEmpty(connectionString))
{
    connectionString = "UseDevelopmentStorage=true"; // Default to local storage for dev
}

// Add Durable Task Worker
builder.Services.AddDurableTaskWorker(workerBuilder =>
{
    // Configure the worker to use the Durable Task Scheduler backend
    workerBuilder.UseDurableTaskScheduler(connectionString);
    
    // Register orchestrations and activities
    workerBuilder.AddTasks(registry =>
    {
        registry.AddOrchestratorFunc<FanOutFanInOrchestrationInput, FanOutFanInTestResult>("FanOutFanInOrchestration", 
            async (ctx, input) =>
            {
                var orchestration = new FanOutFanInOrchestration();
                return await orchestration.RunAsync(ctx, input);
            });
            
        registry.AddActivityFunc<ActivityInput, ActivityResult>("FanOutFanInActivity", 
            async (ctx, input) =>
            {
                var activity = new FanOutFanInActivity();
                return await activity.RunAsync(ctx, input);
            });
    });
});

var app = builder.Build();

// Configure the HTTP request pipeline.
if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI();
}

app.UseHttpsRedirection();

// Add a health check endpoint for the worker
app.MapGet("/health", () => Results.Ok(new { Status = "Healthy" }))
    .WithName("HealthCheck")
    .WithOpenApi();

// Endpoint to check service status
app.MapGet("/status", () =>
{
    return Results.Ok(new { 
        Status = "Running",
        Service = "Worker Service",
        Time = DateTime.UtcNow 
    });
});

app.Run();

// Orchestration for fan-out/fan-in pattern testing with Durable Task
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

// Activity for fan-out/fan-in testing
public class FanOutFanInActivity
{
    public Task<ActivityResult> RunAsync(TaskActivityContext context, ActivityInput input)
    {
        var stopwatch = Stopwatch.StartNew();
        
        // Output "Hello World" instead of delaying
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

// Input and output classes

// Same as defined in the orchestration service
public class FanOutFanInOrchestrationInput
{
    public int Iterations { get; set; }
    public int ParallelActivities { get; set; }
}

public class ActivityInput
{
    public int IterationNumber { get; set; }
    public int ActivityNumber { get; set; }
}

public class ActivityResult
{
    public int IterationNumber { get; set; }
    public int ActivityNumber { get; set; }
    public long ProcessingTimeMs { get; set; }
    public string Output { get; set; } = string.Empty;
}

public class FanOutFanInTestResult
{
    public int TotalActivities { get; set; }
    public long ElapsedTimeMs { get; set; }
    public double AverageActivityTimeMs { get; set; }
    public List<ActivityResult> Results { get; set; } = new List<ActivityResult>();
}
