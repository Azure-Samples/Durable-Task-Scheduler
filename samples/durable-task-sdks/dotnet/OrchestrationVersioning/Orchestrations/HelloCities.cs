using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;

namespace AspNetWebApp.Scenarios;

[DurableTask]
class HelloCities : TaskOrchestrator<string, List<string>>
{
    private readonly string[] Cities = ["Seattle", "Amsterdam", "Hyderabad", "Kuala Lumpur", "Shanghai", "Tokyo"];

    public override async Task<List<string>> RunAsync(TaskOrchestrationContext context, string input)
    {
        List<string> results = [];
        foreach (var city in Cities)
        {
            results.Add(await context.CallSayHelloAsync($"{city} v{context.Version}"));
            if (context.CompareVersionTo("2.0.0") >= 0)
            {
                results.Add(await context.CallSayGoodbyeAsync($"{city} v{context.Version}"));
            }
        }

        Console.WriteLine("HelloCities orchestration completed.");
        return results;
    }
}

[DurableTask]
class SayHello : TaskActivity<string, string>
{
    public override Task<string> RunAsync(TaskActivityContext context, string cityName)
    {
        return Task.FromResult($"Hello, {cityName}!");
    }
}

[DurableTask]
class SayGoodbye : TaskActivity<string, string>
{
    public override Task<string> RunAsync(TaskActivityContext context, string cityName)
    {
        return Task.FromResult<string>($"Goodbye, {cityName}!");
    }
}