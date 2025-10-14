using Microsoft.Azure.Functions.Worker;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.Azure;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.AI;
using Azure.Identity;
using Azure.AI.OpenAI;
using OpenAI.Chat;
using TravelPlannerFunctions.Agents;
using System.ClientModel;

var host = new HostBuilder()
    .ConfigureFunctionsWebApplication()
    .ConfigureServices(services =>
    {
        services.AddApplicationInsightsTelemetryWorkerService();
        services.ConfigureFunctionsApplicationInsights();
        services.AddLogging();
        
        // Configure the shared IChatClient for all agents
        services.AddSingleton<IChatClient>(serviceProvider =>
        {
            var configuration = serviceProvider.GetRequiredService<IConfiguration>();
            
            // Get Azure OpenAI connection information from environment
            var endpoint = configuration["AZURE_OPENAI_ENDPOINT"] 
                ?? throw new InvalidOperationException("AZURE_OPENAI_ENDPOINT environment variable is not configured");
            var deploymentName = configuration["AZURE_OPENAI_DEPLOYMENT_NAME"] ?? "gpt-4o-mini";
            
            // Use DefaultAzureCredential which automatically handles:
            // - Local development: Azure CLI, VS Code, or interactive browser auth
            // - Azure deployment: Managed Identity
            var credential = new DefaultAzureCredential();

            // Create AzureOpenAIClient and wrap it as IChatClient
            var azureClient = new AzureOpenAIClient(new Uri(endpoint), credential);
            var chatClient = azureClient.GetChatClient(deploymentName);

            return chatClient.AsIChatClient();
        });
        
        // Register the specialized agent services
        services.AddSingleton<DestinationRecommenderAgent>();
        services.AddSingleton<ItineraryPlannerAgent>();
        services.AddSingleton<LocalRecommendationsAgent>();

        services.AddAzureClients(clientBuilder =>
        {
            // Use DefaultAzureCredential which automatically handles:
            // - Local development: Azure CLI, VS Code, or interactive browser auth
            // - Azure deployment: Managed Identity
            clientBuilder.UseCredential(new DefaultAzureCredential());

            // If running in local development with Azurite emulator
            var connectionString = Environment.GetEnvironmentVariable("AzureWebJobsStorage");
            if (!string.IsNullOrEmpty(connectionString))
            {
                clientBuilder.AddBlobServiceClient(connectionString);
            }
            // Use the managed identity to connect to the storage account. 
            else
            {
                var storageAccountName = Environment.GetEnvironmentVariable("AzureWebJobsStorage__accountName");
                ArgumentNullException.ThrowIfNullOrEmpty(storageAccountName, "AzureWebJobsStorage__accountName environment variable is not set.");

                clientBuilder.AddBlobServiceClient(
                    new Uri($"https://{storageAccountName}.blob.core.windows.net"),
                    new DefaultAzureCredential());
            }
        });

        // Configure CORS for both local development and Azure Static Web App
        services.AddCors(options =>
        {
            options.AddDefaultPolicy(policy =>
            {
                // Get the Static Web App URL from environment variables (set in Azure)
                var allowedOrigins = Environment.GetEnvironmentVariable("ALLOWED_ORIGINS") ?? "*";

                // Split by comma if multiple origins are provided
                var origins = allowedOrigins.Split(',', StringSplitOptions.RemoveEmptyEntries);

                if (origins.Length == 1 && origins[0] == "*")
                {
                    // For development or if no specific origins are set, allow any origin
                    policy.AllowAnyOrigin()
                          .AllowAnyHeader()
                          .AllowAnyMethod();
                }
                else
                {
                    // For production with specific origins
                    policy.WithOrigins(origins)
                          .AllowAnyHeader()
                          .AllowAnyMethod()
                          .AllowCredentials();
                }
            });
        });
    })
    .ConfigureAppConfiguration(builder =>
    {
        builder.AddEnvironmentVariables();
    })
    .Build();

await host.RunAsync();