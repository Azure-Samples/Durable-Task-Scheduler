using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;
using DurableTaskOnAKS.Models;

namespace DurableTaskOnAKS;

/// <summary>
/// Classifies a document along one dimension (Sentiment, Topic, or Priority).
/// Three instances run in parallel during the fan-out step.
/// </summary>
public class ClassifyDocument : TaskActivity<ClassifyRequest, ClassificationResult>
{
    private readonly ILogger<ClassifyDocument> _log;
    public ClassifyDocument(ILogger<ClassifyDocument> log) => _log = log;

    public override async Task<ClassificationResult> RunAsync(
        TaskActivityContext context, ClassifyRequest req)
    {
        _log.LogInformation("Classifying '{Category}' for {Id}", req.Category, req.DocumentId);
        await Task.Delay(200); // simulate calling a classification service

        // Stub results — replace with real ML / API calls
        var (label, confidence) = req.Category switch
        {
            "Sentiment" => ("Positive", 0.85),
            "Topic"     => ("Technology", 0.92),
            "Priority"  => ("Normal", 0.78),
            _           => ("Unknown", 0.50),
        };

        _log.LogInformation("{Category} = {Label} ({Conf:P0})", req.Category, label, confidence);
        return new ClassificationResult(req.Category, label, confidence);
    }
}
