namespace DurableTaskOnAKS.Client.Models;

/// <summary>
/// A document submitted for processing.
/// Shape must match the Worker's <c>DocumentInfo</c> for JSON round-tripping.
/// </summary>
public record DocumentInfo(string Id, string Title, string Content);
