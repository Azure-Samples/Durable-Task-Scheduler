namespace Demo.Codegen.SandboxWorker;

/// <summary>
/// Input contract matching the main app's ExecuteCodeInput record.
/// Defined separately here so the worker has no dependency on the main app.
/// </summary>
public sealed record ExecuteCodeInput(string PythonCode, string CsvData);

public sealed record ExecuteCodeOutput(string Stdout, string Stderr, int ExitCode);
