using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;

namespace Monitoring;

[DurableTask]
public class MonitorOrchestration : TaskOrchestrator<MonitorInput, string>
{
    public override async Task<string> RunAsync(TaskOrchestrationContext context, MonitorInput input)
    {
        ILogger logger = context.CreateReplaySafeLogger<MonitorOrchestration>();

        int pollingIntervalSeconds = input.PollingIntervalSeconds > 0 ? input.PollingIntervalSeconds : 5;
        int maxPolls = input.MaxPolls > 0 ? input.MaxPolls : 10;
        int currentPoll = input.CurrentPoll;

        if (currentPoll >= maxPolls)
        {
            logger.LogWarning("Max polls ({MaxPolls}) reached for job '{JobId}'. Timing out.", maxPolls, input.JobId);
            return $"Timeout: Job '{input.JobId}' did not complete within {maxPolls} polls.";
        }

        currentPoll++;
        logger.LogInformation("Poll {Current}/{Max} — checking status for job '{JobId}'...",
            currentPoll, maxPolls, input.JobId);

        // Call the activity to check the job status
        JobStatus status = await context.CallActivityAsync<JobStatus>(
            nameof(CheckJobStatusActivity), input.JobId);

        if (status.IsComplete)
        {
            logger.LogInformation("Job '{JobId}' completed with result: {Result}", input.JobId, status.Result);
            return status.Result;
        }

        logger.LogInformation("Job '{JobId}' not yet complete (status: {Status}). Waiting {Interval}s before next poll...",
            input.JobId, status.CurrentStatus, pollingIntervalSeconds);

        // Wait before the next poll
        await context.CreateTimer(TimeSpan.FromSeconds(pollingIntervalSeconds), CancellationToken.None);

        // Restart the orchestration with updated state
        context.ContinueAsNew(input with { CurrentPoll = currentPoll });
        return string.Empty; // Won't be reached
    }
}

public record MonitorInput(
    string JobId,
    int PollingIntervalSeconds = 5,
    int MaxPolls = 10,
    int CurrentPoll = 0);

public record JobStatus(bool IsComplete, string CurrentStatus, string Result);
