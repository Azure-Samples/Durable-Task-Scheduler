using Microsoft.Azure.Functions.Worker;
using Microsoft.Azure.Functions.Worker.Http;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.Extensions.Logging;

namespace DistributedTracing;

public static class Functions
{
    [Function("StartOrchestration")]
    public static async Task<HttpResponseData> HttpStart(
        [HttpTrigger(AuthorizationLevel.Anonymous, "post")] HttpRequestData req,
        [DurableClient] DurableTaskClient client,
        FunctionContext executionContext)
    {
        var logger = executionContext.GetLogger("StartOrchestration");

        string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
            nameof(OrderOrchestration), "Order-12345");

        logger.LogInformation("Started orchestration with ID = '{instanceId}'.", instanceId);

        return await client.CreateCheckStatusResponseAsync(req, instanceId);
    }

    [Function(nameof(OrderOrchestration))]
    public static async Task<string> OrderOrchestration(
        [OrchestrationTrigger] TaskOrchestrationContext context)
    {
        var logger = context.CreateReplaySafeLogger(nameof(OrderOrchestration));

        logger.LogInformation("Starting order processing orchestration");

        var validated = await context.CallActivityAsync<string>(nameof(ValidateOrder), "Order-12345");
        var paid = await context.CallActivityAsync<string>(nameof(ProcessPayment), validated);
        var shipped = await context.CallActivityAsync<string>(nameof(ShipOrder), paid);
        var result = await context.CallActivityAsync<string>(nameof(SendNotification), shipped);

        logger.LogInformation("Order processing completed: {Result}", result);
        return result;
    }

    [Function(nameof(ValidateOrder))]
    public static string ValidateOrder(
        [ActivityTrigger] string orderId, FunctionContext context)
    {
        var logger = context.GetLogger(nameof(ValidateOrder));
        logger.LogInformation("Validating order: {OrderId}", orderId);
        return $"Validated({orderId})";
    }

    [Function(nameof(ProcessPayment))]
    public static string ProcessPayment(
        [ActivityTrigger] string input, FunctionContext context)
    {
        var logger = context.GetLogger(nameof(ProcessPayment));
        logger.LogInformation("Processing payment for: {Input}", input);
        return $"Paid({input})";
    }

    [Function(nameof(ShipOrder))]
    public static string ShipOrder(
        [ActivityTrigger] string input, FunctionContext context)
    {
        var logger = context.GetLogger(nameof(ShipOrder));
        logger.LogInformation("Shipping: {Input}", input);
        return $"Shipped({input})";
    }

    [Function(nameof(SendNotification))]
    public static string SendNotification(
        [ActivityTrigger] string input, FunctionContext context)
    {
        var logger = context.GetLogger(nameof(SendNotification));
        logger.LogInformation("Notifying: {Input}", input);
        return $"Notified({input})";
    }
}
