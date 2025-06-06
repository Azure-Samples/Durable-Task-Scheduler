using DurableFunctionsSaga.Models;
using Microsoft.Azure.Functions.Worker;
using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;
using System;

namespace DurableFunctionsSaga.Activities
{
    /// <summary>
    /// Payment-related activities
    /// </summary>
    public class PaymentActivities
    {
        private readonly ILogger<PaymentActivities> _logger;

        public PaymentActivities(ILogger<PaymentActivities> logger)
        {
            _logger = logger;
        }

        [Function(nameof(ProcessPaymentActivity))]
        public Payment ProcessPaymentActivity([ActivityTrigger] Payment payment)
        {
            _logger.LogInformation("Processing payment for order {OrderId}, amount {Amount}", 
                payment.OrderId, payment.Amount);
            
            // Simulate payment processing
            payment.IsProcessed = true;
            payment.TransactionId = Guid.NewGuid().ToString();
            
            return payment;
        }

        [Function(nameof(RefundPaymentActivity))]
        public Payment RefundPaymentActivity([ActivityTrigger] Payment payment)
        {
            _logger.LogInformation("Compensation: Refunding payment for order {OrderId}, amount {Amount}, transaction {TransactionId}", 
                payment.OrderId, payment.Amount, payment.TransactionId);
            
            // Simulate payment refund
            payment.IsProcessed = false;
            
            return payment;
        }
    }
}
