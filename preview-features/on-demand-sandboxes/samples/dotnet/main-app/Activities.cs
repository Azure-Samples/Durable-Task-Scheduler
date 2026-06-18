using System.ClientModel;
using System.Globalization;
using Azure.Identity;
using Microsoft.DurableTask;
using OpenAI.Chat;

namespace Demo.Codegen.MainApp;

/// <summary>
/// In-process activity. Calls Azure OpenAI to translate a natural-language question
/// into a self-contained pandas script that reads /tmp/data.csv and prints the answer.
/// </summary>
[DurableTask(TaskNames.GenerateCode)]
internal sealed class GenerateCodeActivity : TaskActivity<string, string>
{
    const string SystemPrompt = """
        You are a Python code generator. Given a question about a sales dataset,
        produce a single self-contained Python script that:

        1. Reads /tmp/data.csv with pandas. The columns are: date, region, product, units, revenue.
        2. Assumes the CSV contains rows for exactly one region.
        3. Computes the total revenue for March 2025 in this subset.
        4. Prints ONLY the numeric revenue total to stdout. No code fences, no explanation, no commentary.

        Constraints:
        - Use only the Python standard library and pandas.
        - Do not access the network or filesystem outside /tmp.
        - If there is no March 2025 data in this subset, print 0.
        - Output must be plain text containing only the number.

        Respond with the Python script only. No markdown, no backticks.
        """;

    public override async Task<string> RunAsync(TaskActivityContext context, string question)
    {
        string endpoint = GetRequired("AOAI_ENDPOINT");
        string deployment = GetRequired("AOAI_DEPLOYMENT");

        var client = new Azure.AI.OpenAI.AzureOpenAIClient(
            new Uri(endpoint),
            new DefaultAzureCredential());

        ChatClient chat = client.GetChatClient(deployment);

        ChatCompletion completion = await chat.CompleteChatAsync(
            new SystemChatMessage(SystemPrompt),
            new UserChatMessage(question));

        string code = StripCodeFences(completion.Content[0].Text ?? string.Empty);
        int lineCount = code.Split('\n').Length;
        Console.WriteLine($"[generate] AOAI returned {lineCount} lines of Python:");
        Console.WriteLine("---");
        Console.WriteLine(code);
        Console.WriteLine("---");
        return code;
    }

    static string GetRequired(string name)
        => Environment.GetEnvironmentVariable(name)
            ?? throw new InvalidOperationException($"Environment variable '{name}' is required.");

    static string StripCodeFences(string code)
    {
        string trimmed = code.Trim();
        if (trimmed.StartsWith("```", StringComparison.Ordinal))
        {
            int firstNewline = trimmed.IndexOf('\n');
            if (firstNewline > 0)
            {
                trimmed = trimmed[(firstNewline + 1)..];
            }

            if (trimmed.EndsWith("```", StringComparison.Ordinal))
            {
                trimmed = trimmed[..^3];
            }
        }

        return trimmed.Trim();
    }
}

/// <summary>
/// In-process activity. Wraps the sandboxed Python output in a friendly answer.
/// Kept deliberately simple - in a real app this might call the LLM again to
/// turn raw output into a sentence.
/// </summary>
[DurableTask(TaskNames.FormatAnswer)]
internal sealed class FormatAnswerActivity : TaskActivity<FormatAnswerInput, string>
{
    public override Task<string> RunAsync(TaskActivityContext context, FormatAnswerInput input)
    {
        foreach (RegionExecutionResult result in input.Results)
        {
            if (result.Execution.ExitCode != 0)
            {
                return Task.FromResult(
                    $"Sandbox execution failed for region '{result.Region}' (exit code {result.Execution.ExitCode}): {result.Execution.Stderr}");
            }
        }

        var totals = new List<(string Region, decimal Revenue)>();
        foreach (RegionExecutionResult result in input.Results)
        {
            string stdout = result.Execution.Stdout.Trim();
            if (!decimal.TryParse(stdout, NumberStyles.Float, CultureInfo.InvariantCulture, out decimal revenue))
            {
                return Task.FromResult(
                    $"Sandbox execution returned a non-numeric result for region '{result.Region}': {stdout}");
            }

            totals.Add((result.Region, revenue));
        }

        foreach ((string region, decimal revenue) in totals.OrderByDescending(total => total.Revenue))
        {
            Console.WriteLine($"[fan-out] Region {region}: {revenue.ToString(CultureInfo.InvariantCulture)}");
        }

        string topRegion = totals
            .OrderByDescending(total => total.Revenue)
            .ThenBy(total => total.Region, StringComparer.Ordinal)
            .First()
            .Region;

        return Task.FromResult($"Q: {input.Question}\nA: {topRegion}");
    }
}
