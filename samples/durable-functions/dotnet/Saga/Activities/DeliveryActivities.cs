using DurableFunctionsSaga.Models;
using Microsoft.Azure.Functions.Worker;
using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;
using System;
using System.Threading.Tasks;

namespace DurableFunctionsSaga.Activities
{
    public class DeliveryActivities
    {
        private readonly ILogger<DeliveryActivities> _logger;

        public DeliveryActivities(ILogger<DeliveryActivities> logger)
        {
            _logger = logger;
        }

        [Function(nameof(DeliveryActivity))]
        public Task<Delivery> DeliveryActivity([ActivityTrigger] Delivery delivery, FunctionContext executionContext)
        {
            _logger.LogInformation("Scheduling delivery for order {OrderId} to address: {Address}", 
                delivery.OrderId, delivery.Address);
            
            // Intentionally fail to demonstrate compensation pattern
            if (DateTime.UtcNow.Second % 2 == 0) // Fail 50% of the time
            {
                _logger.LogError("Delivery service is currently unavailable for order {OrderId}", delivery.OrderId);
                throw new Exception("Delivery service is currently unavailable");
            }
            
            // This code will only execute if the random failure doesn't occur
            delivery.Status = "Scheduled";
            
            return Task.FromResult(delivery);
        }
    }
}
