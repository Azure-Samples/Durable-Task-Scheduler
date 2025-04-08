namespace ClientService.Models;

public class ActivityInput
{
    public int IterationNumber { get; set; }
    public int ActivityNumber { get; set; }
}

public class ActivityResult
{
    public int IterationNumber { get; set; }
    public int ActivityNumber { get; set; }
    public long ProcessingTimeMs { get; set; }
    public string Output { get; set; } = string.Empty;
}

public class FanOutFanInTestResult
{
    public int TotalActivities { get; set; }
    public long ElapsedTimeMs { get; set; }
    public double AverageActivityTimeMs { get; set; }
    public List<ActivityResult> Results { get; set; } = new List<ActivityResult>();
}