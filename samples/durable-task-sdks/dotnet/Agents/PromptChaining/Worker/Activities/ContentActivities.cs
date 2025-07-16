using AgentChainingSample.Shared.Models;
using AgentChainingSample.Services;
using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;
using System;
using System.Collections.Generic;
using System.IO;
using System.Text;
using System.Text.RegularExpressions;
using System.Threading.Tasks;

namespace AgentChainingSample.Activities;

/// <summary>
/// Result returned by the AssembleFinalArticleActivity
/// </summary>
public class ArticleResult
{
    /// <summary>
    /// The complete HTML content of the article
    /// </summary>
    public string HtmlContent { get; set; } = string.Empty;
    
    /// <summary>
    /// The local file path where the HTML content is saved
    /// </summary>
    public string FilePath { get; set; } = string.Empty;
    
    /// <summary>
    /// The URL to the article in blob storage (kept for compatibility, always empty)
    /// </summary>
    public string BlobUrl { get; set; } = string.Empty;
}

/// <summary>
/// Activity to research a topic using the research agent with web search
/// </summary>
[DurableTask]
public class ResearchTopicActivity(ResearchAgentService researchAgentService, ILogger<ResearchTopicActivity> logger) 
    : TaskActivity<string, ResearchData>
{

    public override async Task<ResearchData> RunAsync(TaskActivityContext context, string topic)
    {
        logger.LogInformation("Researching topic using web search: {Topic}", topic);

        try
        {
            string researchJson = await researchAgentService.ResearchTopicAsync(topic);
            logger.LogInformation("Successfully collected research data for topic: {Topic}", topic);
            
            // Clean the JSON response
            string cleanJson = researchAgentService.CleanJsonResponse(researchJson);
            
            // Parse the JSON into research data
            ResearchData researchData = ResearchData.FromJson(cleanJson);
            return researchData;
        }
        catch (Exception ex)
        {
            logger.LogError(ex, "Error researching topic: {Topic}", topic);
            throw;
        }
    }
}

/// <summary>
/// Activity to create article content using the content generation agent with knowledge files
/// </summary>
[DurableTask]
public class CreateArticleActivity(ContentGenerationAgentService contentGenerationService, ILogger<CreateArticleActivity> logger)
    : TaskActivity<(string Topic, ResearchData ResearchData), string>
{

    public override async Task<string> RunAsync(TaskActivityContext context, (string Topic, ResearchData ResearchData) input)
    {
        logger.LogInformation("Creating article for topic: {Topic}", input.Topic);

        try
        {
            // Serialize research data back to JSON to pass to agent
            string researchJson = System.Text.Json.JsonSerializer.Serialize(input.ResearchData, 
                new System.Text.Json.JsonSerializerOptions { WriteIndented = true });
            
            string articleContent = await contentGenerationService.CreateArticleAsync(input.Topic, researchJson);
            logger.LogInformation("Successfully created article content of {Length} characters", articleContent.Length);
            return articleContent;
        }
        catch (Exception ex)
        {
            logger.LogError(ex, "Error creating article for topic: {Topic}", input.Topic);
            throw;
        }
    }
}

/// <summary>
/// Activity to generate images using the image generation agent with DALL-E
/// </summary>
[DurableTask]
public class GenerateImagesActivity(ImageGenerationAgentService imageGenerationService, ILogger<GenerateImagesActivity> logger)
    : TaskActivity<(string Topic, string ArticleContent), List<GeneratedImage>>
{

    public override async Task<List<GeneratedImage>> RunAsync(TaskActivityContext context, (string Topic, string ArticleContent) input)
    {
        logger.LogInformation("Generating images for article on topic: {Topic}", input.Topic);

        try
        {
            string imagesJson = await imageGenerationService.GenerateImagesAsync(input.Topic, input.ArticleContent);
            logger.LogInformation("Successfully generated image data");
            
            // Clean the JSON response
            string cleanJson = imageGenerationService.CleanJsonResponse(imagesJson);
            
            // Parse the JSON into generated images
            List<GeneratedImage> images = GeneratedImage.FromJson(cleanJson);
            return images;
        }
        catch (Exception ex)
        {
            logger.LogError(ex, "Error generating images for topic: {Topic}", input.Topic);
            throw;
        }
    }
}

