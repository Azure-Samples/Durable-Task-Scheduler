using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;

namespace SubOrchestrations;

[DurableTask(nameof(OrderOrchestration))]
public class OrderOrchestration : TaskOrchestrator<Order, OrderResult>
{
    public override async Task<OrderResult> RunAsync(TaskOrchestrationContext context, Order order)
    {
        ILogger logger = context.CreateReplaySafeLogger<OrderOrchestration>();
        logger.LogInformation("Processing order '{OrderId}' with {Count} items.", order.OrderId, order.Items.Length);

        var lineResults = new List<LineItemResult>();

        foreach (var item in order.Items)
        {
            // Each line item is processed by a sub-orchestration
            var result = await context.CallSubOrchestratorAsync<LineItemResult>(
                nameof(LineItemOrchestration),
                new LineItemInput(order.OrderId, item));

            lineResults.Add(result);
            logger.LogInformation("Item '{Item}': {Status}", item.ProductName, result.Status);
        }

        decimal totalPrice = lineResults.Where(r => r.IsSuccess).Sum(r => r.Price);
        int successCount = lineResults.Count(r => r.IsSuccess);

        logger.LogInformation("Order '{OrderId}' complete: {Success}/{Total} items, total ${Price:F2}",
            order.OrderId, successCount, lineResults.Count, totalPrice);

        return new OrderResult(order.OrderId, lineResults.ToArray(), totalPrice);
    }
}

[DurableTask(nameof(LineItemOrchestration))]
public class LineItemOrchestration : TaskOrchestrator<LineItemInput, LineItemResult>
{
    public override async Task<LineItemResult> RunAsync(TaskOrchestrationContext context, LineItemInput input)
    {
        ILogger logger = context.CreateReplaySafeLogger<LineItemOrchestration>();
        logger.LogInformation("Processing line item: {Product} x{Qty}", input.Item.ProductName, input.Item.Quantity);

        // Step 1: Validate the item
        bool isValid = await context.CallActivityAsync<bool>(nameof(ValidateItemActivity), input.Item);
        if (!isValid)
        {
            return new LineItemResult(input.Item.ProductName, false, 0, "Validation failed");
        }

        // Step 2: Calculate price
        decimal price = await context.CallActivityAsync<decimal>(nameof(CalculatePriceActivity), input.Item);

        // Step 3: Reserve inventory
        bool reserved = await context.CallActivityAsync<bool>(nameof(ReserveInventoryActivity), input.Item);
        if (!reserved)
        {
            return new LineItemResult(input.Item.ProductName, false, 0, "Out of stock");
        }

        return new LineItemResult(input.Item.ProductName, true, price, "Confirmed");
    }
}

// Data models
public record Order(string OrderId, LineItem[] Items);
public record LineItem(string ProductName, int Quantity, decimal UnitPrice);
public record LineItemInput(string OrderId, LineItem Item);
public record LineItemResult(string ProductName, bool IsSuccess, decimal Price, string Status);
public record OrderResult(string OrderId, LineItemResult[] Items, decimal TotalPrice);
