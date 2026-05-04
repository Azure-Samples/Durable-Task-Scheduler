using System.Runtime.CompilerServices;
using Microsoft.Azure.Functions.Worker;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.AI;
using StackExchange.Redis;

var host = new HostBuilder()
    .ConfigureFunctionsWebApplication()
    .ConfigureServices((context, services) =>
    {
        services.AddApplicationInsightsTelemetryWorkerService();
        services.ConfigureFunctionsApplicationInsights();

        // Register IChatClient backed by Azure OpenAI.
        // Set AZURE_OPENAI_ENDPOINT and AZURE_OPENAI_DEPLOYMENT in local.settings.json.
        string? aiEndpoint = context.Configuration["AZURE_OPENAI_ENDPOINT"];
        string? aiDeployment = context.Configuration["AZURE_OPENAI_DEPLOYMENT"];

        if (!string.IsNullOrEmpty(aiEndpoint) && !string.IsNullOrEmpty(aiDeployment))
        {
            services.AddSingleton<IChatClient>(_ =>
                new Azure.AI.OpenAI.AzureOpenAIClient(
                        new Uri(aiEndpoint), new Azure.Identity.DefaultAzureCredential())
                    .GetChatClient(aiDeployment)
                    .AsIChatClient());
        }
        else
        {
            // Simple echo client for local dev without an Azure OpenAI deployment
            services.AddSingleton<IChatClient>(new EchoChatClient());
        }

        // Register Redis for streaming response chunks from entities to HTTP endpoints
        string redisConnection = context.Configuration["REDIS_CONNECTION_STRING"] ?? "localhost:6379";
        services.AddSingleton<IConnectionMultiplexer>(_ => ConnectionMultiplexer.Connect(redisConnection));
    })
    .Build();

host.Run();

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
        var reply = new ChatMessage(ChatRole.Assistant, $"Echo: {last}");
        return Task.FromResult(new ChatResponse([reply]));
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
