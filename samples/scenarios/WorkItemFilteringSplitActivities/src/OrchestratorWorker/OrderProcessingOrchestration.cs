using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;

namespace OrchestratorWorker;

/// <summary>
/// A simple orchestration that calls two activities sequentially:
///   1. ValidateOrder  (handled by Worker A)
///   2. ShipOrder      (handled by Worker B)
///
/// Because each activity is registered in a different worker process, DTS routes
/// each activity work item to the correct worker via work item filtering.
/// </summary>
[DurableTask(nameof(OrderProcessingOrchestration))]
public class OrderProcessingOrchestration : TaskOrchestrator<string, string>
{
    public override async Task<string> RunAsync(TaskOrchestrationContext context, string orderId)
    {
        ILogger logger = context.CreateReplaySafeLogger<OrderProcessingOrchestration>();

        logger.LogInformation(
            "[Orchestrator] Orchestration | Name=OrderProcessingOrchestration | InstanceId={InstanceId} | Processing order '{OrderId}'",
            context.InstanceId, orderId);

        // Step 1: Validate the order (routed to Validator Worker)
        logger.LogInformation(
            "[Orchestrator] Orchestration | InstanceId={InstanceId} | Dispatching ValidateOrder to Validator Worker...",
            context.InstanceId);
        string validationResult = await context.CallActivityAsync<string>("ValidateOrder", orderId);

        // Step 2: Ship the order (routed to Shipper Worker)
        logger.LogInformation(
            "[Orchestrator] Orchestration | InstanceId={InstanceId} | Dispatching ShipOrder to Shipper Worker...",
            context.InstanceId);
        string shippingResult = await context.CallActivityAsync<string>("ShipOrder", orderId);

        string combined = $"Order '{orderId}' => Validation: [{validationResult}], Shipping: [{shippingResult}]";

        logger.LogInformation(
            "[Orchestrator] Orchestration | InstanceId={InstanceId} | Completed: {Result}",
            context.InstanceId, combined);

        return combined;
    }
}
