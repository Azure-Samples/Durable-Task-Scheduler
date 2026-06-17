using Microsoft.DurableTask.Client.AzureManaged;

namespace Demo.Codegen.MainApp;

/// <summary>
/// Declares the on-demand sandbox worker profile that hosts the <see cref="TaskNames.ExecuteCode"/>
/// activity in an isolated sandbox container. The profile id ("code-executor") surfaces in
/// the DTS dashboard under the On-demand Sandboxes tab.
/// </summary>
[SandboxWorkerProfile("code-executor")]
internal sealed class CodeSandboxWorkerProfile : ISandboxWorkerProfile
{
    public void Configure(SandboxWorkerProfileOptions options)
    {
        options.Image.ImageRef = Environment.GetEnvironmentVariable("DTS_SANDBOX_CONTAINER_IMAGE")
            ?? throw new InvalidOperationException("DTS_SANDBOX_CONTAINER_IMAGE is required.");
        options.Image.ManagedIdentityClientId = Environment.GetEnvironmentVariable("DTS_SANDBOX_IMAGE_PULL_UMI_CLIENT_ID")
            ?? throw new InvalidOperationException("DTS_SANDBOX_IMAGE_PULL_UMI_CLIENT_ID is required.");
        options.SchedulerManagedIdentityClientId = Environment.GetEnvironmentVariable("DTS_SANDBOX_SCHEDULER_UMI_CLIENT_ID")
            ?? throw new InvalidOperationException("DTS_SANDBOX_SCHEDULER_UMI_CLIENT_ID is required.");
        options.Cpu = "1000m";
        options.Memory = "2048Mi";
        options.MaxConcurrentActivities = 1;
        options.AddActivity(TaskNames.ExecuteCode, version: "");
    }
}
