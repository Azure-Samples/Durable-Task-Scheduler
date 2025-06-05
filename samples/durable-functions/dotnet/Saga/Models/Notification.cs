namespace DurableFunctionsSaga.Models
{
    public class Notification
    {
        public string OrderId { get; set; } = string.Empty;
        public string Message { get; set; } = string.Empty;
        public string Status { get; set; } = string.Empty;
    }
}
