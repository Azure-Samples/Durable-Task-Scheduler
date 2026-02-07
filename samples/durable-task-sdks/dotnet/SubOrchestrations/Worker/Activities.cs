using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;

namespace SubOrchestrations;

[DurableTask(nameof(ValidateItemActivity))]
public class ValidateItemActivity : TaskActivity<LineItem, bool>
{
    readonly ILogger<ValidateItemActivity> logger;
    public ValidateItemActivity(ILogger<ValidateItemActivity> logger) => this.logger = logger;

    public override Task<bool> RunAsync(TaskActivityContext context, LineItem item)
    {
        bool isValid = item.Quantity > 0 && item.UnitPrice > 0;
        this.logger.LogInformation("Validating '{Product}': {Result}", item.ProductName, isValid ? "Valid" : "Invalid");
        return Task.FromResult(isValid);
    }
}

[DurableTask(nameof(CalculatePriceActivity))]
public class CalculatePriceActivity : TaskActivity<LineItem, decimal>
{
    readonly ILogger<CalculatePriceActivity> logger;
    public CalculatePriceActivity(ILogger<CalculatePriceActivity> logger) => this.logger = logger;

    public override Task<decimal> RunAsync(TaskActivityContext context, LineItem item)
    {
        decimal total = item.Quantity * item.UnitPrice;
        this.logger.LogInformation("Price for '{Product}': {Qty} x ${Unit:F2} = ${Total:F2}",
            item.ProductName, item.Quantity, item.UnitPrice, total);
        return Task.FromResult(total);
    }
}

[DurableTask(nameof(ReserveInventoryActivity))]
public class ReserveInventoryActivity : TaskActivity<LineItem, bool>
{
    readonly ILogger<ReserveInventoryActivity> logger;
    public ReserveInventoryActivity(ILogger<ReserveInventoryActivity> logger) => this.logger = logger;

    public override Task<bool> RunAsync(TaskActivityContext context, LineItem item)
    {
        // Simulate: items with quantity > 100 are out of stock
        bool reserved = item.Quantity <= 100;
        this.logger.LogInformation("Inventory for '{Product}': {Result}",
            item.ProductName, reserved ? "Reserved" : "Out of stock");
        return Task.FromResult(reserved);
    }
}
