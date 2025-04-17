// filepath: /Users/nickgreenfield1/workspace/Durable-Task-Scheduler/samples/portable-sdks/dotnet/HumanInteraction/Worker/ApprovalActivityInputs.cs
namespace HumanInteraction;

/// <summary>
/// Data structure for the ProcessApprovalActivity input
/// </summary>
public class ProcessApprovalActivityInput
{
    public string RequestId { get; set; } = string.Empty;
    public bool IsApproved { get; set; }
    public string Approver { get; set; } = string.Empty;
    public string Comments { get; set; } = string.Empty;
}
