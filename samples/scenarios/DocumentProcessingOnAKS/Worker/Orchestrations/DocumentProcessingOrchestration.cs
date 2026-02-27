using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;
using DurableTaskOnAKS.Models;

namespace DurableTaskOnAKS;

/// <summary>
/// Demonstrates activity chaining and fan-out / fan-in.
///
///   1. Validate     — reject malformed documents      (chaining)
///   2. Classify ×3  — sentiment, topic, priority       (fan-out / fan-in)
///   3. Return       — assembled result string
/// </summary>
public class DocumentProcessingOrchestration : TaskOrchestrator<DocumentInfo, string>
{
    public override async Task<string> RunAsync(
        TaskOrchestrationContext context, DocumentInfo doc)
    {
        var log = context.CreateReplaySafeLogger<DocumentProcessingOrchestration>();
        log.LogInformation("Processing '{Title}'", doc.Title);

        // Step 1 — Validate (activity chaining)
        bool isValid = await context.CallActivityAsync<bool>(
            nameof(ValidateDocument), doc);

        if (!isValid)
            return $"Document '{doc.Title}' failed validation.";

        // Step 2 — Fan-out: three classification tasks in parallel
        var tasks = new[]
        {
            context.CallActivityAsync<ClassificationResult>(
                nameof(ClassifyDocument), new ClassifyRequest(doc.Id, doc.Content, "Sentiment")),
            context.CallActivityAsync<ClassificationResult>(
                nameof(ClassifyDocument), new ClassifyRequest(doc.Id, doc.Content, "Topic")),
            context.CallActivityAsync<ClassificationResult>(
                nameof(ClassifyDocument), new ClassifyRequest(doc.Id, doc.Content, "Priority")),
        };

        // Fan-in: wait for all three to complete
        ClassificationResult[] results = await Task.WhenAll(tasks);

        // Assemble result
        string labels = string.Join(", ", results.Select(r => $"{r.Category}={r.Label}"));
        string result = $"Processed '{doc.Title}': {labels}";

        log.LogInformation("{Result}", result);
        return result;
    }
}
