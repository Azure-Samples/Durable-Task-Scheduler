using AgentChainingSample.Activities;
using AgentChainingSample.Shared.Models;
using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;

namespace AgentChainingSample.Orchestrations;

/// <summary>
/// Orchestrates the news article generation workflow with multiple specialized agents
/// </summary>
public class ContentCreationOrchestration
{
    private readonly ILogger<ContentCreationOrchestration> _logger;

    public ContentCreationOrchestration(ILogger<ContentCreationOrchestration> logger)
    {
        _logger = logger;
    }

    public async Task<ContentWorkflowResult> RunAsync(TaskOrchestrationContext context, ContentCreationRequest request)
    {
        _logger.LogInformation("Starting news article generation workflow for topic: {Topic}", request.Topic);

        // 1. Research the topic with Research Agent using web search
        ResearchData researchData = await context.CallActivityAsync<ResearchData>(
            "ResearchTopicActivity", 
            request.Topic);
        
        _logger.LogInformation("Research completed for topic: {Topic}. Found {SourceCount} sources and {FactCount} facts", 
            request.Topic, researchData.Sources.Count, researchData.Facts.Count);

        // 2. Create article content with Content Generation Agent using knowledge files
        string articleContent = await context.CallActivityAsync<string>(
            "CreateArticleActivity", 
            (request.Topic, researchData));
        
        _logger.LogInformation("Article content created. Length: {Length} characters", articleContent.Length);

        // 3. Generate images with Image Generation Agent using DALL-E
        List<GeneratedImage> generatedImages = await context.CallActivityAsync<List<GeneratedImage>>(
            "GenerateImagesActivity", 
            (request.Topic, articleContent));
        
        _logger.LogInformation("Generated {Count} images for the article", generatedImages.Count);

        // 4. Assemble the final article with content and images and save to file in the project's tmp directory
        var articleResult = await context.CallActivityAsync<ArticleResult>(
            "AssembleFinalArticleActivity", 
            (articleContent, generatedImages));
        
        _logger.LogInformation("Final article assembled. Length: {Length} characters", articleResult.HtmlContent.Length);
        _logger.LogInformation("Article saved to file: {FilePath}", articleResult.FilePath);

        // 5. Return the complete workflow result
        return new ContentWorkflowResult
        {
            Topic = request.Topic,
            ResearchData = researchData,
            ArticleContent = articleContent,
            GeneratedImages = generatedImages,
            FinalArticle = articleResult.HtmlContent,
            ArticleFilePath = articleResult.FilePath,
            ArticleBlobUrl = articleResult.BlobUrl,
            CompletedTimestamp = DateTime.UtcNow
        };
    }
}
