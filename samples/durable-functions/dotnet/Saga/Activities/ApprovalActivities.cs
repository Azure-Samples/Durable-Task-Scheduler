using DurableFunctionsSaga.Models;
using Microsoft.Azure.Functions.Worker;
using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;
using System;

namespace DurableFunctionsSaga.Activities
{
    /// <summary>
    /// Approval-related activities
    /// </summary>
    public class ApprovalActivities
    {
        private readonly ILogger<ApprovalActivities> _logger;
        private readonly Random _random = new Random();

        public ApprovalActivities(ILogger<ApprovalActivities> logger)
        {
            _logger = logger;
        }

        [Function(nameof(RequestApprovalActivity))]
        public Approval RequestApprovalActivity([ActivityTrigger] Approval approval)
        {
            _logger.LogInformation("Requesting approval for order {OrderId}", approval.OrderId);
            
            // Simulate approval process (approve 75% of orders)
            approval.IsApproved = _random.Next(4) != 0;
            
            if (approval.IsApproved)
            {
                approval.ApprovalId = Guid.NewGuid().ToString();
                _logger.LogInformation("Order {OrderId} approved with approval ID {ApprovalId}", 
                    approval.OrderId, approval.ApprovalId);
            }
            else
            {
                _logger.LogInformation("Order {OrderId} rejected", approval.OrderId);
            }
            
            return approval;
        }
    }
}
