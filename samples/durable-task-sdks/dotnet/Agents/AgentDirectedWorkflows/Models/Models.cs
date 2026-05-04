namespace AgentDirectedWorkflows;

public record ChatMsg(string Role, string Content);
public record ChatRequest(string SessionId, string Message, string? CorrelationId = null);

public class ChatAgentState
{
    public List<ChatMsg> Messages { get; set; } = [];
}

public record ChatTool(string Name, string Description);
