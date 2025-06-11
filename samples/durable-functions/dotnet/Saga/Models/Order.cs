using System;


namespace DurableFunctionsSaga.Models
{
    public class Order
    {
        public string OrderId { get; set; } = Guid.NewGuid().ToString();
        public required string CustomerId { get; set; }
        public required string ProductId { get; set; }
        public int Quantity { get; set; }
        public decimal Amount { get; set; }
        public string Status { get; set; } = "Pending";
    }
}
