namespace ClientService.Models;

// Input model for the HelloWorld test
public class HelloWorldRequest
{
    public int Iterations { get; set; } = 10;
    public int ParallelActivities { get; set; } = 5;
    public int ParallelOrchestrations { get; set; } = 1;
}

// The input for the orchestration
public class HelloWorldInput
{
    public int Iterations { get; set; }
    public int ParallelActivities { get; set; }
}