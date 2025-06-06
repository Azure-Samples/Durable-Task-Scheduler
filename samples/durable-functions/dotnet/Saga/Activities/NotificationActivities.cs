using DurableFunctionsSaga.Models;
using Microsoft.Azure.Functions.Worker;
using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;

namespace DurableFunctionsSaga.Activities
{
    /// <summary>
    /// Notification-related activities
    /// </summary>
    public class NotificationActivities
    {
        private readonly ILogger<NotificationActivities> _logger;

        public NotificationActivities(ILogger<NotificationActivities> logger)
        {
            _logger = logger;
        }

        [Function(nameof(NotifyActivity))]
        public Notification NotifyActivity([ActivityTrigger] Notification notification)
        {
            _logger.LogInformation("Sending notification for order {OrderId}: {Message}", 
                notification.OrderId, notification.Message);
            
            // Simulate notification sending
            return notification;
        }
    }
}
