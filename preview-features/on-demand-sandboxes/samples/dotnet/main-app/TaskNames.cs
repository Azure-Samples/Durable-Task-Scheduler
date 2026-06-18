namespace Demo.Codegen.MainApp;

/// <summary>
/// Activity and orchestrator names shared with the sandbox worker.
/// The sandbox worker registers <see cref="ExecuteCode"/> with the same string.
/// </summary>
internal static class TaskNames
{
    public const string AnalyzeSalesOrchestrator = nameof(AnalyzeSalesOrchestrator);
    public const string GenerateCode = nameof(GenerateCode);
    public const string ExecuteCode = nameof(ExecuteCode);
    public const string FormatAnswer = nameof(FormatAnswer);
}
