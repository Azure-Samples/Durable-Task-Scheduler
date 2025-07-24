# Agent Chaining Sample with Durable Task SDK for .NET

This sample demonstrates an AI-powered news article generator using the Durable Task .NET SDK. The workflow chains multiple specialized AI agents to research topics, generate content, create images, and produce an HTML article.

## What This Sample Does

This application chains together three AI agents to create a complete news article:

1. **Research Agent**: Researches a topic using web search capabilities
2. **Content Generation Agent**: Creates article content based on research findings
3. **Image Generation Agent**: Creates images for the article using DALL-E

The workflow is orchestrated using the Durable Task SDK, which handles the workflow coordination and reliability. When complete, the article is saved as an HTML file with text content and images.

![Agent Chaining Architecture](images/architecture.png)

## Prerequisites

- [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0)
- [Docker](https://www.docker.com/products/docker-desktop/) (for the Durable Task Scheduler emulator)
- Azure AI Projects service
- Azure credentials (via az login)
- Optional: Azure OpenAI service with DALL-E 3

## Running the Sample Locally

### Quick Start Guide

1. **Start the Durable Task Scheduler Emulator**

   ```bash
   # Pull and run the emulator
   docker pull mcr.microsoft.com/dts/dts-emulator:latest
   docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
   ```

2. **Set Environment Variables**

   ```bash
   # Required: Azure AI Projects endpoint
   export AGENT_CONNECTION_STRING="https://your-ai-project-endpoint.services.ai.azure.com/api/projects/your-project-id"

   # Optional: OpenAI model name (defaults to "gpt-4")
   export OPENAI_DEPLOYMENT_NAME="gpt-4-turbo"

   # Optional: For image generation (if omitted, placeholder images will be used)
   export DALLE_ENDPOINT="https://your-resource.openai.azure.com/openai/deployments/dall-e-3/images/generations?api-version=2024-02-01"
   ```

   **Note for Windows users:**
   - Command Prompt: Use `set` instead of `export`
   - PowerShell: Use `$env:VARIABLE_NAME="value"`

3. **Build and Run the Application**

   In the first terminal:
   ```bash
   dotnet build
   dotnet run --project Worker/Worker.csproj
   ```

   In a second terminal:
   ```bash
   dotnet run --project Client/Client.csproj
   ```

4. **Generate an Article**

   Follow the prompts in the client console to enter a news topic. The application will:
   - Research the topic
   - Generate article content
   - Create supporting images 
   - Save an HTML file with the complete article

   When finished, the client will show the path to your generated HTML file (typically in a temp directory like `/var/folders/.../T/article-generator/` on macOS).
## How It Works

The application uses a durable orchestration workflow with four key steps:

1. **Research**: The Research Agent gathers facts, sources, and angles for your topic
2. **Content Creation**: The Content Generation Agent writes a professional news article
3. **Image Generation**: The Image Generation Agent creates descriptive images with DALL-E
4. **Assembly**: All content is combined into a styled HTML document

![sample-article](images/sample-article.png)

## Authentication

The sample uses Azure's DefaultAzureCredential for authentication:

```bash
# Login with your Azure account
az login

# Select the correct subscription if needed
az account set --subscription "your-subscription-name-or-id"
```

Ensure your account has:
- "AI Project User" role for Azure AI Projects
- "Cognitive Services OpenAI User" role for Azure OpenAI (if using DALL-E)

## View the Orchestration Dashboard

Monitor your workflow executions in the Durable Task Scheduler dashboard:
1. Open a browser and go to `http://localhost:8082`
2. Select the `default` task hub
3. Check the Instances tab to see your orchestrations

## Learn More

- [Durable Task SDK for .NET](https://github.com/microsoft/durabletask-dotnet)
- [Azure AI Projects Documentation](https://learn.microsoft.com/azure/ai-services/ai-project/overview)
- [Azure OpenAI Service](https://learn.microsoft.com/azure/ai-services/openai/)
