using DurableFunctionsSaga.Models;
using Microsoft.Azure.Functions.Worker;
using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;
using System;

namespace DurableFunctionsSaga.Activities
{
    /// <summary>
    /// Delivery-related activities
    /// </summary>
    public class DeliveryActivities
    {
        private readonly ILogger<DeliveryActivities> _logger;
        private readonly Random _random = new Random();

        public DeliveryActivities(ILogger<DeliveryActivities> logger)
        {
            _logger = logger;
        }

        [Function(nameof(DeliveryActivity))]
        public Delivery DeliveryActivity([ActivityTrigger] Delivery delivery)
        {
            _logger.LogInformation("Scheduling delivery for order {OrderId} to address {Address}", 
                delivery.OrderId, delivery.Address);
            
            // Simulate a failure 50% of the time to demonstrate compensation
            if (_random.Next(2) == 0)
            {
                _logger.LogError("Failed to schedule delivery for order {OrderId}", delivery.OrderId);
                throw new Exception("Delivery service unavailable - Simulated failure to demonstrate compensation");
            }
            
            // Simulate successful delivery scheduling
            delivery.IsScheduled = true;
            delivery.TrackingNumber = Guid.NewGuid().ToString().Substring(0, 8).ToUpper();
            
            return delivery;
        }
    }
}
