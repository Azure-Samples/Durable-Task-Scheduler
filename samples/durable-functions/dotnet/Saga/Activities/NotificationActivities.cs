using DurableFunctionsSaga.Models;
using Microsoft.Azure.Functions.Worker;
using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;
using System;
using System.Threading.Tasks;

namespace DurableFunctionsSaga.Activities
{
    public class NotificationActivities
    {
        private readonly ILogger<NotificationActivities> _logger;

        public NotificationActivities(ILogger<NotificationActivities> logger)
        {
            _logger = logger;
        }

        [Function(nameof(NotifyActivity))]
        public Task<Notification> NotifyActivity([ActivityTrigger] Notification notification, FunctionContext executionContext)
        {
            _logger.LogInformation("Sending notification: {Message}", notification.Message);
            
            // Simulate sending a notification
            notification.Status = "Sent";
            
            return Task.FromResult(notification);
        }
    }
}
