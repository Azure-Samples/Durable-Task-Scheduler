using DurableFunctionsSaga.Models;
using DurableFunctionsSaga.Saga;
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
            _logger = logger ?? throw new ArgumentNullException(nameof(logger));
        }

        /// <summary>
        /// Process an order using the SAGA pattern with compensations
        /// </summary>
        [Function(nameof(ProcessOrder))]
        public async Task<Order> ProcessOrder([OrchestrationTrigger] TaskOrchestrationContext context)
        {
            var order = context.GetInput<Order>() ?? throw new ArgumentNullException(nameof(Order));
            _logger.LogInformation("Starting order processing saga with ID: {OrchestrationId}", context.InstanceId);
            
            // Create compensations tracker
            var compensations = new Compensations(context, _logger);

            try
            {
                // Step 1: Send order notification (no compensation needed)
                var notification = new Notification
                {
                    OrderId = order.OrderId,
                    Message = $"Processing order {order.OrderId}"
                };
                await context.CallActivityAsync("NotifyActivity", notification);
                
                // Step 2: Reserve inventory
                var inventory = new Inventory
                {
                    ProductId = order.ProductId,
                    ReservedQuantity = order.Quantity
                };
                
                // Register compensation BEFORE performing the operation
                compensations.AddCompensation("ReleaseInventoryActivity", inventory);
                inventory = await context.CallActivityAsync<Inventory>("ReserveInventoryActivity", inventory);
                
                // Step 3: Request approval (no compensation needed)
                var approval = new Approval
                {
                    OrderId = order.OrderId,
                    IsApproved = false
                };
                approval = await context.CallActivityAsync<Approval>("RequestApprovalActivity", approval);
                
                if (!approval.IsApproved)
                {
                    // If not approved, run compensations and return
                    await compensations.CompensateAsync();
                    order.Status = "Cancelled - Not Approved";
                    return order;
                }
                
                // Step 4: Process payment
                var payment = new Payment
                {
                    OrderId = order.OrderId,
                    Amount = order.Amount
                };
                
                // Register compensation BEFORE processing payment
                compensations.AddCompensation("RefundPaymentActivity", payment);
                payment = await context.CallActivityAsync<Payment>("ProcessPaymentActivity", payment);
                
                // Step 5: Update inventory (convert reserved to confirmed)
                compensations.AddCompensation("RestoreInventoryActivity", inventory);
                inventory = await context.CallActivityAsync<Inventory>("UpdateInventoryActivity", inventory);
                
                // Step 6: Process delivery with retry handling
                var delivery = new Delivery
                {
                    OrderId = order.OrderId,
                    Address = $"Customer address for {order.CustomerId}"
                };
                
                // Process delivery - no retries, will directly trigger compensation when it fails
                delivery = await context.CallActivityAsync<Delivery>("DeliveryActivity", delivery);
                
                // All operations completed successfully
                notification = new Notification
                {
                    OrderId = order.OrderId,
                    Message = $"Order {order.OrderId} status: Completed"
                };
                await context.CallActivityAsync("NotifyActivity", notification);
                
                order.Status = "Completed";
                return order;
            }
            catch (Exception ex)
            {
                // An error occurred somewhere - run all compensations
                _logger.LogError(ex, "Error in order processing saga. Running compensations. Order: {OrderId}", order.OrderId);
                
                // Run all compensations in LIFO order
                await compensations.CompensateAsync();
                
                // Send failure notification
                var failureNotification = new Notification
                {
                    OrderId = order.OrderId,
                    Message = $"Order {order.OrderId} processing failed: {ex.Message}"
                };
                await context.CallActivityAsync("NotifyActivity", failureNotification);
                
                order.Status = $"Failed - {ex.GetType().Name}";
                return order;
            }
        }
    }
}
