using DurableFunctionsSaga.Models;
using Microsoft.Azure.Functions.Worker;
using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;
using System;
using System.Threading.Tasks;

namespace DurableFunctionsSaga.Activities
{
    public class ApprovalActivities
    {
        private readonly ILogger<ApprovalActivities> _logger;

        public ApprovalActivities(ILogger<ApprovalActivities> logger)
        {
            _logger = logger;
        }

        [Function(nameof(RequestApprovalActivity))]
        public Task<Approval> RequestApprovalActivity([ActivityTrigger] Approval approval, FunctionContext executionContext)
        {
            _logger.LogInformation("Requesting approval for order {OrderId}", approval.OrderId);
            
            // Simulate approval process
            // In a real-world scenario, this might involve human interaction or calling an external approval service
            
            // Auto-approve for this demo
            approval.IsApproved = true;
            
            _logger.LogInformation("Order {OrderId} has been {Status}", 
                approval.OrderId, approval.IsApproved ? "approved" : "rejected");
            
            return Task.FromResult(approval);
        }
    }
}
