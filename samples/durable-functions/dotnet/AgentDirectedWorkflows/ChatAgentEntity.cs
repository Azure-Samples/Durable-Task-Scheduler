// ============================================================================
// Durable Agent Chat — powered by Durable Entities + Redis Streaming
//
// Each chat session is a durable entity that holds the full conversation
// history. When you send a message, the entity calls the LLM, executes any
// tool calls, and streams response chunks to Redis pub/sub in real-time.
// The HTTP layer subscribes to the Redis channel and forwards chunks as
// Server-Sent Events (SSE) to the client.
//
// Architecture:
//   HTTP POST /api/chat/{sessionId}
//     → subscribe to Redis channel "chat:{sessionId}:{correlationId}"
//     → signal entity (fire-and-forget)
//     → stream SSE from Redis subscription
//   Entity receives signal:
//     → runs agent loop (LLM ←→ tool execution)
//     → publishes response chunks to Redis
//     → publishes [DONE] when complete
// ============================================================================

using System.Text.Json;
using System.Threading.Channels;
using Microsoft.AspNetCore.Http;
using Microsoft.AspNetCore.Mvc;
using Microsoft.Azure.Functions.Worker;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Client.Entities;
using Microsoft.DurableTask.Entities;
using Microsoft.Extensions.AI;
using Microsoft.Extensions.Logging;
using StackExchange.Redis;

namespace AgentDirectedWorkflows;

// ─── The Agent Entity ───
// Each entity instance is one chat session. The Durable Task framework
// automatically persists the entity's State after each operation.

public class ChatAgentEntity : TaskEntity<ChatAgentState>
{
    private readonly IChatClient _chatClient;
    private readonly IConnectionMultiplexer _redis;
    private readonly ILogger<ChatAgentEntity> _logger;

    public ChatAgentEntity(IChatClient chatClient, IConnectionMultiplexer redis, ILogger<ChatAgentEntity> logger)
    {
        _chatClient = chatClient;
        _redis = redis;
        _logger = logger;
    }

    /// <summary>
    /// The core agent loop: send user message to LLM, execute any tool calls,
    /// repeat until the LLM gives a final text reply. Streams response chunks
    /// to Redis pub/sub for real-time delivery to the client.
    /// </summary>
    public async Task Message(ChatRequest request)
    {
        var channel = RedisChannel.Literal($"chat:{Context.Id.Key}:{request.CorrelationId}");
        var pub = _redis.GetSubscriber();

        try
        {
            State.Messages.Add(new ChatMsg("user", request.Message));

            var messages = new List<ChatMessage> { new(ChatRole.System, "You are a helpful assistant.") };
            foreach (var m in State.Messages)
                messages.Add(new ChatMessage(m.Role == "assistant" ? ChatRole.Assistant : ChatRole.User, m.Content));

            var options = new ChatOptions { Tools = AgentTools.AsAITools() };

            // Agent loop: stream from LLM → publish chunks or handle tool calls
            while (true)
            {
                var fullText = new System.Text.StringBuilder();
                var toolCalls = new List<FunctionCallContent>();

                await foreach (var update in _chatClient.GetStreamingResponseAsync(messages, options))
                {
                    foreach (var content in update.Contents.OfType<FunctionCallContent>())
                        toolCalls.Add(content);

                    if (!string.IsNullOrEmpty(update.Text))
                    {
                        fullText.Append(update.Text);
                        var json = JsonSerializer.Serialize(new { type = "chunk", content = update.Text });
                        await pub.PublishAsync(channel, json);
                    }
                }

                if (toolCalls.Count > 0)
                {
                    _logger.LogInformation("Executing tools: {Tools}", string.Join(", ", toolCalls.Select(t => t.Name)));
                    messages.Add(new ChatMessage(ChatRole.Assistant,
                        toolCalls.Select(tc => (AIContent)tc).ToList()));

                    foreach (var tc in toolCalls)
                    {
                        var result = AgentTools.Execute(tc.Name, tc.Arguments);
                        messages.Add(new ChatMessage(ChatRole.Tool, [new FunctionResultContent(tc.CallId, result)]));
                    }
                    continue;
                }

                var reply = fullText.ToString();
                State.Messages.Add(new ChatMsg("assistant", reply));
                await pub.PublishAsync(channel, JsonSerializer.Serialize(new { type = "done" }));
                return;
            }
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Agent loop failed");
            var errorJson = JsonSerializer.Serialize(new { type = "error", content = ex.Message });
            await pub.PublishAsync(channel, errorJson);
        }
    }

    public List<ChatMsg> GetHistory() => State.Messages;

    public void Reset() => State.Messages.Clear();

    [Function(nameof(ChatAgentEntity))]
    public static Task Dispatch([EntityTrigger] TaskEntityDispatcher dispatcher)
        => dispatcher.DispatchAsync<ChatAgentEntity>();
}

