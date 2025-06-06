using DurableFunctionsSaga.Models;
using Microsoft.Azure.Functions.Worker;
using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;
using System;
using System.Threading.Tasks;

namespace DurableFunctionsSaga.Activities
{
    /// <summary>
    /// Inventory-related activities
    /// </summary>
    public class InventoryActivities
    {
        private readonly ILogger<InventoryActivities> _logger;

        public InventoryActivities(ILogger<InventoryActivities> logger)
        {
            _logger = logger;
        }

        [Function(nameof(ReserveInventoryActivity))]
        public Inventory ReserveInventoryActivity([ActivityTrigger] Inventory inventory)
        {
            _logger.LogInformation("Reserving inventory for product {ProductId}, quantity {Quantity}", 
                inventory.ProductId, inventory.ReservedQuantity);
            
            // Simulate inventory reservation
            inventory.Reserved = true;
            
            return inventory;
        }

        [Function(nameof(UpdateInventoryActivity))]
        public Inventory UpdateInventoryActivity([ActivityTrigger] Inventory inventory)
        {
            _logger.LogInformation("Updating inventory for product {ProductId}, quantity {Quantity} from reserved to confirmed", 
                inventory.ProductId, inventory.ReservedQuantity);
            
            // Simulate inventory update
            inventory.Updated = true;
            
            return inventory;
        }

        [Function(nameof(ReleaseInventoryActivity))]
        public Inventory ReleaseInventoryActivity([ActivityTrigger] Inventory inventory)
        {
            _logger.LogInformation("Compensation: Releasing reserved inventory for product {ProductId}, quantity {Quantity}", 
                inventory.ProductId, inventory.ReservedQuantity);
            
            // Simulate releasing inventory
            inventory.Reserved = false;
            
            return inventory;
        }

        [Function(nameof(RestoreInventoryActivity))]
        public Inventory RestoreInventoryActivity([ActivityTrigger] Inventory inventory)
        {
            _logger.LogInformation("Compensation: Restoring inventory for product {ProductId}, quantity {Quantity} to reserved state", 
                inventory.ProductId, inventory.ReservedQuantity);
            
            // Simulate restoring inventory from confirmed back to reserved
            inventory.Updated = false;
            inventory.Reserved = true;
            
            return inventory;
        }
    }
}
