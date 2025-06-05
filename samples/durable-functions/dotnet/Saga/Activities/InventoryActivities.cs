using DurableFunctionsSaga.Models;
using Microsoft.Azure.Functions.Worker;
using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;
using System;
using System.Threading.Tasks;

namespace DurableFunctionsSaga.Activities
{
    public class InventoryActivities
    {
        private readonly ILogger<InventoryActivities> _logger;
        private static int _availableStock = 100; // Simulate inventory database

        public InventoryActivities(ILogger<InventoryActivities> logger)
        {
            _logger = logger;
        }

        [Function(nameof(ReserveInventoryActivity))]
        public Task<Inventory> ReserveInventoryActivity([ActivityTrigger] Inventory inventory, FunctionContext executionContext)
        {
            if (inventory.ReservedQuantity > _availableStock)
            {
                _logger.LogError("Insufficient inventory. Available: {Available}, Requested: {Requested}", 
                    _availableStock, inventory.ReservedQuantity);
                throw new InvalidOperationException($"Insufficient inventory for product {inventory.ProductId}");
            }

            _logger.LogInformation("Reserving {Quantity} units of product {ProductId}", 
                inventory.ReservedQuantity, inventory.ProductId);
            
            // Update available stock
            _availableStock -= inventory.ReservedQuantity;
            inventory.AvailableQuantity = _availableStock;

            return Task.FromResult(inventory);
        }

        [Function(nameof(UpdateInventoryActivity))]
        public Task<Inventory> UpdateInventoryActivity([ActivityTrigger] Inventory inventory, FunctionContext executionContext)
        {
            _logger.LogInformation("Confirming reservation of {Quantity} units of product {ProductId}", 
                inventory.ReservedQuantity, inventory.ProductId);
            
            // In a real system, this would convert reserved inventory to consumed inventory
            
            return Task.FromResult(inventory);
        }

        [Function(nameof(ReleaseInventoryActivity))]
        public Task ReleaseInventoryActivity([ActivityTrigger] Inventory inventory, FunctionContext executionContext)
        {
            _logger.LogInformation("Releasing reservation of {Quantity} units of product {ProductId}", 
                inventory.ReservedQuantity, inventory.ProductId);
            
            // Return the reserved quantity back to available stock
            _availableStock += inventory.ReservedQuantity;
            
            return Task.CompletedTask;
        }

        [Function(nameof(RestoreInventoryActivity))]
        public Task RestoreInventoryActivity([ActivityTrigger] Inventory inventory, FunctionContext executionContext)
        {
            _logger.LogInformation("Restoring {Quantity} units of product {ProductId} to inventory", 
                inventory.ReservedQuantity, inventory.ProductId);
            
            // Add the quantity back to available stock
            _availableStock += inventory.ReservedQuantity;
            
            return Task.CompletedTask;
        }
    }
}
