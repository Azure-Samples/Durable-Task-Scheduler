namespace DurableFunctionsSaga.Models
{
    public class Inventory
    {
        public string ProductId { get; set; } = string.Empty;
        public int AvailableQuantity { get; set; }
        public int ReservedQuantity { get; set; }
    }
}
