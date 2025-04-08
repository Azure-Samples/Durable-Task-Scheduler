namespace OrchestrationService.Models;

// Activity input class
public class ActivityInput
{
    public int IterationNumber { get; set; }
    public int ActivityNumber { get; set; }
}

// Activity result class
public class ActivityResult
{
    public int IterationNumber { get; set; }
    public int ActivityNumber { get; set; }
    public long ProcessingTimeMs { get; set; }
    public string Output { get; set; } = string.Empty;
}