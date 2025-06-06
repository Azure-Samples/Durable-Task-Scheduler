namespace DurableFunctionsSaga.Models
{
    public class Notification
    {
        public required string OrderId { get; set; }
        public required string Message { get; set; }
        public string Channel { get; set; } = "Email";
    }
}
