namespace DurableTaskOnAKS.Models;

/// <summary>A document submitted for processing.</summary>
public record DocumentInfo(string Id, string Title, string Content);

/// <summary>Input for the classification activity — one per dimension.</summary>
public record ClassifyRequest(string DocumentId, string Content, string Category);

/// <summary>Result from a single classification pass.</summary>
public record ClassificationResult(string Category, string Label, double Confidence);
