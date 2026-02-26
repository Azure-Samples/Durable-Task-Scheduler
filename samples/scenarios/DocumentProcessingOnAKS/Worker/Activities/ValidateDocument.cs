using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;
using DurableTaskOnAKS.Models;

namespace DurableTaskOnAKS;

/// <summary>Checks that a document has a title and non-empty content.</summary>
public class ValidateDocument : TaskActivity<DocumentInfo, bool>
{
    private readonly ILogger<ValidateDocument> _log;
    public ValidateDocument(ILogger<ValidateDocument> log) => _log = log;

    public override async Task<bool> RunAsync(TaskActivityContext context, DocumentInfo doc)
    {
        _log.LogInformation("Validating '{Title}'", doc.Title);
        await Task.Delay(100); // simulate I/O

        bool valid = !string.IsNullOrWhiteSpace(doc.Title)
                  && !string.IsNullOrWhiteSpace(doc.Content)
                  && doc.Content.Length <= 10_000;

        _log.LogInformation("'{Title}' valid={Valid}", doc.Title, valid);
        return valid;
    }
}
