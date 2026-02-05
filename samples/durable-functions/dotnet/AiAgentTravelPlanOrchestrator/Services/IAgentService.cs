using System.Text.Json;
using Azure;
using Azure.Identity;
using Azure.AI.Projects;
using Azure.AI.Agents.Persistent;
using System.Net;
using Microsoft.Extensions.Logging;

namespace TravelPlannerFunctions.Services;

/// <summary>
/// Base interface for agent services
/// </summary>
public interface IAgentService
{
    /// <summary>
    /// Gets the agent ID used by this service
    /// </summary>
    string AgentId { get; }

    /// <summary>
    /// Gets the connection string for this agent service
    /// </summary>
    string ConnectionString { get; }

    /// <summary>
    /// Gets a response from the agent
    /// </summary>
    Task<string> GetAgentResponseAsync(string prompt);

    /// <summary>
    /// Cleans JSON response from markdown formatting
    /// </summary>
    string CleanJsonResponse(string response);
}

/// <summary>
/// Base implementation for agent services
/// </summary>
public abstract class BaseAgentService : IAgentService
{
    protected readonly JsonSerializerOptions JsonOptions;
    protected readonly ILogger<BaseAgentService> Logger;

    // Retry configuration
    private const int MaxRetryAttempts = 5;
    private const int InitialRetryDelayMs = 1000; // Start with a 1 second delay

    public string AgentId { get; }
    public string ConnectionString { get; }

    protected BaseAgentService(string agentId, string connectionString, ILogger<BaseAgentService> logger)
    {
        AgentId = agentId;
        ConnectionString = connectionString;
        Logger = logger ?? throw new ArgumentNullException(nameof(logger));

        JsonOptions = new JsonSerializerOptions
        {
            PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
            WriteIndented = true,
            PropertyNameCaseInsensitive = true,
            AllowTrailingCommas = true
        };
    }

    /// <summary>
    /// Gets an agent identifier from environment variables
    /// </summary>
    /// <param name="envVarName">Environment variable name for the agent ID</param>
    /// <returns>The agent ID</returns>
    /// <exception cref="InvalidOperationException">Thrown when environment variable is not configured</exception>
    protected static string GetAgentIdentifier(string envVarName)
    {
        string? agentId = Environment.GetEnvironmentVariable(envVarName);
        if (string.IsNullOrEmpty(agentId))
        {
            throw new InvalidOperationException($"{envVarName} environment variable is not configured");
        }
        return agentId;
    }

    /// <summary>
    /// Validates and normalizes JSON responses from agents
    /// </summary>
    /// <param name="response">The JSON response from an agent</param>
    /// <returns>Validated JSON string</returns>
    public string CleanJsonResponse(string response)
    {
        if (string.IsNullOrEmpty(response))
        {
            Logger.LogWarning("[JSON-PARSER] Response was null or empty");
            return "{}";
        }

        Logger.LogInformation($"[JSON-PARSER] Processing response ({response.Length} chars)");

        // Trim any whitespace
        response = response.Trim();

        // Simple case: Check if response is already valid JSON
        try
        {
            using (JsonDocument.Parse(response))
            {
                Logger.LogInformation("[JSON-PARSER] Response is valid JSON");
                return response;
            }
        }
        catch (JsonException)
        {
            Logger.LogInformation("[JSON-PARSER] Initial JSON validation failed, attempting to extract JSON");
        }

        // Handle markdown code blocks if present
        if (response.Contains("```"))
        {
            // Find start and end of code block
            int codeBlockStart = response.IndexOf("```");
            int codeBlockEnd = response.LastIndexOf("```");

            if (codeBlockStart != codeBlockEnd) // Make sure we found both opening and closing markers
            {
                // Extract content between code blocks
                int contentStart = response.IndexOf('\n', codeBlockStart);
                if (contentStart != -1 && contentStart < codeBlockEnd)
                {
                    response = response.Substring(contentStart, codeBlockEnd - contentStart).Trim();
                    Logger.LogInformation("[JSON-PARSER] Extracted content from code block");
                }
            }

            // Remove any remaining backticks
            response = response.Replace("```", "").Trim();
        }
        // Handle single backtick blocks
        else if (response.StartsWith("`") && response.EndsWith("`"))
        {
            response = response.Substring(1, response.Length - 2).Trim();
            Logger.LogInformation("[JSON-PARSER] Removed single backticks");
        }

        // Final validation
        try
        {
            using (JsonDocument.Parse(response))
            {
                Logger.LogInformation("[JSON-PARSER] Successfully validated JSON");
                return response;
            }
        }
        catch (JsonException ex)
        {
            Logger.LogError($"[JSON-PARSER] Failed to parse JSON: {ex.Message}");

            // If we can't parse it, log the problematic JSON and return empty object
            if (response.Length > 200)
            {
                Logger.LogError($"[JSON-PARSER] Problematic JSON (first 200 chars): {response.Substring(0, 200)}...");
            }
            else
            {
                Logger.LogError($"[JSON-PARSER] Problematic JSON: {response}");
            }

            return "{}";
        }
    }

