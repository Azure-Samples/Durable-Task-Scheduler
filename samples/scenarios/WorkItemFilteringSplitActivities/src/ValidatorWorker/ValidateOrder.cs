using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;

namespace ValidatorWorker;

/// <summary>
/// Validates an incoming order. This activity is registered only in Worker A,
/// so DTS will route ValidateOrder work items exclusively to Worker A.
/// </summary>
[DurableTask(nameof(ValidateOrder))]
public class ValidateOrder : TaskActivity<string, string>
{
    readonly ILogger<ValidateOrder> logger;

    public ValidateOrder(ILoggerFactory loggerFactory)
    {
        this.logger = loggerFactory.CreateLogger<ValidateOrder>();
    }

    public override Task<string> RunAsync(TaskActivityContext context, string orderId)
    {
        this.logger.LogInformation(
            "[Validator] Activity | Name=ValidateOrder | InstanceId={InstanceId} | Validating order '{OrderId}'...",
            context.InstanceId, orderId);

        // Simulate validation
        string result = $"Order {orderId} is valid";

        this.logger.LogInformation(
            "[Validator] Activity | Name=ValidateOrder | InstanceId={InstanceId} | Result: {Result}",
            context.InstanceId, result);

        return Task.FromResult(result);
    }
}
