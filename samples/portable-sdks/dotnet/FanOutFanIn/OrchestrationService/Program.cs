using Microsoft.AspNetCore.Mvc;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Client.AzureManaged;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Worker;
using Microsoft.DurableTask.Worker.AzureManaged;
using System.Text.Json.Serialization;
using OrchestrationService.Models;
using OrchestrationService.Orchestrations;

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
    builder.Configuration["DurableTaskScheduler:ConnectionString"] = connectionString;
}

// Add Durable Task Client with Azure Managed backend
builder.Services.AddDurableTaskClient("FanOutFanInClient", options =>
{
    options.UseDurableTaskScheduler(connectionString);
});

// Add Durable Task Worker for orchestrations (moved from WorkerService)
builder.Services.AddDurableTaskWorker(workerBuilder =>
{
    // Configure the worker to use the Durable Task Scheduler backend
    workerBuilder.UseDurableTaskScheduler(connectionString);
    
    // Register orchestrations
    workerBuilder.AddTasks(registry =>
    {
        registry.AddOrchestratorFunc<FanOutFanInOrchestrationInput, FanOutFanInTestResult>("FanOutFanInOrchestration", 
            async (ctx, input) =>
            {
                var orchestration = new FanOutFanInOrchestration();
                return await orchestration.RunAsync(ctx, input);
            });
    });
});

// Add controllers with JSON options
builder.Services.AddControllers()
    .AddJsonOptions(options =>
    {
        options.JsonSerializerOptions.Converters.Add(new JsonStringEnumConverter());
    });

var app = builder.Build();

// Configure the HTTP request pipeline.
if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI();
}

app.UseHttpsRedirection();
app.UseRouting();
app.MapControllers();

// Sample endpoint to trigger an orchestration
app.MapPost("/api/orchestrations", async (
    [FromServices] IDurableTaskClientProvider clientProvider,
    [FromBody] FanOutFanInRequest request) =>
{
    // Get the named client using the provider
    var client = clientProvider.GetClient("FanOutFanInClient");
    
    var instanceIds = new List<string>();
    var tasks = new List<Task<string>>();
    
    // Schedule multiple orchestrations in parallel
    for (int i = 0; i < request.ParallelOrchestrations; i++)
    {
        tasks.Add(client.ScheduleNewOrchestrationInstanceAsync(
            "FanOutFanInOrchestration", 
            new FanOutFanInOrchestrationInput
            {
                Iterations = request.Iterations,
                ParallelActivities = request.ParallelActivities
            }));
    }
    
    // Wait for all orchestrations to be scheduled
    instanceIds = (await Task.WhenAll(tasks)).ToList();
    
    return Results.Accepted($"/api/orchestrations/status", new { 
        count = instanceIds.Count,
        instanceIds = instanceIds 
    });
})
.WithName("StartFanOutFanInOrchestration")
.WithOpenApi();

// Endpoint to check orchestration status
app.MapGet("/api/orchestrations/{instanceId}", async (
    [FromServices] IDurableTaskClientProvider clientProvider,
    string instanceId) =>
{
    // Get the named client using the provider
    var client = clientProvider.GetClient("FanOutFanInClient");
    
    var instance = await client.GetInstanceAsync(instanceId);
    if (instance == null)
    {
        return Results.NotFound();
    }

    return Results.Ok(new
    {
        Id = instance.InstanceId,
        Status = instance.RuntimeStatus.ToString(),
        Created = instance.CreatedAt,
        LastUpdated = instance.LastUpdatedAt,
        Output = instance.SerializedOutput
    });
})
.WithName("GetOrchestrationStatus")
.WithOpenApi();

// Endpoint to check multiple orchestration statuses
app.MapPost("/api/orchestrations/status", async (
    [FromServices] IDurableTaskClientProvider clientProvider,
    [FromBody] List<string> instanceIds) =>
{
    // Get the named client using the provider
    var client = clientProvider.GetClient("FanOutFanInClient");
    
    var tasks = instanceIds.Select(id => client.GetInstanceAsync(id)).ToArray();
    var instances = await Task.WhenAll(tasks);
    
    var results = instances
        .Where(instance => instance != null)  // Filter out null instances first
        .Select(instance => new
        {
            Id = instance!.InstanceId,        // Use null-forgiving operator since we filtered nulls
            Status = instance.RuntimeStatus.ToString(),
            Created = instance.CreatedAt,
            LastUpdated = instance.LastUpdatedAt,
            Output = instance.SerializedOutput
        })
        .ToList();
        
    return Results.Ok(new
    {
        TotalRequested = instanceIds.Count,
        FoundInstances = results.Count,
        Instances = results
    });
})
.WithName("GetMultipleOrchestrationStatus")
.WithOpenApi();

// Endpoint to check service status
app.MapGet("/status", () =>
{
    return Results.Ok(new { 
        Status = "Running",
        Service = "Orchestration Service",
        Time = DateTime.UtcNow 
    });
});

app.Run();
