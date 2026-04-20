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
//   HTTP POST /chat/{sessionId}
//     → subscribe to Redis channel "chat:{sessionId}:{correlationId}"
//     → signal entity (fire-and-forget)
//     → stream SSE from Redis subscription
//   Entity receives signal:
//     → runs agent loop (LLM ←→ tool execution)
//     → publishes response chunks to Redis
//     → publishes [DONE] when complete
// ============================================================================

using System.Text.Json;
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
                    // Accumulate tool calls
                    foreach (var content in update.Contents.OfType<FunctionCallContent>())
                        toolCalls.Add(content);

                    // Stream text chunks to Redis immediately
                    if (!string.IsNullOrEmpty(update.Text))
                    {
                        fullText.Append(update.Text);
                        var json = JsonSerializer.Serialize(new { type = "chunk", content = update.Text });
                        await pub.PublishAsync(channel, json);
                    }
                }

                if (toolCalls.Count > 0)
                {
                    // LLM wants to call tools — execute and loop
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

                // Final text reply — save to durable state
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
}
