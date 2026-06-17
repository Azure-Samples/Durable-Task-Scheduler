using System.Diagnostics;
using Microsoft.DurableTask;

namespace Demo.Codegen.SandboxWorker;

/// <summary>
/// Runs LLM-generated Python in the on-demand sandbox.
///
/// Each invocation gets a fresh container instance. The CSV is written to /tmp/data.csv
/// and the generated script is executed against it. We capture stdout/stderr and the
/// exit code so the orchestrator can surface failures cleanly.
/// </summary>
[DurableTask("ExecuteCode")]
internal sealed class ExecuteCodeActivity : TaskActivity<ExecuteCodeInput, ExecuteCodeOutput>
{
    public override async Task<ExecuteCodeOutput> RunAsync(TaskActivityContext context, ExecuteCodeInput input)
    {
        string sandboxName = Environment.GetEnvironmentVariable("DTS_SANDBOX_ID")
            ?? Environment.MachineName;

        Console.WriteLine($"[sandbox] Starting ExecuteCode in sandbox '{sandboxName}' (pid={Environment.ProcessId})");
        Console.WriteLine("[sandbox] This is isolated on-demand sandbox compute managed by DTS.");

        // Show the audience exactly what untrusted code landed in this sandbox.
        string[] codeLines = input.PythonCode.Split('\n');
        int byteCount = System.Text.Encoding.UTF8.GetByteCount(input.PythonCode);
        Console.WriteLine($"[sandbox] Received generated Python ({codeLines.Length} lines, {byteCount} bytes)");
        Console.WriteLine("[sandbox] --- generated script ---");
        const int maxDisplayLines = 30;
        int displayCount = Math.Min(codeLines.Length, maxDisplayLines);
        for (int i = 0; i < displayCount; i++)
        {
            Console.WriteLine(codeLines[i]);
        }

        if (codeLines.Length > maxDisplayLines)
        {
            Console.WriteLine($"... (truncated, {codeLines.Length - maxDisplayLines} more lines)");
        }

        Console.WriteLine("[sandbox] --- end script ---");

        string workDir = Path.Combine("/tmp", $"run-{Guid.NewGuid():N}");
        Directory.CreateDirectory(workDir);
        Console.WriteLine($"[sandbox] Created isolated work directory: {workDir}");

        string csvPath = Path.Combine(workDir, "data.csv");
        string scriptPath = Path.Combine(workDir, "script.py");

        await File.WriteAllTextAsync(csvPath, input.CsvData);
        await File.WriteAllTextAsync(scriptPath, input.PythonCode);
        Console.WriteLine($"[sandbox] Wrote dataset: {csvPath}");
        Console.WriteLine($"[sandbox] Wrote generated script: {scriptPath}");

        // The generated script reads /tmp/data.csv. Copy into the canonical location
        // so the LLM doesn't need to know about per-invocation working directories.
        File.Copy(csvPath, "/tmp/data.csv", overwrite: true);
        Console.WriteLine("[sandbox] Mounted dataset at expected path: /tmp/data.csv");
        string[] csvLines = input.CsvData.Split('\n', StringSplitOptions.RemoveEmptyEntries);
        if (csvLines.Length > 0)
        {
            string[] csvHeaders = csvLines[0].Split(',');
            int csvRowCount = csvLines.Length - 1;
            Console.WriteLine($"[sandbox] Dataset loaded: {csvRowCount} rows × {csvHeaders.Length} columns [{string.Join(", ", csvHeaders)}]");
        }

        using var process = new Process
        {
            StartInfo = new ProcessStartInfo
            {
                FileName = "python3",
                ArgumentList = { scriptPath },
                RedirectStandardOutput = true,
                RedirectStandardError = true,
                UseShellExecute = false,
                WorkingDirectory = workDir,
            },
        };

        var sw = Stopwatch.StartNew();
        process.Start();
        Console.WriteLine($"[sandbox] Executing command: python3 {scriptPath}");

        // Cap execution time so a runaway script can't hold the sandbox forever.
        using var cts = new CancellationTokenSource(TimeSpan.FromSeconds(30));
        Task<string> stdoutTask = process.StandardOutput.ReadToEndAsync(cts.Token);
        Task<string> stderrTask = process.StandardError.ReadToEndAsync(cts.Token);

        try
        {
            await process.WaitForExitAsync(cts.Token);
        }
        catch (OperationCanceledException)
        {
            try { process.Kill(entireProcessTree: true); }
            catch { /* best effort */ }

            Console.WriteLine("[sandbox] ERROR: Timeout: execution exceeded 30 seconds.");
            return new ExecuteCodeOutput(
                Stdout: string.Empty,
                Stderr: "Execution timed out after 30 seconds.",
                ExitCode: 124);
        }

        sw.Stop();
        string stdout = await stdoutTask;
        string stderr = await stderrTask;

        Console.WriteLine($"[sandbox] Python process completed in {sw.ElapsedMilliseconds}ms (exit code {process.ExitCode})");

        if (!string.IsNullOrWhiteSpace(stdout))
        {
            Console.WriteLine("[sandbox] stdout from generated script:");
            Console.WriteLine(stdout.TrimEnd());
        }
        else
        {
            Console.WriteLine("[sandbox] stdout from generated script: <empty>");
        }

        if (process.ExitCode != 0 && !string.IsNullOrWhiteSpace(stderr))
        {
            Console.WriteLine($"[sandbox] ERROR: {stderr.TrimEnd()}");
        }

        if (process.ExitCode == 0)
        {
            Console.WriteLine("[sandbox] Returning captured stdout to the orchestrator.");
        }

        return new ExecuteCodeOutput(stdout, stderr, process.ExitCode);
    }
}
