namespace DurableFunctionsSaga.Models
{
    public class Delivery
    {
        public string OrderId { get; set; } = string.Empty;
        public string Address { get; set; } = string.Empty;
        public string Status { get; set; } = string.Empty;
    }
}
