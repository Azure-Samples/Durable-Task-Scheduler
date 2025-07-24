using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Configuration;
using AgentChainingSample.Shared.Models;
using Azure.Identity;
using Azure.Core;
using System.Collections.Generic;
using System.Net.Http;
using System.Threading.Tasks;

namespace AgentChainingSample.Services;

/// <summary>
/// Image generation agent service for news articles with direct DALL-E integration
/// </summary>
public class ImageGenerationAgentService : BaseAgentService
{
    private const string EndpointConfigKey = "AGENT_CONNECTION_STRING";
    private const string DallEEndpointKey = "DALLE_ENDPOINT";
    
    private readonly IHttpClientFactory _httpClientFactory;
    private readonly string? _dallEEndpoint;
    private readonly bool _dalleEnabled;
    private readonly DefaultAzureCredential? _credential;
    
    public ImageGenerationAgentService(
        ILogger<ImageGenerationAgentService> logger, 
        IConfiguration configuration,
        IHttpClientFactory httpClientFactory) 
        : base(configuration[EndpointConfigKey] ?? 
              throw new InvalidOperationException($"Configuration key '{EndpointConfigKey}' not set"), 
              logger,
              configuration)
    {
        // Set required properties for initialization
        this.AgentName = "ImageGenerationAgent";
        this._systemPrompt = @"You are an expert image specialist for news articles.
Create detailed descriptions of images that would complement news stories.
Focus on photorealistic, journalistically appropriate imagery and compositions.
Provide descriptive captions that enhance understanding of the article content.";
    
        _httpClientFactory = httpClientFactory ?? throw new ArgumentNullException(nameof(httpClientFactory));
        
        // Check if DALL-E endpoint is set in configuration
        string? dalleEndpoint = configuration[DallEEndpointKey];
        
        // Only enable DALL-E if endpoint is provided
        _dalleEnabled = !string.IsNullOrEmpty(dalleEndpoint);
        
        if (_dalleEnabled)
        {
            Logger.LogInformation($"DALL-E image generation is enabled with Microsoft Entra ID authentication");
            Logger.LogInformation($"Using DALL-E endpoint: {dalleEndpoint}");
            
            // Verify endpoint format - DALL-E endpoint is different from Azure AI Projects endpoint
            if (!dalleEndpoint.Contains("openai.azure.com") && !dalleEndpoint.Contains("api-version"))
            {
                Logger.LogWarning($"DALL-E endpoint format appears incorrect: {dalleEndpoint}");
                Logger.LogWarning("DALL-E endpoint should follow the format: https://your-resource.openai.azure.com/openai/deployments/your-deployment-name/images/generations?api-version=2023-12-01-preview");
                Logger.LogWarning("This is different from the Azure AI Projects endpoint used for agent creation");
                Logger.LogWarning("Using placeholders instead of real image generation due to incorrect endpoint format");
                
                // Disable DALL-E if endpoint format is incorrect
                _dalleEnabled = false;
                _dallEEndpoint = null;
                return;
            }
            
            // Use the DALL-E endpoint from configuration directly without modification
            // Note: The DALLE_ENDPOINT should be the complete URL including deployment name and API version
            // Example format: https://your-resource.openai.azure.com/openai/deployments/your-deployment-name/images/generations?api-version=2023-12-01-preview
            _dallEEndpoint = dalleEndpoint;
            
            // Create DefaultAzureCredential for authentication
            _credential = new DefaultAzureCredential();
        }
        else
        {
            Logger.LogWarning("DALL-E image generation is disabled. Set DALLE_ENDPOINT environment variable to enable it.");
            Logger.LogWarning("The DALLE_ENDPOINT should point to your Azure OpenAI service's DALL-E deployment, which is different from the AI Projects endpoint.");
            _dallEEndpoint = null;
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
        if (!_dalleEnabled || _dallEEndpoint == null || _credential == null)
        {
            throw new InvalidOperationException("DALL-E is not enabled. Check configuration for DALLE_ENDPOINT.");
        }
        
        // Get an HTTP client from the factory
        using var httpClient = _httpClientFactory.CreateClient("DallEClient");
        
        // Create the request body for DALL-E API
        var requestBody = new
        {
            prompt = prompt,
            n = 1,
            size = "1024x1024",
            response_format = "url",
            quality = "standard"
        };
        
        using var content = new StringContent(
            System.Text.Json.JsonSerializer.Serialize(requestBody), 
            System.Text.Encoding.UTF8, 
            "application/json");
            
        Logger.LogInformation($"Using DALL-E endpoint directly: {_dallEEndpoint}");
        Logger.LogInformation($"Calling DALL-E API with prompt: {prompt}");
        
        // Get an access token from Azure using DefaultAzureCredential
        try 
        {
            Logger.LogInformation("Acquiring token for DALL-E API using DefaultAzureCredential");
            var tokenRequestContext = new Azure.Core.TokenRequestContext(
                scopes: new[] { "https://cognitiveservices.azure.com/.default" });
            var accessToken = await _credential.GetTokenAsync(tokenRequestContext);
            
            // Add the bearer token to the request header
            httpClient.DefaultRequestHeaders.Authorization = 
                new System.Net.Http.Headers.AuthenticationHeaderValue("Bearer", accessToken.Token);
                
            Logger.LogInformation("Successfully acquired authentication token for DALL-E API");
        }
        catch (Exception authEx)
        {
            Logger.LogError(authEx, "Failed to acquire authentication token for DALL-E API. Make sure you're logged in with 'az login' and have access to the Azure OpenAI resource");
            throw new UnauthorizedAccessException("Failed to authenticate with DALL-E API. See inner exception for details.", authEx);
        }
        
        // Make the API call - use the full endpoint URL directly from the configuration
        // The DALLE_ENDPOINT should already contain the complete URL including deployment name and API version
        using var response = await httpClient.PostAsync(_dallEEndpoint, content);
        
        // Check if the request was successful
        if (!response.IsSuccessStatusCode)
        {
            string errorContent = await response.Content.ReadAsStringAsync();
            Logger.LogError($"DALL-E API call failed: {response.StatusCode}, {errorContent}");
            throw new HttpRequestException($"DALL-E API call failed: {response.StatusCode}, {errorContent}");
        }
        
        // Parse the response
        string jsonResponse = await response.Content.ReadAsStringAsync();
        var responseObject = System.Text.Json.JsonSerializer.Deserialize<DalleResponse>(jsonResponse);
        
        if (responseObject == null || responseObject.data == null || responseObject.data.Length == 0)
        {
            throw new InvalidOperationException("No image URL returned from DALL-E API");
        }
        
        if (string.IsNullOrEmpty(responseObject.data[0].url))
        {
            throw new InvalidOperationException("Image URL is null or empty in DALL-E response");
        }
        
        Logger.LogInformation("Successfully generated image with DALL-E");
        return responseObject.data[0].url;
    }
}
