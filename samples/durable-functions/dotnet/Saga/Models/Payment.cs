namespace DurableFunctionsSaga.Models
{
    public class Payment
    {
        public required string OrderId { get; set; }
        public decimal Amount { get; set; }
        public bool IsProcessed { get; set; }
        public string? TransactionId { get; set; }
    }
}
