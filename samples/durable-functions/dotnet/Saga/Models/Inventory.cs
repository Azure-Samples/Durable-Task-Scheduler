namespace DurableFunctionsSaga.Models
{
    public class Inventory
    {
        public required string ProductId { get; set; }
        public int ReservedQuantity { get; set; }
        public bool Reserved { get; set; }
        public bool Updated { get; set; }
    }
}
