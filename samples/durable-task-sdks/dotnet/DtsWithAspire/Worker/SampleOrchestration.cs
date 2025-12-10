using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;

[DurableTask]
public class SampleOrchestration : TaskOrchestrator<string, List<string>>
{
    public override async Task<List<string>> RunAsync(TaskOrchestrationContext context, string input)
    {
        ILogger logger = context.CreateReplaySafeLogger(nameof(SampleOrchestration));
        logger.LogInformation("Saying hello.");
        var outputs = new List<string>();

        // Replace name and input with values relevant for your Durable Functions Activity
        outputs.Add(await context.CallActivityAsync<string>(nameof(SayHello), "Tokyo"));
        outputs.Add(await context.CallActivityAsync<string>(nameof(SayHello), "Seattle"));
        outputs.Add(await context.CallActivityAsync<string>(nameof(SayHello), "London"));

        // returns ["Hello Tokyo!", "Hello Seattle!", "Hello London!"]
        return outputs;
    }
}

[DurableTask]
public class SayHello(ILogger<SayHello> logger) : TaskActivity<string, string>
{
    public override Task<string> RunAsync(TaskActivityContext context, string name)
    {
        logger.LogInformation("Saying hello to {name}.", name);
        return Task.FromResult($"Hello {name}!");
    }
}