// ─── HTTP Endpoints ───

public class ChatEndpoints
{
    private readonly IConnectionMultiplexer _redis;

    public ChatEndpoints(IConnectionMultiplexer redis)
    {
        _redis = redis;
    }

    /// <summary>
    /// Send a message to the agent. Streams SSE by default; add ?stream=false for a simple JSON response.
    /// </summary>
    [Function("SendMessage")]
    public async Task SendMessage(
        [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "chat/{sessionId}")] HttpRequest req,
        string sessionId,
        [DurableClient] DurableTaskClient client)
    {
        bool stream = !string.Equals(req.Query["stream"], "false", StringComparison.OrdinalIgnoreCase);
        var body = await req.ReadFromJsonAsync<ChatRequest>();
        var message = body?.Message ?? "Hello";
        var correlationId = Guid.NewGuid().ToString("N");
        var channel = RedisChannel.Literal($"chat:{sessionId}:{correlationId}");

        // Subscribe to Redis BEFORE signaling the entity to avoid missing messages
        var sub = _redis.GetSubscriber();
        var queue = Channel.CreateUnbounded<string>();
        await sub.SubscribeAsync(channel, (_, msg) => queue.Writer.TryWrite(msg!));

        // Signal the entity (fire-and-forget)
        var entityId = new EntityInstanceId(nameof(ChatAgentEntity), sessionId);
        await client.Entities.SignalEntityAsync(entityId, "Message",
            new ChatRequest(sessionId, message, correlationId));

        var ct = req.HttpContext.RequestAborted;
        using var timeout = new CancellationTokenSource(TimeSpan.FromMinutes(2));
        using var linked = CancellationTokenSource.CreateLinkedTokenSource(ct, timeout.Token);

        try
        {
            if (stream)
            {
                // Streaming mode: forward chunks as Server-Sent Events
                req.HttpContext.Response.ContentType = "text/event-stream";
                req.HttpContext.Response.Headers.CacheControl = "no-cache";

                await foreach (var chunk in queue.Reader.ReadAllAsync(linked.Token))
                {
                    await req.HttpContext.Response.WriteAsync($"data: {chunk}\n\n", linked.Token);
                    await req.HttpContext.Response.Body.FlushAsync(linked.Token);

                    try
                    {
                        var doc = JsonDocument.Parse(chunk);
                        var type = doc.RootElement.GetProperty("type").GetString();
                        if (type is "done" or "error") break;
                    }
                    catch { /* not JSON or missing type — keep streaming */ }
                }
            }
            else
            {
                // Non-streaming mode: collect all chunks, return complete JSON response
                var fullResponse = new System.Text.StringBuilder();
                await foreach (var chunk in queue.Reader.ReadAllAsync(linked.Token))
                {
                    try
                    {
                        var doc = JsonDocument.Parse(chunk);
                        var type = doc.RootElement.GetProperty("type").GetString();
                        if (type == "chunk")
                            fullResponse.Append(doc.RootElement.GetProperty("content").GetString());
                        if (type is "done" or "error")
                        {
                            if (type == "error")
                            {
                                req.HttpContext.Response.StatusCode = 500;
                                await req.HttpContext.Response.WriteAsJsonAsync(new
                                {
                                    sessionId,
                                    error = doc.RootElement.GetProperty("content").GetString()
                                }, linked.Token);
                                return;
                            }
                            break;
                        }
                    }
                    catch { /* keep reading */ }
                }

                await req.HttpContext.Response.WriteAsJsonAsync(new
                {
                    sessionId,
                    message = fullResponse.ToString()
                }, linked.Token);
            }
        }
        finally
        {
            await sub.UnsubscribeAsync(channel);
        }
    }

    /// <summary>Get the full conversation history for a session.</summary>
    [Function("GetHistory")]
    public async Task<IActionResult> GetHistory(
        [HttpTrigger(AuthorizationLevel.Anonymous, "get", Route = "chat/{sessionId}/history")] HttpRequest req,
        string sessionId,
        [DurableClient] DurableTaskClient client)
    {
        var entityId = new EntityInstanceId(nameof(ChatAgentEntity), sessionId);
        var entity = await client.Entities.GetEntityAsync<ChatAgentState>(entityId);
        if (entity is null) return new NotFoundResult();
        return new OkObjectResult(new { sessionId, history = entity.State.Messages });
    }

    /// <summary>Reset a session's conversation history.</summary>
    [Function("ResetSession")]
    public async Task<IActionResult> Reset(
        [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "chat/{sessionId}/reset")] HttpRequest req,
        string sessionId,
        [DurableClient] DurableTaskClient client)
    {
        var entityId = new EntityInstanceId(nameof(ChatAgentEntity), sessionId);
        await client.Entities.SignalEntityAsync(entityId, "Reset");
        return new OkObjectResult(new { sessionId, status = "reset" });
    }
}
