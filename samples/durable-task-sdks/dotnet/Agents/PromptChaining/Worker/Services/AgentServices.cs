using Microsoft.Extensions.Logging;
using AgentChainingSample.Shared.Models;
using Azure.Identity;
using Azure.Core;

namespace AgentChainingSample.Services;

/// <summary>
/// Research agent service for news article research
/// </summary>
public class ResearchAgentService : BaseAgentService
{
    private const string AgentName = "ResearchAgent";
    private const string EndpointEnvVar = "AGENT_CONNECTION_STRING";
    private readonly string _systemPrompt = @"You are an expert research agent specializing in news topics.
Your task is to analyze topics, identify key facts, and organize information in a journalistic format.
Focus on accuracy, thoroughness, and objectivity when researching any topic.";
    
    private bool _initialized = false;

    public ResearchAgentService(ILogger<ResearchAgentService> logger) 
        : base(Environment.GetEnvironmentVariable(EndpointEnvVar) ?? 
              throw new InvalidOperationException($"Environment variable '{EndpointEnvVar}' not set"), 
              logger)
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

/// <summary>
/// Content generation agent service for news articles
/// </summary>
public class ContentGenerationAgentService : BaseAgentService
{
    private const string AgentName = "ContentGenerationAgent";
    private const string EndpointEnvVar = "AGENT_CONNECTION_STRING";
    private readonly string _systemPrompt = @"You are an expert news article writer with knowledge of journalistic standards.
Write professional articles with compelling headlines, strong leads, and proper sourcing.
Follow AP style guidelines, inverted pyramid structure, and maintain journalistic integrity.";
    
    private bool _initialized = false;

    public ContentGenerationAgentService(ILogger<ContentGenerationAgentService> logger) 
        : base(Environment.GetEnvironmentVariable(EndpointEnvVar) ?? 
              throw new InvalidOperationException($"Environment variable '{EndpointEnvVar}' not set"), 
              logger)
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

/// <summary>
/// Image generation agent service for news articles with direct DALL-E integration
/// </summary>
public class ImageGenerationAgentService : BaseAgentService
{
    private const string AgentName = "ImageGenerationAgent";
    private const string EndpointEnvVar = "AGENT_CONNECTION_STRING";
    private readonly string _systemPrompt = @"You are an expert image specialist for news articles.
Create detailed descriptions of images that would complement news stories.
Focus on photorealistic, journalistically appropriate imagery and compositions.
Provide descriptive captions that enhance understanding of the article content.";
    
    private bool _initialized = false;
    private readonly HttpClient? _httpClient;
    private readonly string? _dallEEndpoint;
    private readonly bool _dalleEnabled;
    private readonly DefaultAzureCredential? _credential;
    
