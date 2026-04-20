using System.Runtime.CompilerServices;
using System.Text.Json;
using System.Threading.Channels;
using AgentDirectedWorkflows;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Client.AzureManaged;
using Microsoft.DurableTask.Entities;
using Microsoft.DurableTask.Worker;
using Microsoft.DurableTask.Worker.AzureManaged;
using Microsoft.Extensions.AI;
using StackExchange.Redis;

var builder = WebApplication.CreateBuilder(args);

string connectionString = builder.Configuration["DURABLE_TASK_SCHEDULER_CONNECTION_STRING"]
    ?? "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";

string redisConnection = builder.Configuration["REDIS_CONNECTION_STRING"] ?? "localhost:6379";

// Configure Azure OpenAI if available, otherwise use a simple echo client for demo purposes.
string? aiEndpoint = builder.Configuration["AZURE_OPENAI_ENDPOINT"];
string? aiDeployment = builder.Configuration["AZURE_OPENAI_DEPLOYMENT"];

if (!string.IsNullOrEmpty(aiEndpoint) && !string.IsNullOrEmpty(aiDeployment))
{
    builder.Services.AddSingleton<IChatClient>(_ =>
        new Azure.AI.OpenAI.AzureOpenAIClient(new Uri(aiEndpoint), new Azure.Identity.DefaultAzureCredential())
            .GetChatClient(aiDeployment)
            .AsIChatClient());
}
else
{
    builder.Services.AddSingleton<IChatClient>(new EchoChatClient());
}

// Register Redis
builder.Services.AddSingleton<IConnectionMultiplexer>(_ => ConnectionMultiplexer.Connect(redisConnection));

// Register Durable Task worker (entity only — no orchestration bridge needed)
builder.Services.AddDurableTaskWorker(b =>
{
    b.AddTasks(r =>
    {
        r.AddEntity(nameof(ChatAgentEntity), sp =>
            ActivatorUtilities.CreateInstance<ChatAgentEntity>(sp));
    });
    b.UseDurableTaskScheduler(connectionString);
});
builder.Services.AddDurableTaskClient(b => b.UseDurableTaskScheduler(connectionString));

builder.Services.AddLogging(l => l.AddSimpleConsole(o =>
{
    o.SingleLine = true;
    o.UseUtcTimestamp = true;
    o.TimestampFormat = "yyyy-MM-ddTHH:mm:ss.fffZ ";
}).SetMinimumLevel(LogLevel.Warning).AddFilter("AgentDirectedWorkflows", LogLevel.Information));

var app = builder.Build();

// ─── HTTP Endpoints ───
// Each sessionId maps to a durable entity instance — a persistent agent with durable state.

// Send a message to the agent. Streams SSE by default; add ?stream=false for a simple JSON response.
app.MapPost("/chat/{sessionId}", async (HttpContext httpContext, DurableTaskClient client,
    IConnectionMultiplexer redis, string sessionId, ChatRequest req) =>
{
    bool stream = !string.Equals(httpContext.Request.Query["stream"], "false", StringComparison.OrdinalIgnoreCase);
    var correlationId = Guid.NewGuid().ToString("N");
    var channel = RedisChannel.Literal($"chat:{sessionId}:{correlationId}");

    // Subscribe to Redis BEFORE signaling the entity to avoid missing messages
    var sub = redis.GetSubscriber();
    var queue = Channel.CreateUnbounded<string>();
    await sub.SubscribeAsync(channel, (_, message) => queue.Writer.TryWrite(message!));

    // Signal the entity (fire-and-forget) — it will publish chunks to Redis
    var entityId = new EntityInstanceId(nameof(ChatAgentEntity), sessionId);
    await client.Entities.SignalEntityAsync(entityId, "Message",
        new ChatRequest(sessionId, req.Message, correlationId));

    var ct = httpContext.RequestAborted;
    using var timeout = new CancellationTokenSource(TimeSpan.FromMinutes(2));
    using var linked = CancellationTokenSource.CreateLinkedTokenSource(ct, timeout.Token);

    try
    {
        if (stream)
        {
            // Streaming mode: forward chunks as Server-Sent Events
            httpContext.Response.ContentType = "text/event-stream";
            httpContext.Response.Headers.CacheControl = "no-cache";

            await foreach (var message in queue.Reader.ReadAllAsync(linked.Token))
            {
                await httpContext.Response.WriteAsync($"data: {message}\n\n", linked.Token);
                await httpContext.Response.Body.FlushAsync(linked.Token);

                try
                {
                    var doc = JsonDocument.Parse(message);
                    var type = doc.RootElement.GetProperty("type").GetString();
                    if (type is "done" or "error") break;
                }
                catch { /* not JSON or no type field — keep streaming */ }
            }
        }
        else
        {
            // Non-streaming mode: collect all chunks, return complete JSON response
            var fullResponse = new System.Text.StringBuilder();
            await foreach (var message in queue.Reader.ReadAllAsync(linked.Token))
            {
                try
                {
                    var doc = JsonDocument.Parse(message);
                    var type = doc.RootElement.GetProperty("type").GetString();
                    if (type == "chunk")
                        fullResponse.Append(doc.RootElement.GetProperty("content").GetString());
                    if (type is "done" or "error")
                    {
                        if (type == "error")
                        {
                            httpContext.Response.StatusCode = 500;
                            await httpContext.Response.WriteAsJsonAsync(new
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

            await httpContext.Response.WriteAsJsonAsync(new
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
});

// Get conversation history
app.MapGet("/chat/{sessionId}/history", async (DurableTaskClient client, string sessionId) =>
{
    var entityId = new EntityInstanceId(nameof(ChatAgentEntity), sessionId);
    var entity = await client.Entities.GetEntityAsync<ChatAgentState>(entityId);
    if (entity is null) return Results.NotFound();
    return Results.Ok(new { sessionId, history = entity.State.Messages });
});

// Reset a conversation
app.MapPost("/chat/{sessionId}/reset", async (DurableTaskClient client, string sessionId) =>
{
    var entityId = new EntityInstanceId(nameof(ChatAgentEntity), sessionId);
    await client.Entities.SignalEntityAsync(entityId, "Reset");
    return Results.Ok(new { sessionId, status = "reset" });
});

app.Run();

/// <summary>
/// Trivial IChatClient that echoes back the user message with simulated streaming.
/// Set AZURE_OPENAI_ENDPOINT + AZURE_OPENAI_DEPLOYMENT env vars to use a real LLM.
/// </summary>
class EchoChatClient : IChatClient
{
    public Task<ChatResponse> GetResponseAsync(
        IEnumerable<ChatMessage> messages, ChatOptions? options = null, CancellationToken ct = default)
    {
        var last = messages.LastOrDefault(m => m.Role == ChatRole.User)?.Text ?? "Hello";
        return Task.FromResult(new ChatResponse([new ChatMessage(ChatRole.Assistant, $"Echo: {last}")]));
    }

    public async IAsyncEnumerable<ChatResponseUpdate> GetStreamingResponseAsync(
        IEnumerable<ChatMessage> messages, ChatOptions? options = null,
        [EnumeratorCancellation] CancellationToken ct = default)
    {
        var last = messages.LastOrDefault(m => m.Role == ChatRole.User)?.Text ?? "Hello";
        foreach (var word in $"Echo: {last}".Split(' '))
        {
            yield return new ChatResponseUpdate(ChatRole.Assistant, word + " ");
            await Task.Delay(50, ct);
        }
    }

    public void Dispose() { }
    public object? GetService(Type serviceType, object? serviceKey = null) => null;
}
