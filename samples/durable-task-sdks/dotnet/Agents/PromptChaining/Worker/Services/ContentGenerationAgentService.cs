using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Configuration;

namespace AgentChainingSample.Services;

/// <summary>
/// Content generation agent service for news articles
/// </summary>
public class ContentGenerationAgentService : BaseAgentService
{
    private const string AgentName = "ContentGenerationAgent";
    private const string EndpointConfigKey = "AGENT_CONNECTION_STRING";
    private readonly string _systemPrompt = @"You are an expert news article writer with knowledge of journalistic standards.
Write professional articles with compelling headlines, strong leads, and proper sourcing.
Follow AP style guidelines, inverted pyramid structure, and maintain journalistic integrity.";
    
    private bool _initialized = false;

    public ContentGenerationAgentService(ILogger<ContentGenerationAgentService> logger, IConfiguration configuration) 
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
    /// Gets the content generation agent to create a news article from research data
    /// </summary>
    /// <param name="topic">The news topic</param>
    /// <param name="researchJson">Research data in JSON format</param>
    /// <returns>Complete news article</returns>
    public async Task<string> CreateArticleAsync(string topic, string researchJson)
    {
        await InitializeAsync();
        
        string prompt = $@"Write a professional news article about '{topic}' using the following research data:

{researchJson}

Imagine you are a professional journalist writing for a respected news outlet. Apply your knowledge
of AP style guidelines and journalistic best practices when writing this article.

Follow these guidelines:
1. Create a compelling headline that captures reader attention
2. Start with a strong lead paragraph that captures the 5 Ws (who, what, when, where, why)
3. Include key facts from the research in order of importance
4. Cite sources appropriately within the text using journalistic attribution standards
5. Organize the article with logical structure following inverted pyramid style
6. Write in a balanced, objective tone adhering to journalistic standards
7. Include quotes if available in the research with proper attribution
8. End with a conclusion that summarizes or provides perspective

The article should be approximately 400-600 words in length.
Format the article with appropriate HTML tags (<h1>, <p>, etc.) following journalistic standards.";

        Logger.LogInformation($"Requesting article creation for topic: {topic}");
        return await GetResponseAsync(prompt);
    }
}
