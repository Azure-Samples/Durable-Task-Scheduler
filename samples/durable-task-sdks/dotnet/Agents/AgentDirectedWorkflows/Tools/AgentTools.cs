using Microsoft.Extensions.AI;

namespace AgentDirectedWorkflows;

/// <summary>
/// Tools that the agent can use. Add new tools here to give the agent new capabilities.
/// Each tool needs: a definition (name + description for the LLM) and an implementation.
/// </summary>
public static class AgentTools
{
    // Tool definitions — these are sent to the LLM so it knows what it can call.
    public static readonly ChatTool[] Definitions = [
        new("get_weather", "Get current weather for a location"),
    ];

    /// <summary>Builds the AITool list that the LLM understands.</summary>
    public static List<AITool> AsAITools() =>
        Definitions.Select(t =>
            AIFunctionFactory.Create((string location) => "", t.Name, t.Description) as AITool).ToList();

    /// <summary>Executes a tool by name and returns the result string.</summary>
    public static string Execute(string name, IDictionary<string, object?>? args)
    {
        var location = args?.Values.FirstOrDefault()?.ToString() ?? "unknown";
        return name switch
        {
            "get_weather" => $"72°F and sunny in {location}",
            _ => $"Unknown tool: {name}",
        };
    }
}
