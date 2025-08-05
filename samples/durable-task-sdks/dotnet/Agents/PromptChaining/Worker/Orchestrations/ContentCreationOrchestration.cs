using AgentChainingSample.Activities;
using AgentChainingSample.Worker.Models;
using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;

namespace AgentChainingSample.Orchestrations;

/// <summary>
/// Orchestrates the news article generation workflow with multiple specialized agents
/// </summary>
[DurableTask]
public class ContentCreationOrchestration : TaskOrchestrator<ContentCreationRequest, ContentWorkflowResult>
{
    public override async Task<ContentWorkflowResult> RunAsync(TaskOrchestrationContext context, ContentCreationRequest request)
    {
        // Create a replay-safe logger
        ILogger logger = context.CreateReplaySafeLogger<ContentCreationOrchestration>();
        logger.LogInformation("Starting news article generation workflow for topic: {Topic}", request.Topic);

        // 1. Research the topic with Research Agent using web search
        ResearchData researchData = await context.CallActivityAsync<ResearchData>(
            nameof(ResearchTopicActivity), 
            request.Topic);
        
        logger.LogInformation("Research completed for topic: {Topic}. Found {SourceCount} sources and {FactCount} facts", 
            request.Topic, researchData.Sources.Count, researchData.Facts.Count);

        // 2. Create article content with Content Generation Agent using knowledge files
        string articleContent = await context.CallActivityAsync<string>(
            nameof(CreateArticleActivity), 
            (request.Topic, researchData));
        
        logger.LogInformation("Article content created. Length: {Length} characters", articleContent.Length);

        // 3. Generate images with Image Generation Agent using DALL-E
        List<GeneratedImage> generatedImages = await context.CallActivityAsync<List<GeneratedImage>>(
            nameof(GenerateImagesActivity), 
            (request.Topic, articleContent));
        
        logger.LogInformation("Generated {Count} images for the article", generatedImages.Count);

        // 4. Assemble the final article with content and images and save to file in the project's tmp directory
        var articleResult = await context.CallActivityAsync<ArticleResult>(
            nameof(AssembleFinalArticleActivity), 
<<<<<<< HEAD
<<<<<<< HEAD
            (articleContent, generatedImages, context.InstanceId));
        
        logger.LogInformation("Final article assembled. Length: {Length} characters", articleResult.HtmlContent.Length);
        logger.LogInformation("Article saved to file: {FilePath}", articleResult.FilePath);
        logger.LogInformation("Article endpoint: {Endpoint}", articleResult.ArticleEndpoint);
=======
            (articleContent, generatedImages));
        
        logger.LogInformation("Final article assembled. Length: {Length} characters", articleResult.HtmlContent.Length);
        logger.LogInformation("Article saved to file: {FilePath}", articleResult.FilePath);
>>>>>>> 21471c1 (Address PR feedback)
=======
            (articleContent, generatedImages, context.InstanceId));
        
        logger.LogInformation("Final article assembled. Length: {Length} characters", articleResult.HtmlContent.Length);
        logger.LogInformation("Article saved to file: {FilePath}", articleResult.FilePath);
        logger.LogInformation("Article endpoint: {Endpoint}", articleResult.ArticleEndpoint);
>>>>>>> c34d9d0 (Added Bicep)

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
            ArticleEndpoint = articleResult.ArticleEndpoint,
            CompletedTimestamp = DateTime.UtcNow
        };
    }
}