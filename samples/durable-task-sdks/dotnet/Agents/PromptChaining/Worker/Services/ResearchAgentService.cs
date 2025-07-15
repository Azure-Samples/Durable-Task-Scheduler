using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Configuration;

namespace AgentChainingSample.Services;

/// <summary>
/// Research agent service for news article research
/// </summary>
public class ResearchAgentService : BaseAgentService
{
    private const string AgentName = "ResearchAgent";
    private const string EndpointConfigKey = "AGENT_CONNECTION_STRING";
    private readonly string _systemPrompt = @"You are an expert research agent specializing in news topics.
Your task is to analyze topics, identify key facts, and organize information in a journalistic format.
Focus on accuracy, thoroughness, and objectivity when researching any topic.";
    
    private bool _initialized = false;

    public ResearchAgentService(ILogger<ResearchAgentService> logger, IConfiguration configuration) 
        : base(configuration[EndpointConfigKey] ?? 
              throw new InvalidOperationException($"Configuration key '{EndpointConfigKey}' not set"), 
              logger,
              configuration)
    {
    }
    
    /// <summary>
    /// Initializes the agent if needed
    /// </summary>
    private async Task InitializeAsync()
    {
        if (!_initialized)
        {
            await EnsureAgentExistsAsync(AgentName, _systemPrompt);
            _initialized = true;
        }
    }
    
    /// <summary>
    /// Gets the research agent to gather information about a topic using web search
    /// </summary>
    /// <param name="topic">The news topic to research</param>
    /// <returns>Research data in JSON format</returns>
    public async Task<string> ResearchTopicAsync(string topic)
    {
        await InitializeAsync();
        
        string prompt = $@"Research the following news topic: '{topic}'.

Imagine you are a professional researcher for a news organization. Your task is to gather and organize 
comprehensive information about this topic that would be useful for writing a news article.

Follow these steps:
1. Consider what background information would be helpful
2. Think about latest developments or news on this topic
3. Consider different perspectives or viewpoints
4. Identify key facts, statistics, or potential quotes
5. Consider expert opinions or analyses that might be available

Organize your findings into:
- Key facts and details about the topic
- Potential news sources that would cover this topic
- A concise summary of what's important about this topic
- Possible angles for a news article

Respond in JSON format with the following structure:
{{
  ""facts"": [""fact1"", ""fact2"", ""fact3"", ...],
  ""sources"": [
    {{
      ""url"": ""https://example-source.com/article"",
      ""title"": ""Example Article Title"",
      ""description"": ""Brief description of what this source might contain""
    }},
    ...
  ],
  ""summary"": ""A concise summary of key findings about this topic"",
  ""articleAngles"": [""angle1"", ""angle2"", ...]
}}";

        Logger.LogInformation($"Requesting research for topic: {topic}");
        return await GetResponseAsync(prompt);
    }
}
