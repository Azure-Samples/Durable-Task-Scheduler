// filepath: /Users/nickgreenfield1/workspace/Durable-Task-Scheduler/samples/durable-task-sdks/dotnet/HumanInteraction/Worker/ApprovalOrchestration.cs
using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;
using System.Text.Json;

namespace HumanInteraction;

/// <summary>
/// Data structure for the approval request input
/// </summary>
public class ApprovalRequestData
{
    public string RequestId { get; set; } = string.Empty;
    public string Requester { get; set; } = string.Empty;
    public string Item { get; set; } = string.Empty;
    public double TimeoutHours { get; set; } = 24.0;
}

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

/// <summary>
/// Data structure representing the submission result
/// </summary>
public class SubmissionResult
{
    public string RequestId { get; set; } = string.Empty;
    public string Status { get; set; } = string.Empty;
    public string SubmittedAt { get; set; } = string.Empty;
    public string ApprovalUrl { get; set; } = string.Empty;
}

/// <summary>
/// Data structure representing the approval processing result
/// </summary>
public class ApprovalResult
{
    public string RequestId { get; set; } = string.Empty;
    public string Status { get; set; } = string.Empty;
    public string ProcessedAt { get; set; } = string.Empty;
    public string? Approver { get; set; }
}

[DurableTask]
public class ApprovalOrchestration : TaskOrchestrator<ApprovalRequestData, ApprovalResult>
{
    public override async Task<ApprovalResult> RunAsync(TaskOrchestrationContext context, ApprovalRequestData input)
    {
        string requestId = input.RequestId;
        string requester = input.Requester;
        string item = input.Item;
        double timeoutHours = input.TimeoutHours;

        // Step 1: Submit the approval request
        var requestData = new ApprovalRequestData
        {
            RequestId = requestId,
            Requester = requester,
            Item = item
        };

        // Submit the approval request
        SubmissionResult submissionResult = await context.CallActivityAsync<SubmissionResult>(
            nameof(SubmitApprovalRequestActivity), 
            requestData);

        // Make the status available via custom status
        context.SetCustomStatus(submissionResult);

        // Create a durable timer for the timeout
        DateTime timeoutDeadline = context.CurrentUtcDateTime.AddHours(timeoutHours);
        
        using var timeoutCts = new CancellationTokenSource();
        
        // Set up the timeout task that we can cancel if approval comes before timeout
        Task timeoutTask = context.CreateTimer(timeoutDeadline, timeoutCts.Token);
        
        // Wait for an external event (approval/rejection)
        string approvalEventName = "approval_response";
        
        Task<ApprovalResponseData> approvalTask = context.WaitForExternalEvent<ApprovalResponseData>(approvalEventName);

        // Wait for either the timeout or the approval response, whichever comes first
        Task completedTask = await Task.WhenAny(approvalTask, timeoutTask);

        // Process based on which task completed
        ApprovalResult result;
        
        if (completedTask == approvalTask)
        {
            // Human responded in time - cancel the timeout timer
            timeoutCts.Cancel();

            // Get the event result
            ApprovalResponseData approvalData = approvalTask.Result;
            
            // Process the approval
            result = await context.CallActivityAsync<ApprovalResult>(
                nameof(ProcessApprovalActivity),
                new ProcessApprovalActivityInput
                {
                    RequestId = requestId,
                    IsApproved = approvalData.IsApproved,
                    Approver = approvalData.Approver,
                    Comments = approvalData.Comments
                });
        }
        else
        {
            // Timeout occurred
            result = new ApprovalResult
            {
                RequestId = requestId,
                Status = "Timeout",
                ProcessedAt = context.CurrentUtcDateTime.ToString("o")
            };
        }

        return result;
    }
}

[DurableTask]
public class SubmitApprovalRequestActivity : TaskActivity<ApprovalRequestData, SubmissionResult>
{
    private readonly ILogger<SubmitApprovalRequestActivity> _logger;

    public SubmitApprovalRequestActivity(ILogger<SubmitApprovalRequestActivity> logger)
    {
        _logger = logger;
    }

    public override Task<SubmissionResult> RunAsync(TaskActivityContext context, ApprovalRequestData requestData)
    {
        string requestId = requestData.RequestId;
        string requester = requestData.Requester;
        string item = requestData.Item;

        _logger.LogInformation("Submitting approval request {RequestId} from {Requester} for {Item}", requestId, requester, item);

        // In a real system, this would send an email, notification, or update a database
        SubmissionResult result = new SubmissionResult
        {
            RequestId = requestId,
            Status = "Pending",
            SubmittedAt = DateTime.UtcNow.ToString("o"),
            ApprovalUrl = $"http://localhost:8000/api/approvals/{requestId}"
        };

        return Task.FromResult(result);
    }
}

[DurableTask]
public class ProcessApprovalActivity : TaskActivity<ProcessApprovalActivityInput, ApprovalResult>
{
    private readonly ILogger<ProcessApprovalActivity> _logger;

    public ProcessApprovalActivity(ILogger<ProcessApprovalActivity> logger)
    {
        _logger = logger;
    }

    public override Task<ApprovalResult> RunAsync(TaskActivityContext context, ProcessApprovalActivityInput input)
    {
        // Use the strongly-typed input
        string requestId = input.RequestId;
        bool isApproved = input.IsApproved;
        string approver = input.Approver;
        string comments = input.Comments ?? "";

        string approvalStatus = isApproved ? "Approved" : "Rejected";
        _logger.LogInformation("Processing {ApprovalStatus} request {RequestId} by {Approver}", 
            approvalStatus, requestId, approver);

        // In a real system, this would update a database, trigger workflows, etc.
        ApprovalResult result = new ApprovalResult
        {
            RequestId = requestId,
            Status = approvalStatus,
            ProcessedAt = DateTime.UtcNow.ToString("o"),
            Approver = approver
        };

        return Task.FromResult(result);
    }
}
