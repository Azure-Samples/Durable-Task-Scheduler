using Azure.Identity;
using Microsoft.AspNetCore.Mvc;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Client.AzureManaged;
using AgentChainingSample.Client.Models;
using System.Text.Json;

var builder = WebApplication.CreateBuilder(args);

// Add services to the container
builder.Services.AddEndpointsApiExplorer();

// Get connection string from configuration with fallback to default local emulator connection
string connectionString = builder.Configuration["ENDPOINT"] ??
                         builder.Configuration["DTS_CONNECTION_STRING"] ?? 
                         "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";

// If we have the endpoint but not a full connection string, construct it
if (connectionString.StartsWith("Endpoint=") && !connectionString.Contains("TaskHub="))
{
    string taskHub = builder.Configuration["TASKHUB"] ?? builder.Configuration["TASKHUB_NAME"] ?? "default";
    string clientId = builder.Configuration["AZURE_MANAGED_IDENTITY_CLIENT_ID"] ?? "";
    
    if (!string.IsNullOrEmpty(clientId))
    {
        connectionString = $"{connectionString};Authentication=ManagedIdentity;ClientID={clientId};TaskHub={taskHub}";
    }
    else
    {
        connectionString = $"{connectionString};TaskHub={taskHub}";
    }
}

// Determine if we're connecting to the local emulator
bool isLocalEmulator = connectionString.Contains("localhost");

// Enable HTTP/2 cleartext support for emulator
if (isLocalEmulator)
{
    AppContext.SetSwitch("System.Net.Http.SocketsHttpHandler.Http2UnencryptedSupport", true);
    builder.Services.AddLogging(logging => logging.AddConsole().SetMinimumLevel(LogLevel.Information));
    builder.Services.AddDurableTaskClient(options =>
    {
        options.UseDurableTaskScheduler(connectionString);
    });
}
else
{
    builder.Services.AddLogging(logging => logging.AddConsole().SetMinimumLevel(LogLevel.Information));
    builder.Services.AddDurableTaskClient(options =>
    {
        options.UseDurableTaskScheduler(connectionString);
    });
}

var app = builder.Build();
var logger = app.Services.GetRequiredService<ILogger<Program>>();

logger.LogInformation("Starting Agent Chaining Sample - Content Creation Client");
// Log the connection string with sensitive info redacted
string managedIdentityClientId = builder.Configuration["AZURE_MANAGED_IDENTITY_CLIENT_ID"] ?? "";
string logConnectionString = !string.IsNullOrEmpty(managedIdentityClientId) ? 
    connectionString.Replace(managedIdentityClientId, "[REDACTED]") : 
    connectionString;
logger.LogInformation("Connection string: {ConnectionString}", logConnectionString);
logger.LogInformation("TaskHub: {TaskHub}", builder.Configuration["TASKHUB"] ?? builder.Configuration["TASKHUB_NAME"] ?? "default");
logger.LogInformation("Environment Variables:");
logger.LogInformation("  TASKHUB: {Value}", builder.Configuration["TASKHUB"]);
logger.LogInformation("  TASKHUB_NAME: {Value}", builder.Configuration["TASKHUB_NAME"]);
logger.LogInformation("  ENDPOINT: {Value}", builder.Configuration["ENDPOINT"]);
logger.LogInformation("  DTS_CONNECTION_STRING: {Value}", builder.Configuration["DTS_CONNECTION_STRING"]);
logger.LogInformation("  DTS_URL: {Value}", builder.Configuration["DTS_URL"]);
logger.LogInformation("This sample implements a news article generator workflow with multiple specialized agents");

// Configure the HTTP request pipeline
app.UseHttpsRedirection();

// Define routes
app.MapGet("/", () => 
{
    return "Agent Chaining Content Generator API - Use /api/content to create content";
});

// Add a health check endpoint
app.MapGet("/health", () => Results.Ok("Healthy"));

// Get status of an orchestration
app.MapGet("/api/content/{instanceId}", async (string instanceId, [FromServices] DurableTaskClient client) => 
{
    try
    {
        var metadata = await client.GetInstanceAsync(instanceId, getInputsAndOutputs: true);
        if (metadata == null)
        {
            return Results.NotFound($"No orchestration found with ID: {instanceId}");
        }

        // If the orchestration is complete, return the result
        if (metadata.RuntimeStatus == OrchestrationRuntimeStatus.Completed && metadata.ReadOutputAs<ContentWorkflowResult>() is ContentWorkflowResult result)
        {
            return Results.Ok(result);
        }
        
        // Otherwise return status
        return Results.Ok(new
        {
            InstanceId = metadata.InstanceId,
            CreatedAt = metadata.CreatedAt,
            LastUpdatedAt = metadata.LastUpdatedAt,
            Status = metadata.RuntimeStatus.ToString()
        });
    }
    catch (Exception ex)
    {
        logger.LogError(ex, "Error getting orchestration status");
        return Results.Problem("Error retrieving orchestration status");
    }
});