    public async Task<string> GetAgentResponseAsync(string prompt)
    {
        int retryCount = 0;
        int retryDelay = InitialRetryDelayMs;
        bool shouldRetry;

        do
        {
            shouldRetry = false;

            try
            {
                // Create a client using the connection string
                if (string.IsNullOrEmpty(ConnectionString))
                {
                    throw new InvalidOperationException($"Connection string for agent {AgentId} is not set.");
                }

                // Create an agents client with the connection string (endpoint)
                var tenantId = Environment.GetEnvironmentVariable("AZURE_TENANT_ID");
                var clientId = Environment.GetEnvironmentVariable("AZURE_CLIENT_ID");

                ArgumentNullException.ThrowIfNullOrEmpty(tenantId, nameof(tenantId));
                ArgumentNullException.ThrowIfNullOrEmpty(clientId, nameof(clientId));

                var projectClient = new AIProjectClient(new Uri(ConnectionString), new DefaultAzureCredential(
                    new DefaultAzureCredentialOptions
                    {
                        TenantId = tenantId,
                        ManagedIdentityClientId = clientId
                    }));
                var client = projectClient.GetPersistentAgentsClient();
                Logger.LogInformation($"Successfully created PersistentAgentsClient for agent {AgentId}");

                // Create a thread
                PersistentAgentThread thread = await client.Threads.CreateThreadAsync();
                string threadId = thread.Id;
                Logger.LogInformation($"Created thread, thread ID: {threadId}");

                // Send the prompt to the thread
                await client.Messages.CreateMessageAsync(
                    threadId,
                    MessageRole.User,
                    prompt);

                // Create a run with the agent using the agent ID
                ThreadRun run = await client.Runs.CreateRunAsync(
                    threadId,
                    AgentId);

                Logger.LogInformation($"Created run, run ID: {run.Id}");

                // Poll the run until it's completed
                do
                {
                    await Task.Delay(TimeSpan.FromMilliseconds(500));
                    run = await client.Runs.GetRunAsync(threadId, run.Id);
                }
                while (run.Status == RunStatus.Queued
                    || run.Status == RunStatus.InProgress
                    || run.Status == RunStatus.RequiresAction);

                Logger.LogInformation($"Run completed with status: {run.Status}");

                if (run.Status == RunStatus.Failed)
                {
                    string errorMessage = run.LastError?.Message ?? string.Empty;
                    throw new Exception($"Run failed: {errorMessage}");
                }

                // Get messages from the assistant thread
                var messagesPageable = client.Messages.GetMessagesAsync(threadId);

                // Get the most recent message from the assistant
                string lastMessage = string.Empty;
                await foreach (PersistentThreadMessage message in messagesPageable)
                {
                    // Skip user messages, we only want assistant responses
                    if (message.Role == MessageRole.User)
                    {
                        continue;
                    }

                    if (message.ContentItems.Count > 0)
                    {
                        MessageContent contentItem = message.ContentItems[0];
                        if (contentItem is MessageTextContent textItem)
                        {
                            lastMessage = textItem.Text;
                            break; // We got our response
                        }
                    }
                }

                return lastMessage;
            }
            catch (RequestFailedException ex) when (
                (ex.Status == (int)HttpStatusCode.TooManyRequests ||  // 429 Too Many Requests
                ex.Status == 503) &&                                  // 503 Service Unavailable
                retryCount < MaxRetryAttempts)
            {
                shouldRetry = true;
                retryDelay = await HandleRetry(++retryCount, retryDelay, ex.Message);
            }
            catch (Exception ex)
            {
                Logger.LogError(ex, "Error calling agent {AgentId}", AgentId);
                throw;
            }
        } while (shouldRetry);

        // This should not be reached unless all retry attempts fail
        throw new Exception($"Failed to get a response from agent {AgentId} after {MaxRetryAttempts} attempts");
    }

    private async Task<int> HandleRetry(int retryCount, int retryDelay, string errorMessage)
    {
        // Calculate exponential backoff with jitter
        int maxJitterMs = retryDelay / 4;
        Random random = new Random();
        int jitter = random.Next(-maxJitterMs, maxJitterMs);
        int actualDelay = retryDelay + jitter;

        Logger.LogInformation($"Rate limit hit for agent {AgentId}. Retrying in {actualDelay}ms (attempt {retryCount} of {MaxRetryAttempts}). Error: {errorMessage}");

        // Wait for the calculated delay
        await Task.Delay(actualDelay);

        // Double the delay for the next retry (exponential backoff)
        return retryDelay * 2;
    }
}