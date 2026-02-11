using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;

namespace Monitoring;

[DurableTask]
public class CheckJobStatusActivity : TaskActivity<string, JobStatus>
{
    readonly ILogger<CheckJobStatusActivity> logger;

    public CheckJobStatusActivity(ILogger<CheckJobStatusActivity> logger)
    {
        this.logger = logger;
    }

    public override Task<JobStatus> RunAsync(TaskActivityContext context, string jobId)
    {
        // Simulate a job that completes after a few polls.
        // In a real scenario, this would call an external API.
        int hash = Math.Abs(jobId.GetHashCode());
        int completionPoll = (hash % 4) + 3; // Completes between poll 3-6
        int currentAttempt = int.Parse(
            Environment.GetEnvironmentVariable($"POLL_COUNT_{jobId}") ?? "0") + 1;
        Environment.SetEnvironmentVariable($"POLL_COUNT_{jobId}", currentAttempt.ToString());

        if (currentAttempt >= completionPoll)
        {
            this.logger.LogInformation("Job '{JobId}' has completed!", jobId);
            return Task.FromResult(new JobStatus(true, "Completed", $"Job '{jobId}' finished successfully after {currentAttempt} checks."));
        }

        string status = currentAttempt == 1 ? "Starting" : "In Progress";
        this.logger.LogInformation("Job '{JobId}' status: {Status} ({Current}/{Target})",
            jobId, status, currentAttempt, completionPoll);

        return Task.FromResult(new JobStatus(false, status, string.Empty));
    }
}