/// <summary>
/// Activity to assemble the final article with images in HTML format and save to file
/// </summary>
[DurableTask]
public class AssembleFinalArticleActivity(ILogger<AssembleFinalArticleActivity> logger, string? outputDirectory = null)
    : TaskActivity<(string ArticleContent, List<GeneratedImage> Images), ArticleResult>
{
    // Use system temp directory by default, or the provided directory if specified
    private readonly string _outputDirectory = outputDirectory ?? Path.GetTempPath();

    public override async Task<ArticleResult> RunAsync(TaskActivityContext context, (string ArticleContent, List<GeneratedImage> Images) input)
    {
        logger.LogInformation("Assembling final article with images in HTML format");
        
        try
        {
            // Use the original HTML content
            string htmlContent = input.ArticleContent;
            
            // Get existing title if any
            string title = ExtractTextBetweenTags(htmlContent, "h1");
            if (string.IsNullOrEmpty(title))
            {
                title = "News Article";
            }
            
            // Create a proper HTML document structure with styling
            var htmlBuilder = new StringBuilder();
            htmlBuilder.AppendLine("<!DOCTYPE html>");
            htmlBuilder.AppendLine("<html lang=\"en\">");
            htmlBuilder.AppendLine("<head>");
            htmlBuilder.AppendLine("    <meta charset=\"UTF-8\">");
            htmlBuilder.AppendLine("    <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">");
            htmlBuilder.AppendLine($"    <title>{title}</title>");
            htmlBuilder.AppendLine("    <style>");
            htmlBuilder.AppendLine("        body { font-family: Arial, sans-serif; line-height: 1.6; margin: 0; padding: 20px; max-width: 800px; margin: 0 auto; }");
            htmlBuilder.AppendLine("        h1 { color: #333; margin-bottom: 25px; }");
            htmlBuilder.AppendLine("        p { margin-bottom: 20px; }");
            htmlBuilder.AppendLine("        img { max-width: 100%; height: auto; margin: 20px 0; }");
            htmlBuilder.AppendLine("        .image-container { margin: 30px 0; }");
            htmlBuilder.AppendLine("        .caption { font-style: italic; color: #666; margin-top: 10px; }");
            htmlBuilder.AppendLine("        .metadata { color: #999; font-size: 0.8em; border-top: 1px solid #eee; margin-top: 40px; padding-top: 20px; }");
            htmlBuilder.AppendLine("    </style>");
            htmlBuilder.AppendLine("</head>");
            htmlBuilder.AppendLine("<body>");
            
            // Extract sections from the HTML content
            var paragraphs = ExtractParagraphs(htmlContent);
            
            // Insert images at strategic points
            if (paragraphs.Count > 0)
            {
                // Add the first paragraph
                htmlBuilder.AppendLine(paragraphs[0]);
                
                // Add first image after the first paragraph
                if (input.Images.Count > 0)
                {
                    var image = input.Images[0];
                    htmlBuilder.AppendLine("<div class=\"image-container\">");
                    htmlBuilder.AppendLine($"    <img src=\"{image.ImageUrl}\" alt=\"{image.Description}\">");
                    htmlBuilder.AppendLine($"    <div class=\"caption\">{image.Caption}</div>");
                    htmlBuilder.AppendLine("</div>");
                }
                
                // Add middle paragraphs
                for (int i = 1; i < paragraphs.Count - 1; i++)
                {
                    htmlBuilder.AppendLine(paragraphs[i]);
                }
                
                // Add second image before the last paragraph
                if (input.Images.Count > 1)
                {
                    var image = input.Images[1];
                    htmlBuilder.AppendLine("<div class=\"image-container\">");
                    htmlBuilder.AppendLine($"    <img src=\"{image.ImageUrl}\" alt=\"{image.Description}\">");
                    htmlBuilder.AppendLine($"    <div class=\"caption\">{image.Caption}</div>");
                    htmlBuilder.AppendLine("</div>");
                }
                
                // Add the last paragraph
                if (paragraphs.Count > 1)
                {
                    htmlBuilder.AppendLine(paragraphs[paragraphs.Count - 1]);
                }
                
                // Add any remaining images
                for (int i = 2; i < input.Images.Count; i++)
                {
                    var image = input.Images[i];
                    htmlBuilder.AppendLine("<div class=\"image-container\">");
                    htmlBuilder.AppendLine($"    <img src=\"{image.ImageUrl}\" alt=\"{image.Description}\">");
                    htmlBuilder.AppendLine($"    <div class=\"caption\">{image.Caption}</div>");
                    htmlBuilder.AppendLine("</div>");
                }
            }
            else
            {
                // If we couldn't extract paragraphs, use the original content
                htmlBuilder.AppendLine(htmlContent);
                
                // Add images at the end
                foreach (var image in input.Images)
                {
                    htmlBuilder.AppendLine("<div class=\"image-container\">");
                    htmlBuilder.AppendLine($"    <img src=\"{image.ImageUrl}\" alt=\"{image.Description}\">");
                    htmlBuilder.AppendLine($"    <div class=\"caption\">{image.Caption}</div>");
                    htmlBuilder.AppendLine("</div>");
                }
            }
            
            // Add metadata at the bottom
            htmlBuilder.AppendLine("<div class=\"metadata\">");
            htmlBuilder.AppendLine($"    <p>Generated on: {DateTime.UtcNow:yyyy-MM-dd HH:mm:ss} UTC</p>");
            htmlBuilder.AppendLine("    <p>Images generated with DALL-E</p>");
            htmlBuilder.AppendLine("    <p>Research conducted with web search</p>");
            htmlBuilder.AppendLine("</div>");
            
            htmlBuilder.AppendLine("</body>");
            htmlBuilder.AppendLine("</html>");
            
            string finalHtml = htmlBuilder.ToString();
            
            // Generate a filename based on title and timestamp
            string sanitizedTitle = SanitizeForFileName(title);
            string timestamp = DateTime.UtcNow.ToString("yyyyMMddHHmmss");
            string filename = $"{sanitizedTitle}-{timestamp}.html";
            
            // Create subdirectory for this app's output files
            string outputDirectory = Path.Combine(_outputDirectory, "article-generator");
            
            // Ensure output directory exists
            Directory.CreateDirectory(outputDirectory);
            
            // Save the HTML content to the output directory
            string localFilePath = Path.Combine(outputDirectory, filename);
            await File.WriteAllTextAsync(localFilePath, finalHtml);
            
            logger.LogInformation("HTML article saved to file: {FilePath}", localFilePath);
            
            logger.LogInformation(
                "Successfully assembled final article in HTML format of {Length} characters with {ImageCount} images", 
                finalHtml.Length, input.Images.Count);
                
            return new ArticleResult
            {
                HtmlContent = finalHtml,
                FilePath = localFilePath,
                BlobUrl = string.Empty // No blob URL since we're not uploading
            };
        }
        catch (Exception ex)
        {
            logger.LogError(ex, "Error assembling final article");
            throw;
        }
    }
    
    /// <summary>
    /// Extracts paragraphs from HTML content
    /// </summary>
    private List<string> ExtractParagraphs(string html)
    {
        var paragraphs = new List<string>();
        var matches = Regex.Matches(html, @"<p>.*?</p>|<h\d>.*?</h\d>", RegexOptions.Singleline);
        
        foreach (Match match in matches)
        {
            paragraphs.Add(match.Value);
        }
        
        return paragraphs;
    }
    
    /// <summary>
    /// Extracts text between HTML tags
    /// </summary>
    private string ExtractTextBetweenTags(string html, string tag)
    {
        var match = Regex.Match(
            html, 
            $"<{tag}>(.*?)</{tag}>", 
            RegexOptions.Singleline);
            
        return match.Success ? match.Groups[1].Value : string.Empty;
    }
    
    /// <summary>
    /// Sanitizes a string for use in a filename
    /// </summary>
    private string SanitizeForFileName(string input)
    {
        // Remove invalid filename characters
        string invalidChars = Regex.Escape(new string(Path.GetInvalidFileNameChars()));
        string invalidRegex = $"[{invalidChars}]";
        string result = Regex.Replace(input, invalidRegex, "-");
        
        // Limit length and trim
        result = result.Substring(0, Math.Min(50, result.Length)).Trim('-');
        
        // Replace spaces with dashes
        result = Regex.Replace(result, @"\s+", "-");
        
        return result;
    }
}
