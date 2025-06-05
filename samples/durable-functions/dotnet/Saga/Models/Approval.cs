namespace DurableFunctionsSaga.Models
{
    public class Approval
    {
        public string OrderId { get; set; } = string.Empty;
        public bool IsApproved { get; set; }
    }
}