// Create a new content generation request
app.MapPost("/api/content", async ([FromBody] ContentCreationRequest request, [FromServices] DurableTaskClient client) => 
{
    try
    {
        if (string.IsNullOrEmpty(request.Topic))
        {
            return Results.BadRequest("Topic is required");
        }

        // Set request ID if not provided
        request.RequestId ??= Guid.NewGuid().ToString();
        
        // Start the orchestration
        string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
            "ContentCreationOrchestration", 
            request);
        
        logger.LogInformation("Started orchestration with ID: {InstanceId} for topic: {Topic}", instanceId, request.Topic);
        
        return Results.Accepted($"/api/content/{instanceId}", new
        {
            InstanceId = instanceId,
            Topic = request.Topic,
            Status = "Accepted",
            StatusQueryGetUri = $"/api/content/{instanceId}"
        });
    }
    catch (Exception ex)
    {
        logger.LogError(ex, "Error starting orchestration");
        return Results.Problem("Error starting content generation process");
    }
});

// Get all active orchestrations
app.MapGet("/api/content", async ([FromServices] DurableTaskClient client) => 
{
    try
    {
        var query = new OrchestrationQuery
        {
            PageSize = 100,
            // Get only running and pending orchestrations using the correct property
            // Check latest API documentation for property name
            FetchInputsAndOutputs = false
        };

        var resultList = new List<object>();
        await foreach (var instance in client.GetAllInstancesAsync(query))
        {
            resultList.Add(new
            {
                InstanceId = instance.InstanceId,
                CreatedAt = instance.CreatedAt,
                LastUpdatedAt = instance.LastUpdatedAt,
                Status = instance.RuntimeStatus.ToString()
            });
        }
        
        return Results.Ok(resultList);
    }
    catch (Exception ex)
    {
        logger.LogError(ex, "Error listing orchestrations");
        return Results.Problem("Error retrieving orchestrations");
    }
});

// Get the final HTML document for viewing
app.MapGet("/api/content/{instanceId}/document", async (string instanceId, [FromServices] DurableTaskClient client) => 
{
    try
    {
        var metadata = await client.GetInstanceAsync(instanceId, getInputsAndOutputs: true);
        if (metadata == null)
        {
            return Results.NotFound($"No orchestration found with ID: {instanceId}");
        }

        if (metadata.RuntimeStatus == OrchestrationRuntimeStatus.Completed && metadata.ReadOutputAs<ContentWorkflowResult>() is ContentWorkflowResult result)
        {
            if (!string.IsNullOrEmpty(result.FinalArticle))
            {
                return Results.Content(result.FinalArticle, "text/html", System.Text.Encoding.UTF8);
            }
            else
            {
                return Results.NotFound("Final article content not available");
            }
        }
        else
        {
            return Results.BadRequest($"Orchestration is not completed. Current status: {metadata.RuntimeStatus}");
        }
    }
    catch (Exception ex)
    {
        logger.LogError(ex, "Error retrieving document");
        return Results.Problem("Error retrieving document");
    }
});

// Download the final HTML document as a file
app.MapGet("/api/content/{instanceId}/download", async (string instanceId, [FromServices] DurableTaskClient client) => 
{
    try
    {
        var metadata = await client.GetInstanceAsync(instanceId, getInputsAndOutputs: true);
        if (metadata == null)
        {
            return Results.NotFound($"No orchestration found with ID: {instanceId}");
        }

        if (metadata.RuntimeStatus == OrchestrationRuntimeStatus.Completed && metadata.ReadOutputAs<ContentWorkflowResult>() is ContentWorkflowResult result)
        {
            if (!string.IsNullOrEmpty(result.FinalArticle))
            {
                var contentBytes = System.Text.Encoding.UTF8.GetBytes(result.FinalArticle);
                var fileName = $"article-{instanceId}-{DateTime.UtcNow:yyyyMMddHHmmss}.html";
                
                return Results.File(contentBytes, "text/html", fileName);
            }
            else
            {
                return Results.NotFound("Final article content not available");
            }
        }
        else
        {
            return Results.BadRequest($"Orchestration is not completed. Current status: {metadata.RuntimeStatus}");
        }
    }
    catch (Exception ex)
    {
        logger.LogError(ex, "Error downloading document");
        return Results.Problem("Error downloading document");
    }
});

// Wait for a specific result (polling endpoint)
app.MapGet("/api/content/{instanceId}/wait", async (string instanceId, int timeoutSeconds, [FromServices] DurableTaskClient client) => 
{
    try
    {
        timeoutSeconds = Math.Min(timeoutSeconds, 60); // Cap at 60 seconds max
        using var timeoutCts = new CancellationTokenSource(TimeSpan.FromSeconds(timeoutSeconds));
        
        OrchestrationMetadata metadata = await client.WaitForInstanceCompletionAsync(
            instanceId,
            getInputsAndOutputs: true,
            timeoutCts.Token);
        
        if (metadata.RuntimeStatus == OrchestrationRuntimeStatus.Completed)
        {
            ContentWorkflowResult? result = metadata.ReadOutputAs<ContentWorkflowResult>();
            return Results.Ok(result);
        }
        else
        {
            return Results.Ok(new
            {
                InstanceId = metadata.InstanceId,
                Status = metadata.RuntimeStatus.ToString(),
                Message = "Orchestration not yet complete"
            });
        }
    }
    catch (OperationCanceledException)
    {
        return Results.Accepted($"/api/content/{instanceId}", new { Message = "Operation timed out, but still processing" });
    }
    catch (Exception ex)
    {
        logger.LogError(ex, "Error waiting for orchestration");
        return Results.Problem("Error waiting for content generation to complete");
    }
});

// Start the app
try
{
    logger.LogInformation("Starting web host on port 5000");
    app.Run("http://0.0.0.0:5000");
}
catch (Exception ex)
{
    logger.LogError(ex, "Error starting client application");
}