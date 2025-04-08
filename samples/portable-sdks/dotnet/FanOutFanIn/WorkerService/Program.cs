using Microsoft.DurableTask;
using Microsoft.DurableTask.Worker;
using Microsoft.DurableTask.Worker.AzureManaged;
using Microsoft.Extensions.Logging;
using System.Text.Json.Serialization;
using WorkerService.Activities;
using WorkerService.Models;
using WorkerService.Orchestrations;

var builder = WebApplication.CreateBuilder(args);

// Add services to the container.
builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen();

// Get connection string from environment variable or configuration
string connectionString = builder.Configuration["DURABLE_TASK_SCHEDULER_CONNECTION_STRING"] 
    ?? Environment.GetEnvironmentVariable("DURABLE_TASK_CONNECTION_STRING")
    ?? builder.Configuration["DurableTaskScheduler:ConnectionString"];

if (string.IsNullOrEmpty(connectionString))
{
    connectionString = "UseDevelopmentStorage=true"; // Default to local storage for dev
    builder.Configuration["DurableTaskScheduler:ConnectionString"] = connectionString;
    Console.WriteLine("Using default development storage connection string");
}

// Register SayHello and HelloWorld for DI 
builder.Services.AddTransient<SayHello>();
builder.Services.AddTransient<HelloWorld>();

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
    
    // Set specific categories to more verbose logging
    logging.AddFilter("Microsoft.DurableTask", LogLevel.Debug);
    logging.AddFilter("WorkerService", LogLevel.Debug);
});

// Create a logger factory for initial configuration
var loggerFactory = LoggerFactory.Create(logging => {
    logging.AddConsole();
    logging.SetMinimumLevel(LogLevel.Information);
});
var logger = loggerFactory.CreateLogger<Program>();
logger.LogInformation("Configuring Durable Task Worker with connection: {ConnectionString}", 
    connectionString.StartsWith("UseDevelopment") ? "UseDevelopmentStorage=true" : "[redacted]");

// Add Durable Task Worker with the standard registration pattern
builder.Services.AddDurableTaskWorker(workerBuilder =>
{
    // Configure the worker to use the Durable Task Scheduler backend
    workerBuilder.UseDurableTaskScheduler(connectionString);
    
    // Register all tasks (orchestrations and activities)
    workerBuilder.AddTasks(registry =>
    {
        // Register the HelloWorld orchestration
        logger.LogInformation("Registering HelloWorld");
        registry.AddOrchestratorFunc<HelloWorldInput, HelloWorldResult>(
            "HelloWorld", 
            async (ctx, input) =>
            {
                // Create a properly typed logger for the orchestration
                var orchestrationLogger = loggerFactory.CreateLogger<HelloWorld>();
                var orchestration = new HelloWorld(orchestrationLogger);
                logger.LogDebug("Orchestration execution starting. Instance: {InstanceId}", ctx.InstanceId);
                var result = await orchestration.RunAsync(ctx, input);
                logger.LogDebug("Orchestration execution completed. Instance: {InstanceId}", ctx.InstanceId);
                return result;
            });

        logger.LogInformation("Registering SayHello");
        
        // Register the SayHello activity
        registry.AddActivityFunc<ActivityInput, ActivityResult>(
            "SayHello",
            async (ctx, input) =>
            {
                try
                {
                    logger.LogDebug("Activity function invoked. InstanceId: {InstanceId}, ActivityNumber: {ActivityNumber}, IterationNumber: {IterationNumber}", 
                        ctx.InstanceId, 
                        input.ActivityNumber,
                        input.IterationNumber);
                        
                    // Create a properly typed logger for the activity
                    var activityLogger = loggerFactory.CreateLogger<SayHello>();
                    
                    // Create activity instance with the correct logger type
                    var activity = new SayHello(activityLogger);
                    
                    // Capture the execution in a try-catch for better diagnostics
                    try
                    {
                        var result = await activity.RunAsync(ctx, input);
                        logger.LogDebug("Activity function completed successfully. InstanceId: {InstanceId}, ActivityNumber: {ActivityNumber}", 
                            ctx.InstanceId, 
                            input.ActivityNumber);
                        return result;
                    }
                    catch (Exception activityEx)
                    {
                        logger.LogError(activityEx, "Activity execution failed. InstanceId: {InstanceId}, ActivityNumber: {ActivityNumber}, Error: {ErrorMessage}", 
                            ctx.InstanceId, 
                            input.ActivityNumber, 
                            activityEx.Message);
                        throw;
                    }
                }
                catch (Exception ex)
                {
                    logger.LogError(ex, "Critical error in activity function wrapper. InstanceId: {InstanceId}, Error: {ErrorMessage}", 
                        ctx.InstanceId, 
                        ex.Message);
                    throw;
                }
            });
    });
});

// Configure JSON options for any controllers
builder.Services.AddControllers()
    .AddJsonOptions(options =>
    {
        options.JsonSerializerOptions.Converters.Add(new JsonStringEnumConverter());
        options.JsonSerializerOptions.DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull;
    });

var app = builder.Build();

// Log application startup
var serviceLogger = app.Services.GetRequiredService<ILogger<Program>>();
serviceLogger.LogInformation("Worker Service starting up at {Time}", DateTime.UtcNow);

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

// Add a health check endpoint for the worker
app.MapGet("/health", (ILogger<Program> endpointLogger) => 
{
    endpointLogger.LogInformation("Health check requested");
    return Results.Ok(new { 
        Status = "Healthy",
        Service = "Worker Service",
        Time = DateTime.UtcNow
    });
})
.WithName("HealthCheck")
.WithOpenApi();

// Endpoint to check service status
app.MapGet("/status", (ILogger<Program> endpointLogger) =>
{
    endpointLogger.LogInformation("Status check requested");
    return Results.Ok(new { 
        Status = "Running",
        Service = "Worker Service",
        Time = DateTime.UtcNow 
    });
});

// Log that the app is about to start running
serviceLogger.LogInformation("Worker Service initialized and ready to process activities");

app.Run();
