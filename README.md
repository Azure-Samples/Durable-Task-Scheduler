## Azure Durable Task Scheduler & Durable Functions

The durable task scheduler is a solution for durable execution in Azure. Durable execution is a fault-tolerant approach to running code that handles failures and interruptions through automatic retries and state persistence. Scenarios where durable execution is required include distributed transactions, multi-agent orchestration, data processing, infrastructure management, and others. Coupled with a developer orchestration framework like Durable Functions or the Durable Task SDKs, the durable task scheduler enables developers to author stateful apps that run on any compute environment without the need to architect for fault tolerance. 

Developers can use the durable task scheduler with the following orchestration frameworks: 
- [Durable Functions](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-overview) 
- Durable Task SDKs, also referred to as "portable SDKs"

> **Note:** Though the durable task scheduler can also be used with the [Durable Task Framework](https://github.com/Azure/durabletask), we recommend new apps to use the Durable Task SDK over the Durable Task Framework as the former follows more modern .NET conventions and will be the focus of our future investments.

### Use with Durable Functions 
When used with Durable Functions, a feature of Azure Functions, the durable task scheduler plays the role the ["backend provider"](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-storage-providers), where state data is persisted as the app runs. While other backend providers are supported, only the durable task scheduler offers a fully managed experience, which removes operational overhead from users. Additionally, the scheduler offers exceptional performance, reliability, and the ease of monitoring orchestrations. 

### Use with Durable Task SDKs or "portable SDKs"
The Durable Task SDKs provide a lightweight client library for the durable task scheduler. When running orchestrations, apps using these SDKs would make a connection to the scheduler's orchestration engine in Azure. These SDKs are called "portable" because apps that leverage them can be hosted in various compute environments, such as Azure Container Apps, Azure Kubernetes Service, Azure App Service, or VMs. 

![Durable Task Scheduler in all Azure Computes](./media/images/durable-task-sdks/dts-in-all-computes.png)

For more information on how to use the Azure Functions durable task scheduler and to explore its features, please refer to the [official documentation](https://aka.ms/dts-documentation)

## Choosing your orchestration framework
Refer to [Choosing an orchestration framework](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/choose-orchestration-framework) article for guidance on how to pick the framework for your use case. 

## API reference docs
Durable Task SDKs
- [.NET](https://learn.microsoft.com/dotnet/api/microsoft.durabletask?view=durabletask-dotnet-1.x)
- [Python](https://github.com/microsoft/durabletask-python)
- [Java](https://learn.microsoft.com/java/api/com.microsoft.durabletask?view=durabletask-java-1.x)
- JavaScript (coming soon)


Durable Functions
- [.NET (isolated)](https://learn.microsoft.com/dotnet/api/microsoft.azure.functions.worker.extensions.durabletask?view=azure-dotnet)
- [Python](https://learn.microsoft.com/python/api/azure-functions-durable/azure.durable_functions?view=azure-python)
- [Java](https://learn.microsoft.com/java/api/com.microsoft.durabletask.azurefunctions?view=azure-java-stable)
- [JavaScript](https://learn.microsoft.com/javascript/api/durable-functions/?view=azure-node-latest)

## AI-Assisted Development Skills

This repository includes specialized skills for AI coding assistants to help you build durable workflows more effectively. These skills provide best practices, code patterns, and contextual guidance that AI assistants can use to generate high-quality code.

### Supported AI Assistants

The skills are compatible with:
- **[GitHub Copilot](https://github.com/features/copilot)** - Works with GitHub Copilot Chat and Copilot Workspace via custom instructions
- **[Claude Code](https://claude.ai/code)** - Works as Claude skills that provide specialized domain knowledge

### Available Skills

| Skill | Description | Path |
|-------|-------------|------|
| **durable-functions-dotnet** | Build durable workflows using Azure Durable Functions with .NET isolated worker. Covers orchestrations, activities, entities, and patterns like function chaining, fan-out/fan-in, async HTTP APIs, human interaction, monitoring, and stateful aggregators. | [.github/skills/durable-functions-dotnet](.github/skills/durable-functions-dotnet/SKILL.md) |
| **durable-task-dotnet** | Build durable workflows in .NET using the Durable Task SDK (portable SDK). Covers orchestrations, activities, entities, and common patterns without Azure Functions dependency. | [.github/skills/durable-task-dotnet](.github/skills/durable-task-dotnet/SKILL.md) |
| **durable-task-java** | Build durable workflows in Java using the Durable Task SDK. Covers orchestrations, activities, and patterns like function chaining, fan-out/fan-in, human interaction, and monitoring. | [.github/skills/durable-task-java](.github/skills/durable-task-java/SKILL.md) |
| **durable-task-python** | Build durable workflows in Python using the Durable Task SDK. Covers orchestrations, activities, entities, and patterns including function chaining, fan-out/fan-in, human interaction, and stateful agents. | [.github/skills/durable-task-python](.github/skills/durable-task-python/SKILL.md) |

### What the Skills Provide

Each skill includes:
- **Quick start templates** - Minimal setup code to get started quickly
- **Pattern implementations** - Detailed examples for common workflow patterns
- **Determinism rules** - Critical guidance on writing replay-safe orchestration code with WRONG vs CORRECT examples
- **Error handling** - Best practices for exception handling and retry policies
- **Connection & authentication** - Configuration for local development and Azure deployment
- **Code examples** - Production-ready snippets for activities, orchestrators, and client operations

### Using with GitHub Copilot

To use these skills with GitHub Copilot:

1. **VS Code**: Reference the skill files in your Copilot Chat by mentioning them with `#file:.github/skills/durable-task-dotnet/SKILL.md`
2. **Copilot Instructions**: Add the skill paths to your `.github/copilot-instructions.md` file:
   ```markdown
   When working with Durable Task or Durable Functions, reference these skills:
   - .github/skills/durable-functions-dotnet/SKILL.md (for Azure Functions .NET)
   - .github/skills/durable-task-dotnet/SKILL.md (for .NET SDK)
   - .github/skills/durable-task-java/SKILL.md (for Java SDK)
   - .github/skills/durable-task-python/SKILL.md (for Python SDK)
   ```

### Using with Claude Code

To use these skills with Claude Code:

1. **Manual reference**: Ask Claude to read the skill file before generating code:
   ```
   Please read .github/skills/durable-task-python/SKILL.md and then help me create a fan-out/fan-in orchestration
   ```
2. **Automatic loading**: Skills in the `.github/skills` directory are automatically detected by Claude Code when working on relevant files

### Example Prompt

```
Using the durable-task-dotnet skill, create an orchestration that:
1. Validates an order
2. Processes payment (with retry policy)
3. Sends confirmation email
4. Handles failures with compensation
```

## Tell us what you think

Your feedback is essential in shaping the future direction of this product. We encourage you to share your experiences, both the good and the bad. If there are any missing features or capabilities that you would like to see supported in the Durable Task Scheduler, we want to hear about them.

> **Note:** This repo is a sample repo. If you have feature requests or feedback, you can share by dropping us an issue in the [Azure/Durable Task Scheduler repo](https://github.com/azure/Durable-Task-Scheduler) or send an email to our product managers Nick and Lily ([nicholas.greenfield@microsoft.com](mailto:nicholas.greenfield@microsoft.com); [jiayma@microsoft.com](mailto:jiayma@microsoft.com)).
