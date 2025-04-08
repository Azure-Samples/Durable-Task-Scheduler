namespace WorkerService.Models;

// Input model for the HelloWorld test
public class HelloWorldRequest
{
    public int Iterations { get; set; } = 10;
    public int ParallelActivities { get; set; } = 5;
    public int ParallelOrchestrations { get; set; } = 1;
}

// The input for the HelloWorld orchestration
public class HelloWorldInput
{
    public int Iterations { get; set; }
    public int ParallelActivities { get; set; }
}

// The result of the HelloWorld test
public class HelloWorldResult
{
    public int TotalActivities { get; set; }
    public long ElapsedTimeMs { get; set; }
    public double AverageActivityTimeMs { get; set; }
    public List<ActivityResult> Results { get; set; } = new List<ActivityResult>();
}