using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;

namespace ShipperWorker;

/// <summary>
/// Ships an order. This activity is registered only in Worker B,
/// so DTS will route ShipOrder work items exclusively to Worker B.
/// </summary>
[DurableTask(nameof(ShipOrder))]
public class ShipOrder : TaskActivity<string, string>
{
    readonly ILogger<ShipOrder> logger;

    public ShipOrder(ILoggerFactory loggerFactory)
    {
        this.logger = loggerFactory.CreateLogger<ShipOrder>();
    }

    public override Task<string> RunAsync(TaskActivityContext context, string orderId)
    {
        this.logger.LogInformation(
            "[Shipper] Activity | Name=ShipOrder | InstanceId={InstanceId} | Shipping order '{OrderId}'...",
            context.InstanceId, orderId);

        // Simulate shipping
        string trackingNumber = $"TRACK-{orderId}-{Random.Shared.Next(1000, 9999)}";
        string result = $"Shipped with tracking {trackingNumber}";

        this.logger.LogInformation(
            "[Shipper] Activity | Name=ShipOrder | InstanceId={InstanceId} | Result: {Result}",
            context.InstanceId, result);

        return Task.FromResult(result);
    }
}
