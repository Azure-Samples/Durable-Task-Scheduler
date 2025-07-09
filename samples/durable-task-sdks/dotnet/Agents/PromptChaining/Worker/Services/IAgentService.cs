using System.Text.Json;
using Azure;
using Azure.AI.Projects;
using Azure.AI.Agents.Persistent;
using Azure.Core;
using Azure.Identity;
using Microsoft.Extensions.Logging;
using System.Text;

namespace AgentChainingSample.Services;

/// <summary>
/// Interface for agent services
/// </summary>
public interface IAgentService
{
    /// <summary>
    /// Gets the agent ID used by this service
    /// </summary>
    string AgentId { get; set; }

    /// <summary>
    /// Gets the endpoint URI for this agent service
    /// </summary>
    string Endpoint { get; }
    
    /// <summary>
    /// Ensures the agent exists, creating it if necessary
    /// </summary>
    Task<string> EnsureAgentExistsAsync(string agentName, string systemPrompt);

    /// <summary>
    /// Gets a response from the agent
    /// </summary>
    Task<string> GetResponseAsync(string prompt);

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
    protected readonly ILogger Logger;
    protected readonly AIProjectClient ProjectClient;
    protected PersistentAgentsClient AgentsClient;
    protected readonly TokenCredential Credential;
    
    // Retry configuration
    private const int MaxRetryAttempts = 3;
    private const int InitialRetryDelayMs = 1000; // Start with a 1 second delay
    
    // Deployment name from the instructions
    protected const string DeploymentName = "gpt-4o-mini";

    public string AgentId { get; set; }
    public string Endpoint { get; }

