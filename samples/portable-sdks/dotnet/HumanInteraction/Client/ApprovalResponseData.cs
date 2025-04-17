using System;

namespace HumanInteraction.Client
{
    /// <summary>
    /// Data structure for the approval response
    /// </summary>
    public class ApprovalResponseData
    {
        public bool IsApproved { get; set; }
        public string Approver { get; set; } = string.Empty;
        public string Comments { get; set; } = string.Empty;
        public string ResponseTime { get; set; } = string.Empty;
    }
}
