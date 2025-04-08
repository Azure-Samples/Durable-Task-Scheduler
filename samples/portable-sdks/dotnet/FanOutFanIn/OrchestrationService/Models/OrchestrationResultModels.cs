namespace OrchestrationService.Models;

public class FanOutFanInTestResult
{
    public int TotalActivities { get; set; }
    public long ElapsedTimeMs { get; set; }
    public double AverageActivityTimeMs { get; set; }
    public List<ActivityResult> Results { get; set; } = new List<ActivityResult>();
}