    protected BaseAgentService(string endpointUrl, ILogger<BaseAgentService> logger)
    {
        AgentId = string.Empty; // Will be set during initialization
        Endpoint = endpointUrl ?? throw new ArgumentNullException(nameof(endpointUrl));
        Logger = logger ?? throw new ArgumentNullException(nameof(logger));
        
        // Create credential for authentication
        Credential = new DefaultAzureCredential();
        
        Logger.LogInformation($"Initializing Azure AI Projects client with endpoint: {Endpoint}");
        
        // Create the AIProjectClient using the endpoint and credential
        ProjectClient = new AIProjectClient(new Uri(Endpoint), Credential);
        
        // Get the PersistentAgentsClient for agent operations
        AgentsClient = ProjectClient.GetPersistentAgentsClient();
        
        Logger.LogInformation("Azure AI Projects client and Persistent Agents client initialized successfully");

        JsonOptions = new JsonSerializerOptions
        {
            PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
            WriteIndented = true,
            PropertyNameCaseInsensitive = true,
            AllowTrailingCommas = true
        };
    }
    

    
    /// <summary>
    /// Ensures the agent exists, creating it if necessary
    /// </summary>
    public async Task<string> EnsureAgentExistsAsync(string agentName, string systemPrompt)
    {
        Logger.LogInformation($"Setting up agent: {agentName}");
        Logger.LogInformation($"Using Azure AI Projects endpoint: {Endpoint}");

        try
        {
            // Check if an agent with this name already exists
            PersistentAgent? existingAgent = null;
            string existingAgentId = string.Empty;
            
            try
            {
                Logger.LogInformation($"Attempting to retrieve agents from Azure AI Projects endpoint: {Endpoint}");
                var agents = AgentsClient.Administration.GetAgents();
                
                // Try to find an existing agent with the same name
                foreach (var agent in agents)
                {
                    if (agent.Name == agentName)
                    {
                        existingAgent = agent;
                        existingAgentId = agent.Id;
                        Logger.LogInformation($"Found existing agent: {agentName} with ID: {existingAgentId}");
                        break;
                    }
                }
            }
            catch (RequestFailedException rfEx)
            {
                Logger.LogWarning($"Error getting agents: {rfEx.Message}. Status: {rfEx.Status}");
                throw; // Re-throw the exception to surface it
            }
            
            string agentId;
            
            if (existingAgent == null)
            {
                try
                {
                    Logger.LogInformation($"Creating new agent: {agentName}");
                    Logger.LogInformation($"Using Azure AI Projects endpoint: {Endpoint}");
                    Logger.LogInformation($"Using model deployment: {DeploymentName}");
                    
                    // Create a new agent
                    var agentResponse = await AgentsClient.Administration.CreateAgentAsync(
                        DeploymentName,
                        agentName,
                        systemPrompt,
                        $"News Article Generator agent created on {DateTime.UtcNow:yyyy-MM-dd}");
                    
                    agentId = agentResponse.Value.Id;
                    
                    Logger.LogInformation($"Created new agent: {agentName} with ID: {agentId}");
                }
                catch (Exception ex)
                {
                    Logger.LogWarning($"Error creating agent: {ex.Message}");
                    throw; // Re-throw the exception to surface it
                }
            }
            else
            {
                agentId = existingAgentId;
                
                try
                {
                    // Update the agent's system prompt if it exists
                    var updatedAgent = await AgentsClient.Administration.UpdateAgentAsync(
                        agentId,
                        systemPrompt,
                        $"News Article Generator agent updated on {DateTime.UtcNow:yyyy-MM-dd}");
                    Logger.LogInformation($"Updated existing agent: {agentName} with ID: {agentId}");
                }
                catch (Exception ex)
                {
                    Logger.LogWarning($"Error updating agent: {ex.Message}. Using existing agent ID anyway.");
                }
            }
            
            // Set the agent ID field
            AgentId = agentId;
            return agentId;
        }
        catch (Exception ex)
        {
            Logger.LogWarning($"Unexpected error in EnsureAgentExistsAsync: {ex.Message}");
            throw; // Re-throw the exception to surface it
        }
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
                int startIndex = response.IndexOf('\n', codeBlockStart) + 1;
                int endIndex = codeBlockEnd;
                
                // Make sure we have valid start and end indices
                if (startIndex > 0 && endIndex > startIndex)
                {
                    string codeContent = response.Substring(startIndex, endIndex - startIndex).Trim();
                    Logger.LogInformation("[JSON-PARSER] Extracted content from code block");
                    
                    // Remove any language specifier like ```json
                    if (codeContent.StartsWith("json", StringComparison.OrdinalIgnoreCase))
                    {
                        codeContent = codeContent.Substring(4).Trim();
                    }
                    
                    response = codeContent;
                }
            }
        }

        // Check if response is wrapped in backticks
        if (response.StartsWith("`") && response.EndsWith("`"))
        {
            response = response.Substring(1, response.Length - 2).Trim();
            Logger.LogInformation("[JSON-PARSER] Removed backticks");
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
            return "{}"; // Return empty JSON object as fallback
        }
    }

    public async Task<string> GetResponseAsync(string prompt)
    {
        
        int retryCount = 0;
        int retryDelay = InitialRetryDelayMs;
        bool shouldRetry;

        do
        {
            shouldRetry = false;

            try
            {
                // Check if agent ID is set
                if (string.IsNullOrEmpty(AgentId))
                {
                    throw new InvalidOperationException($"Agent ID is not set. Call EnsureAgentExistsAsync first.");
                }

                Logger.LogInformation($"Getting response from agent {AgentId}");

                // Create a thread
                var threadResponse = await AgentsClient.Threads.CreateThreadAsync();
                string threadId = threadResponse.Value.Id;
                Logger.LogInformation($"Created thread, thread ID: {threadId}");

                // Create message content
                var messageContent = new List<MessageInputContentBlock>
                {
                    new MessageInputTextBlock(prompt)
                };
                
                // Send the prompt to the thread
                var messageResponse = await AgentsClient.Messages.CreateMessageAsync(
                    threadId: threadId,
                    role: MessageRole.User,
                    content: prompt);
                
                var threadMessage = messageResponse.Value;
                Logger.LogInformation($"Created message, message ID: {threadMessage.Id}");

                // Create a run with the agent using the agent ID
                var runResponse = await AgentsClient.Runs.CreateRunAsync(threadId, AgentId);
                
                var run = runResponse.Value;
                Logger.LogInformation($"Created run, run ID: {run.Id}");

                // Poll the run until it's completed
                do
                {
                    await Task.Delay(TimeSpan.FromMilliseconds(500));
                    var getRunResponse = await AgentsClient.Runs.GetRunAsync(threadId, run.Id);
                    run = getRunResponse.Value;
                }
                while (run.Status == RunStatus.Queued
                    || run.Status == RunStatus.InProgress
                    || run.Status == RunStatus.RequiresAction);

                Logger.LogInformation($"Run completed with status: {run.Status}");

                if (run.Status == RunStatus.Failed)
                {
                    // Try to extract error message
                    string errorMessage = run.LastError?.Message ?? string.Empty;
                    
                    // Check if the error is due to rate limiting
                    if (errorMessage.Contains("Rate limit") && retryCount < MaxRetryAttempts)
                    {
                        shouldRetry = true;
                        retryDelay = await HandleRetry(++retryCount, retryDelay, errorMessage);
                        continue;
                    }

                    throw new Exception($"Run failed: {errorMessage}");
                }

                // Get messages from the assistant thread
                var messagesResponse = AgentsClient.Messages.GetMessagesAsync(
                    threadId: threadId, 
                    order: ListSortOrder.Descending); // Get newest first
                
                List<PersistentThreadMessage> assistantMessages = new List<PersistentThreadMessage>();
                
                await foreach (var message in messagesResponse)
                {
                    if (message.Role == "assistant")
                    {
                        assistantMessages.Add(message);
                    }
                }
                
                if (assistantMessages.Count == 0)
                {
                    Logger.LogWarning("No assistant messages found in the response");
                    return string.Empty;
                }

                // Get the most recent message from the assistant (first in list with Descending order)
                var latestMessage = assistantMessages.First();
                
                // Extract all text content items from the message
                StringBuilder contentBuilder = new StringBuilder();
                
                foreach (var contentItem in latestMessage.ContentItems)
                {
                    if (contentItem is MessageTextContent textContent)
                    {
                        contentBuilder.Append(textContent.Text);
                    }
                }
                
                string responseContent = contentBuilder.ToString();
                
                Logger.LogInformation($"Retrieved response from agent {AgentId}, content length: {responseContent.Length} characters");
                
                return responseContent;
            }
            catch (RequestFailedException ex) when (
                (ex.Status == 429 ||  // 429 Too Many Requests
                ex.Status == 503) &&  // 503 Service Unavailable
                retryCount < MaxRetryAttempts)
            {
                shouldRetry = true;
                retryDelay = await HandleRetry(++retryCount, retryDelay, ex.Message);
            }
            catch (Exception ex)
            {
                // Check if the exception message contains indication of rate limit
                if (ex.Message.Contains("Rate limit") && retryCount < MaxRetryAttempts)
                {
                    shouldRetry = true;
                    retryDelay = await HandleRetry(++retryCount, retryDelay, ex.Message);
                }
                else
                {
                    Logger.LogError($"Error calling agent {AgentId}: {ex.Message}");
                    throw;
                }
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
