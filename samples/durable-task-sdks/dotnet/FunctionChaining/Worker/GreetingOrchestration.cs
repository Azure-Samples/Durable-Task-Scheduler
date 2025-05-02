using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;

namespace FunctionChaining;

[DurableTask]
public class GreetingOrchestration : TaskOrchestrator<string, string>
{
    public override async Task<string> RunAsync(TaskOrchestrationContext context, string name)
    {
        // Step 1: Say hello to the person
        string greeting = await context.CallActivityAsync<string>(nameof(SayHelloActivity), name);
        
        // Step 2: Process the greeting
        string processedGreeting = await context.CallActivityAsync<string>(nameof(ProcessGreetingActivity), greeting);
        
        // Step 3: Finalize the response
        string finalResponse = await context.CallActivityAsync<string>(nameof(FinalizeResponseActivity), processedGreeting);
        
        return finalResponse;
    }
}

[DurableTask]
public class SayHelloActivity : TaskActivity<string, string>
{
    private readonly ILogger<SayHelloActivity> _logger;

    public SayHelloActivity(ILogger<SayHelloActivity> logger)
    {
        _logger = logger;
    }

    public override Task<string> RunAsync(TaskActivityContext context, string name)
    {
        _logger.LogInformation("Activity SayHello called with name: {Name}", name);
        
        // First activity that greets the user
        string result = $"Hello {name}!";
        
        return Task.FromResult(result);
    }
}

[DurableTask]
public class ProcessGreetingActivity : TaskActivity<string, string>
{
    private readonly ILogger<ProcessGreetingActivity> _logger;

    public ProcessGreetingActivity(ILogger<ProcessGreetingActivity> logger)
    {
        _logger = logger;
    }

    public override Task<string> RunAsync(TaskActivityContext context, string greeting)
    {
        _logger.LogInformation("Activity ProcessGreeting called with greeting: {Greeting}", greeting);
        
        // Second activity that processes the greeting
        string result = $"{greeting} How are you today?";
        
        return Task.FromResult(result);
    }
}

[DurableTask]
public class FinalizeResponseActivity : TaskActivity<string, string>
{
    private readonly ILogger<FinalizeResponseActivity> _logger;

    public FinalizeResponseActivity(ILogger<FinalizeResponseActivity> logger)
    {
        _logger = logger;
    }

    public override Task<string> RunAsync(TaskActivityContext context, string response)
    {
        _logger.LogInformation("Activity FinalizeResponse called with response: {Response}", response);
        
        // Third activity that finalizes the response
        string result = $"{response} I hope you're doing well!";
        
        return Task.FromResult(result);
    }
}
