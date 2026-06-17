using Microsoft.DurableTask;

namespace Demo.Codegen.MainApp;

/// <summary>
/// 3-step workflow that answers a question over a CSV using LLM-generated Python.
/// </summary>
[DurableTask(nameof(AnalyzeSalesOrchestrator))]
internal sealed class AnalyzeSalesOrchestrator : TaskOrchestrator<AnalyzeSalesInput, string>
{
    public override async Task<string> RunAsync(TaskOrchestrationContext context, AnalyzeSalesInput input)
    {
        // Generate one chunk-friendly Python script up front and reuse it for every region.
        string pythonCode = await context.CallActivityAsync<string>(
            TaskNames.GenerateCode,
            input.Question);

        // Fan out: one sandbox execution per region-specific CSV partition.
        RegionChunk[] chunks = SplitCsvByRegion(input.CsvData);
        Task<ExecuteCodeOutput>[] executions = chunks
            .Select(chunk => context.CallActivityAsync<ExecuteCodeOutput>(
                TaskNames.ExecuteCode,
                new ExecuteCodeInput(pythonCode, chunk.CsvData)))
            .ToArray();

        // Fan in: wait for every sandbox result, then hand the set to the formatter.
        ExecuteCodeOutput[] results = await Task.WhenAll(executions);
        RegionExecutionResult[] regionResults = chunks
            .Zip(results, (chunk, execution) => new RegionExecutionResult(chunk.Region, execution))
            .ToArray();

        return await context.CallActivityAsync<string>(
            TaskNames.FormatAnswer,
            new FormatAnswerInput(input.Question, regionResults));
    }








    static RegionChunk[] SplitCsvByRegion(string csvData)
    {
        // Partition the dataset deterministically so the same generated script can run once per region.
        string normalized = csvData.Replace("\r\n", "\n", StringComparison.Ordinal);
        string[] lines = normalized.Split('\n', StringSplitOptions.RemoveEmptyEntries);
        if (lines.Length < 2)
        {
            return [];
        }

        string header = lines[0];
        Dictionary<string, List<string>> rowsByRegion = new(StringComparer.OrdinalIgnoreCase);

        foreach (string row in lines.Skip(1))
        {
            string[] cells = row.Split(',');
            if (cells.Length < 2)
            {
                continue;
            }

            string region = cells[1].Trim();
            if (!rowsByRegion.TryGetValue(region, out List<string>? rows))
            {
                rows = [];
                rowsByRegion.Add(region, rows);
            }

            rows.Add(row);
        }

        return rowsByRegion
            .OrderBy(pair => pair.Key, StringComparer.OrdinalIgnoreCase)
            .Select(pair => new RegionChunk(pair.Key, string.Join('\n', new[] { header }.Concat(pair.Value))))
            .ToArray();
    }
}
