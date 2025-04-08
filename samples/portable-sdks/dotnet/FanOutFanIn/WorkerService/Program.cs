using Microsoft.DurableTask;
using Microsoft.DurableTask.Worker;
using Microsoft.DurableTask.Worker.AzureManaged;
using WorkerService.Models;
using WorkerService.Activities;

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
    
    // Register activity only (orchestration moved to OrchestrationService)
    workerBuilder.AddTasks(registry =>
    {            
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
