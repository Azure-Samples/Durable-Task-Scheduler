namespace OrchestrationService.Models;

// Input model for the fan-out/fan-in test
public class FanOutFanInRequest
{
    public int Iterations { get; set; } = 10;
    public int ParallelActivities { get; set; } = 5;
    public int ParallelOrchestrations { get; set; } = 1;
}

// The input for the orchestration
public class FanOutFanInOrchestrationInput
{
    public int Iterations { get; set; }
    public int ParallelActivities { get; set; }
}