using DurableFunctionsSaga.Models;
using Microsoft.Azure.Functions.Worker;
using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;
using System;
using System.Threading.Tasks;

namespace DurableFunctionsSaga.Orchestrators
{
    public class OrderProcessingOrchestrator
    {
        private readonly ILogger<OrderProcessingOrchestrator> _logger;

        public OrderProcessingOrchestrator(ILogger<OrderProcessingOrchestrator> logger)
        {
            _logger = logger;
        }

        [Function(nameof(ProcessOrder))]
        public async Task<Order> ProcessOrder([OrchestrationTrigger] TaskOrchestrationContext context)
        {
            var order = context.GetInput<Order>() ?? throw new ArgumentNullException(nameof(Order));
            var orchestrationId = context.InstanceId;

            _logger.LogInformation("Starting order processing saga with ID: {OrchestrationId}", orchestrationId);

            try
            {
                // Step 1: Send order notification
                var notification = new Notification
                {
                    OrderId = order.OrderId,
                    Message = $"Processing order {order.OrderId}"
                };
                notification = await context.CallActivityAsync<Notification>("NotifyActivity", notification);
                
                // Step 2: Reserve inventory
                var inventory = new Inventory
                {
                    ProductId = order.ProductId,
                    ReservedQuantity = order.Quantity
                };
                inventory = await context.CallActivityAsync<Inventory>("ReserveInventoryActivity", inventory);
                
                // Step 3: Request order approval
                var approval = new Approval
                {
                    OrderId = order.OrderId,
                    IsApproved = false
                };
                approval = await context.CallActivityAsync<Approval>("RequestApprovalActivity", approval);
                
                if (!approval.IsApproved)
                {
                    // If not approved, cancel the order and release the inventory
                    await context.CallActivityAsync("ReleaseInventoryActivity", inventory);
                    order.Status = "Cancelled - Not Approved";
                    return order;
                }
                
                // Step 4: Process payment
                var payment = new Payment
                {
                    OrderId = order.OrderId,
                    Amount = order.Amount
                };
                
                try
                {
                    payment = await context.CallActivityAsync<Payment>("ProcessPaymentActivity", payment);
                }
                catch (Exception ex)
                {
                    // Compensation: Refund payment if needed, then release inventory
                    _logger.LogError(ex, "Payment processing failed. Initiating compensation.");
                    await context.CallActivityAsync("RefundPaymentActivity", payment);
                    await context.CallActivityAsync("ReleaseInventoryActivity", inventory);
                    order.Status = "Failed - Payment Error";
                    return order;
                }
                
                // Step 5: Update inventory (convert reserved to confirmed)
                try
                {
                    inventory = await context.CallActivityAsync<Inventory>("UpdateInventoryActivity", inventory);
                }
                catch (Exception ex)
                {
                    // Compensation: Refund payment and release inventory
                    _logger.LogError(ex, "Inventory update failed. Initiating compensation.");
                    await context.CallActivityAsync("RefundPaymentActivity", payment);
                    await context.CallActivityAsync("ReleaseInventoryActivity", inventory);
                    order.Status = "Failed - Inventory Error";
                    return order;
                }
                
                // Step 6: Process delivery (this will intentionally fail to demonstrate compensation)
                var delivery = new Delivery
                {
                    OrderId = order.OrderId,
                    Address = $"Customer address for {order.CustomerId}"
                };
                
                try
                {
                    delivery = await context.CallActivityAsync<Delivery>("DeliveryActivity", delivery);
                    order.Status = "Completed";
                }
                catch (Exception ex)
                {
                    // Full compensation: Refund payment, restore inventory
                    _logger.LogError(ex, "Delivery failed. Initiating full compensation.");
                    await context.CallActivityAsync("RefundPaymentActivity", payment);
                    await context.CallActivityAsync("RestoreInventoryActivity", inventory);
                    order.Status = "Failed - Delivery Error";
                }
                
                // Send final notification
                notification = new Notification
                {
                    OrderId = order.OrderId,
                    Message = $"Order {order.OrderId} status: {order.Status}"
                };
                await context.CallActivityAsync("NotifyActivity", notification);
                
                return order;
            }
            catch (Exception ex)
            {
                // Handle any unhandled exceptions
                _logger.LogError(ex, "Unhandled exception in order processing saga. Order: {OrderId}", order.OrderId);
                order.Status = "Failed - System Error";
                return order;
            }
        }
    }
}