    public ImageGenerationAgentService(ILogger<ImageGenerationAgentService> logger) 
        : base(Environment.GetEnvironmentVariable(EndpointEnvVar) ?? 
              throw new InvalidOperationException($"Environment variable '{EndpointEnvVar}' not set"), 
              logger)
    {
        // Check if DALL-E endpoint is set
        string? dalleEndpoint = Environment.GetEnvironmentVariable("DALLE_ENDPOINT");
        
        // Only enable DALL-E if endpoint is provided
        _dalleEnabled = !string.IsNullOrEmpty(dalleEndpoint);
        
        if (_dalleEnabled)
        {
            Logger.LogInformation("DALL-E image generation is enabled with Microsoft Entra ID authentication");
            
            // Use the DALL-E endpoint environment variable directly without modification
            _dallEEndpoint = dalleEndpoint;
            
            // Create HTTP client
            _httpClient = new HttpClient();
            
            // Create DefaultAzureCredential for authentication
            _credential = new DefaultAzureCredential();
        }
        else
        {
            Logger.LogWarning("DALL-E image generation is disabled. Set DALLE_ENDPOINT and DALLE_API_KEY environment variables to enable it.");
            _dallEEndpoint = null;
            _httpClient = null;
        }
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
    /// Gets image descriptions from the agent and then generates actual images using DALL-E
    /// </summary>
    /// <param name="topic">The news topic</param>
    /// <param name="articleText">The article text</param>
    /// <returns>Generated image details in JSON format</returns>
    public async Task<string> GenerateImagesAsync(string topic, string articleText)
    {
        await InitializeAsync();
        
        // Step 1: Get image descriptions from the agent
        string prompt = $@"Create descriptions for two compelling images to accompany a news article about '{topic}'.
Here's the article text to help you understand the content:

{articleText}

For each image, create an extremely detailed prompt that would work well with DALL-E image generation.
Make the prompts detailed, specific, and visually descriptive to generate photorealistic journalistic images.

For each image:

1. First, identify key visual concepts from the article that would make compelling images:
   - Main subject or focus of the article
   - Key scene or setting described
   - Visual representation of important concepts

2. Then, create a detailed prompt for DALL-E generation:
   - Be extremely specific about subject matter
   - Include details about composition, perspective, lighting
   - Specify photorealistic style appropriate for news
   - Include details about setting, mood, and context

3. Write a descriptive caption that would accompany the image in the article

Respond in JSON format with the following structure:
[
  {{
    ""description"": ""Brief description of what this image represents"",
    ""prompt"": ""Detailed DALL-E prompt to generate a photorealistic news image"",
    ""caption"": ""Caption for this image as it would appear in the article""
  }},
  {{
    ""description"": ""Brief description of what this second image represents"",
    ""prompt"": ""Detailed DALL-E prompt to generate a photorealistic news image"",
    ""caption"": ""Caption for this second image as it would appear in the article""
  }}
]";

        Logger.LogInformation("Requesting image descriptions from agent");
        string descriptionsJson = await GetResponseAsync(prompt);
        string cleanDescriptionsJson = CleanJsonResponse(descriptionsJson);
        
        try
        {
            // Step 2: Parse the descriptions and generate actual images with DALL-E
            var imageDescriptions = System.Text.Json.JsonSerializer.Deserialize<List<ImageDescription>>(
                cleanDescriptionsJson, 
                new System.Text.Json.JsonSerializerOptions { PropertyNameCaseInsensitive = true });
                
            if (imageDescriptions == null || !imageDescriptions.Any())
            {
                Logger.LogWarning("No valid image descriptions received from agent");
                return "[]";
            }
            
            Logger.LogInformation($"Received {imageDescriptions.Count} image descriptions");
            
            // Generate images for each description
            var generatedImages = new List<GeneratedImage>();
            
            if (_dalleEnabled)
            {
                Logger.LogInformation("Generating images with DALL-E");
                
                foreach (var description in imageDescriptions)
                {
                    try
                    {
                        // Call DALL-E API to generate the image
                        var imageUrl = await GenerateImageWithDallE(description.Prompt);
                        
                        generatedImages.Add(new GeneratedImage
                        {
                            Description = description.Description,
                            Prompt = description.Prompt,
                            ImageUrl = imageUrl,
                            Caption = description.Caption
                        });
                        
                        Logger.LogInformation($"Successfully generated image: {description.Description}");
                    }
                    catch (Exception ex)
                    {
                        Logger.LogError(ex, $"Error generating image for prompt: {description.Prompt}");
                        AddPlaceholderImage(generatedImages, description);
                    }
                }
            }
            else
            {
                // DALL-E is not configured, use placeholders instead
                Logger.LogWarning("DALL-E is not configured. Using placeholder images instead.");
                
                foreach (var description in imageDescriptions)
                {
                    AddPlaceholderImage(generatedImages, description);
                }
            }
            
            // Return the generated images as JSON
            return System.Text.Json.JsonSerializer.Serialize(generatedImages);
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Error processing image descriptions");
            return "[]";
        }
    }
    
    /// <summary>
    /// Adds a placeholder image to the list of generated images
    /// </summary>
    private void AddPlaceholderImage(List<GeneratedImage> images, ImageDescription description)
    {
        images.Add(new GeneratedImage
        {
            Description = description.Description,
            Prompt = description.Prompt,
            ImageUrl = "https://via.placeholder.com/800x600?text=Image+Generation+Sample",
            Caption = description.Caption
        });
        
        Logger.LogInformation($"Added placeholder image for: {description.Description}");
    }
    
    /// <summary>
    /// Calls the DALL-E API to generate an image from a prompt using Microsoft Entra ID authentication
    /// </summary>
    /// <param name="prompt">The image generation prompt</param>
    /// <returns>URL of the generated image</returns>
    private async Task<string> GenerateImageWithDallE(string prompt)
    {
        if (!_dalleEnabled || _httpClient == null || _dallEEndpoint == null || _credential == null)
        {
            throw new InvalidOperationException("DALL-E is not enabled. Check environment variable DALLE_ENDPOINT.");
        }
        
        // Create the request body for DALL-E API
        var requestBody = new
        {
            prompt = prompt,
            n = 1,
            size = "1024x1024",
            response_format = "url",
            quality = "standard"
        };
        
        var content = new StringContent(
            System.Text.Json.JsonSerializer.Serialize(requestBody), 
            System.Text.Encoding.UTF8, 
            "application/json");
            
        Logger.LogInformation($"Calling DALL-E API with prompt: {prompt}");
        
        // Get an access token from Azure using DefaultAzureCredential
        var tokenRequestContext = new Azure.Core.TokenRequestContext(
            scopes: new[] { "https://cognitiveservices.azure.com/.default" });
        var accessToken = await _credential.GetTokenAsync(tokenRequestContext);
        
        // Add the bearer token to the request header
        _httpClient.DefaultRequestHeaders.Authorization = 
            new System.Net.Http.Headers.AuthenticationHeaderValue("Bearer", accessToken.Token);
        
        // Make the API call
        var response = await _httpClient.PostAsync(_dallEEndpoint, content);
        
        // Check if the request was successful
        if (!response.IsSuccessStatusCode)
        {
            var errorContent = await response.Content.ReadAsStringAsync();
            Logger.LogError($"DALL-E API error: {response.StatusCode}, {errorContent}");
            throw new Exception($"DALL-E API returned status code {response.StatusCode}");
        }
        
        // Parse the response
        var responseContent = await response.Content.ReadAsStringAsync();
        var responseJson = System.Text.Json.JsonDocument.Parse(responseContent);
        
        // Extract the image URL
        var imageUrl = responseJson.RootElement
            .GetProperty("data")[0]
            .GetProperty("url")
            .GetString();
            
        if (string.IsNullOrEmpty(imageUrl))
        {
            throw new Exception("No image URL found in DALL-E response");
        }
        
        Logger.LogInformation("Successfully generated image with DALL-E");
        return imageUrl;
    }
}

/// <summary>
/// Model for image description
/// </summary>
public class ImageDescription
{
    public string Description { get; set; } = string.Empty;
    public string Prompt { get; set; } = string.Empty;
    public string Caption { get; set; } = string.Empty;
}
