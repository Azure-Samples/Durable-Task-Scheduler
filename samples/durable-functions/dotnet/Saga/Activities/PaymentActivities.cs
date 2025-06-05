using DurableFunctionsSaga.Models;
using Microsoft.Azure.Functions.Worker;
using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;
using System;
using System.Threading.Tasks;

namespace DurableFunctionsSaga.Activities
{
    public class PaymentActivities
    {
        private readonly ILogger<PaymentActivities> _logger;

        public PaymentActivities(ILogger<PaymentActivities> logger)
        {
            _logger = logger;
        }

        [Function(nameof(ProcessPaymentActivity))]
        public Task<Payment> ProcessPaymentActivity([ActivityTrigger] Payment payment, FunctionContext executionContext)
        {
            _logger.LogInformation("Processing payment of {Amount:C} for order {OrderId}", 
                payment.Amount, payment.OrderId);
            
            // Simulate payment processing 
            payment.PaymentId = Guid.NewGuid().ToString();
            payment.Status = "Completed";
            
            return Task.FromResult(payment);
        }

        [Function(nameof(RefundPaymentActivity))]
        public Task RefundPaymentActivity([ActivityTrigger] Payment payment, FunctionContext executionContext)
        {
            if (string.IsNullOrEmpty(payment.PaymentId))
            {
                _logger.LogInformation("No payment to refund for order {OrderId}", payment.OrderId);
                return Task.CompletedTask;
            }
            
            _logger.LogInformation("Refunding payment {PaymentId} of {Amount:C} for order {OrderId}", 
                payment.PaymentId, payment.Amount, payment.OrderId);
            
            // Simulate refund processing
            payment.Status = "Refunded";
            
            return Task.CompletedTask;
        }
    }
}
