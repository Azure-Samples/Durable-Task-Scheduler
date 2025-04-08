using Microsoft.AspNetCore.Mvc;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Client.AzureManaged;
using Microsoft.Extensions.Logging;
using System.Text.Json.Serialization;
using ClientService.Models;

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

// Configure enhanced logging
builder.Services.AddLogging(logging =>
{
    logging.ClearProviders();
    logging.AddConsole(options =>
    {
        options.IncludeScopes = true;
        options.TimestampFormat = "yyyy-MM-dd HH:mm:ss.fff ";
    });
    
    // Set minimum level to Information by default
    logging.SetMinimumLevel(LogLevel.Information);
    
    // Set specific categories to more verbose logging to help with troubleshooting
    logging.AddFilter("Microsoft.DurableTask", LogLevel.Debug);
    logging.AddFilter("ClientService", LogLevel.Debug);
});

// Create a logger factory for initial configuration
var loggerFactory = LoggerFactory.Create(logging => {
    logging.AddConsole();
    logging.SetMinimumLevel(LogLevel.Information);
});
var logger = loggerFactory.CreateLogger<Program>();
logger.LogInformation("Configuring Durable Task Client with connection: {ConnectionString}", 
    connectionString.StartsWith("UseDevelopment") ? "UseDevelopmentStorage=true" : "[redacted]");

// Register the client with a simpler registration pattern
builder.Services.AddDurableTaskClient(clientBuilder =>
{
    clientBuilder.UseDurableTaskScheduler(connectionString);
});

// Add controllers with JSON options
builder.Services.AddControllers()
    .AddJsonOptions(options =>
    {
        options.JsonSerializerOptions.Converters.Add(new JsonStringEnumConverter());
        options.JsonSerializerOptions.DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull;
    });

var app = builder.Build();

// Log application startup
var serviceLogger = app.Services.GetRequiredService<ILogger<Program>>();
serviceLogger.LogInformation("Client Service starting up at {Time}", DateTime.UtcNow);

// Configure the HTTP request pipeline.
if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI();
    serviceLogger.LogInformation("Development environment detected, Swagger enabled");
}

app.UseHttpsRedirection();
app.UseRouting();
app.MapControllers();

// Sample endpoint to trigger an orchestration
app.MapPost("/api/orchestrations", async (
    [FromServices] DurableTaskClient client,
    [FromServices] ILogger<Program> endpointLogger,
    [FromBody] FanOutFanInRequest request) =>
{
    endpointLogger.LogInformation("Starting {Count} orchestration(s) with {Iterations} iterations and {ParallelActivities} parallel activities per iteration", 
        request.ParallelOrchestrations, 
        request.Iterations, 
        request.ParallelActivities);
        
    var instanceIds = new List<string>();
    var tasks = new List<Task<string>>();
    
    try
    {
        // Schedule multiple orchestrations in parallel
        for (int i = 0; i < request.ParallelOrchestrations; i++)
        {
            endpointLogger.LogDebug("Scheduling orchestration {OrchestratorNumber} of {TotalOrchestrators}", i+1, request.ParallelOrchestrations);
            
            tasks.Add(client.ScheduleNewOrchestrationInstanceAsync(
                "HelloWorld", 
                new FanOutFanInOrchestrationInput
                {
                    Iterations = request.Iterations,
                    ParallelActivities = request.ParallelActivities
                }));
        }
        
        // Wait for all orchestrations to be scheduled
        instanceIds = (await Task.WhenAll(tasks)).ToList();
        
        endpointLogger.LogInformation("Successfully scheduled {Count} orchestration(s)", instanceIds.Count);
        
        return Results.Accepted($"/api/orchestrations/status", new { 
            count = instanceIds.Count,
            instanceIds = instanceIds 
        });
    }
    catch (Exception ex)
    {
        endpointLogger.LogError(ex, "Failed to schedule orchestrations: {ErrorMessage}", ex.Message);
        return Results.Problem("Failed to schedule orchestrations: " + ex.Message);
    }
})
.WithName("StartHelloWorldOrchestration")
.WithOpenApi();

// Endpoint to check orchestration status
app.MapGet("/api/orchestrations/{instanceId}", async (
    [FromServices] DurableTaskClient client,
    [FromServices] ILogger<Program> endpointLogger,
    string instanceId) =>
{
    endpointLogger.LogInformation("Checking status for orchestration: {InstanceId}", instanceId);
    
    try
    {
        var instance = await client.GetInstanceAsync(instanceId);
        if (instance == null)
        {
            endpointLogger.LogWarning("Orchestration instance not found: {InstanceId}", instanceId);
            return Results.NotFound();
        }

        endpointLogger.LogDebug("Orchestration status: {InstanceId} is {Status}", 
            instanceId, 
            instance.RuntimeStatus);
            
        return Results.Ok(new
        {
            Id = instance.InstanceId,
            Status = instance.RuntimeStatus.ToString(),
            Created = instance.CreatedAt,
            LastUpdated = instance.LastUpdatedAt,
            Output = instance.SerializedOutput
        });
    }
    catch (Exception ex)
    {
        endpointLogger.LogError(ex, "Error retrieving orchestration status for {InstanceId}: {ErrorMessage}", 
            instanceId, 
            ex.Message);
        return Results.Problem("Error retrieving orchestration status: " + ex.Message);
    }
})
.WithName("GetOrchestrationStatus")
.WithOpenApi();

// Endpoint to check multiple orchestration statuses
app.MapPost("/api/orchestrations/status", async (
    [FromServices] DurableTaskClient client,
    [FromServices] ILogger<Program> endpointLogger,
    [FromBody] List<string> instanceIds) =>
{
    endpointLogger.LogInformation("Checking status for {Count} orchestration(s)", instanceIds.Count);
    
    try
    {
        var tasks = instanceIds.Select(id => client.GetInstanceAsync(id)).ToArray();
        var instances = await Task.WhenAll(tasks);
        
        var results = instances
            .Where(instance => instance != null)
            .Select(instance => new
            {
                Id = instance!.InstanceId,
                Status = instance.RuntimeStatus.ToString(),
                Created = instance.CreatedAt,
                LastUpdated = instance.LastUpdatedAt,
                Output = instance.SerializedOutput
            })
            .ToList();
            
        endpointLogger.LogInformation("Found {FoundCount} of {RequestedCount} requested orchestration(s)", 
            results.Count, 
            instanceIds.Count);
            
        return Results.Ok(new
        {
            TotalRequested = instanceIds.Count,
            FoundInstances = results.Count,
            Instances = results
        });
    }
    catch (Exception ex)
    {
        endpointLogger.LogError(ex, "Error retrieving multiple orchestration statuses: {ErrorMessage}", ex.Message);
        return Results.Problem("Error retrieving orchestration statuses: " + ex.Message);
    }
})
.WithName("GetMultipleOrchestrationStatus")
.WithOpenApi();

// Endpoint to check service status
app.MapGet("/status", (ILogger<Program> endpointLogger) =>
{
    endpointLogger.LogInformation("Status check requested");
    return Results.Ok(new { 
        Status = "Running",
        Service = "Client Service",
        Time = DateTime.UtcNow 
    });
});

// Log that the app is about to start running
serviceLogger.LogInformation("Client Service initialized and ready to process orchestrations");

app.Run();
