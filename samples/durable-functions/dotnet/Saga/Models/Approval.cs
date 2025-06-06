namespace DurableFunctionsSaga.Models
{
    public class Approval
    {
        public required string OrderId { get; set; }
        public bool IsApproved { get; set; }
        public string? ApprovalId { get; set; }
    }
}
