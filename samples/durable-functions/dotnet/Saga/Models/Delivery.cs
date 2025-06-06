namespace DurableFunctionsSaga.Models
{
    public class Delivery
    {
        public required string OrderId { get; set; }
        public required string Address { get; set; }
        public bool IsScheduled { get; set; }
        public string? TrackingNumber { get; set; }
    }
}
