using System.Net;
using Microsoft.Azure.Functions.Worker;
using Microsoft.Azure.Functions.Worker.Http;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;

namespace TaskQueues;

public static class Functions
{
    [Function(nameof(StartOrder))]
    public static async Task<HttpResponseData> StartOrder(
        [HttpTrigger(AuthorizationLevel.Function, "post", Route = "orders/{orderId}")] HttpRequestData request,
        [DurableClient] DurableTaskClient client, string orderId)
    {
        if (string.IsNullOrWhiteSpace(orderId))
        {
            HttpResponseData badRequest = request.CreateResponse(HttpStatusCode.BadRequest);
            await badRequest.WriteStringAsync("Order ID must not be empty.");
            return badRequest;
        }
        string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
            nameof(ProcessOrder), orderId,
            new StartOrchestrationOptions { Tags = new Dictionary<string, string> { ["DTS_TaskQueue"] = "orders" } });
        return client.CreateCheckStatusResponse(request, instanceId);
    }

    [Function(nameof(ProcessOrder))]
    public static async Task<string> ProcessOrder([OrchestrationTrigger] TaskOrchestrationContext context)
    {
        string orderId = context.GetInput<string>() ?? throw new ArgumentException("Order ID is required.");
        string recorded = await context.CallActivityAsync<string>(nameof(RecordOrder), orderId);
        string partner = await context.CallActivityAsync<string>(
            nameof(CallPartner), orderId,
            new TaskOptions(tags: new Dictionary<string, string> { ["DTS_TaskQueue"] = "partner-api" }));
        string validated = await context.CallSubOrchestratorAsync<string>(
            nameof(ValidateOrder), orderId,
            new SubOrchestrationOptions { Tags = new Dictionary<string, string> { ["DTS_TaskQueue"] = "partner-api" } });
        return $"{recorded}; {partner}; {validated}";
    }

    [Function(nameof(ValidateOrder))]
    public static async Task<string> ValidateOrder([OrchestrationTrigger] TaskOrchestrationContext context)
    {
        string orderId = context.GetInput<string>() ?? throw new ArgumentException("Order ID is required.");
        await context.CallActivityAsync<string>(nameof(RecordOrder), orderId);
        return $"Validated {orderId}";
    }

    [Function(nameof(RecordOrder))]
    public static string RecordOrder([ActivityTrigger] string orderId) => $"Recorded {orderId}";

    [Function(nameof(CallPartner))]
    public static string CallPartner([ActivityTrigger] string orderId) => $"Partner accepted {orderId}";
}
