namespace Demo.Codegen.MainApp;

/// <summary>
/// Input passed from the orchestrator to the on-demand sandbox ExecuteCode activity.
/// The activity writes the CSV to disk and runs the Python script against it.
/// </summary>
public sealed record ExecuteCodeInput(string PythonCode, string CsvData);

/// <summary>
/// Output of the sandboxed Python execution.
/// </summary>
public sealed record ExecuteCodeOutput(string Stdout, string Stderr, int ExitCode);

/// <summary>
/// A deterministic CSV partition used to fan out on-demand sandbox executions.
/// </summary>
public sealed record RegionChunk(string Region, string CsvData);

/// <summary>
/// Captures the sandbox execution result for a single region partition.
/// </summary>
public sealed record RegionExecutionResult(string Region, ExecuteCodeOutput Execution);

/// <summary>
/// Input to the orchestrator: a natural-language question and the CSV to answer it over.
/// </summary>
public sealed record AnalyzeSalesInput(string Question, string CsvData);

/// <summary>
/// Input to the final formatting activity after fan-out/fan-in completes.
/// </summary>
public sealed record FormatAnswerInput(string Question, RegionExecutionResult[] Results);
