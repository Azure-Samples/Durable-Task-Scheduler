using Microsoft.Azure.Functions.Worker;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.Azure;
using Microsoft.Extensions.Logging;
using Azure.Identity;
using TravelPlannerFunctions.Services;
using Azure.AI.Projects;

var host = new HostBuilder()
    .ConfigureFunctionsWebApplication()
    .ConfigureServices(services =>
    {
        services.AddApplicationInsightsTelemetryWorkerService();
        services.ConfigureFunctionsApplicationInsights();
        services.AddLogging();
        
        services.AddSingleton<DestinationRecommenderService>((serviceProvider) =>
        {
            
            var configuration = serviceProvider.GetRequiredService<IConfiguration>();
            var loggerFactory = serviceProvider.GetRequiredService<ILoggerFactory>();
            var destinationConnectionString = configuration["DESTINATION_RECOMMENDER_CONNECTION"] ?? throw new InvalidOperationException("Destination recommender connection string is not configured");
            var logger = loggerFactory.CreateLogger<DestinationRecommenderService>();
            return new DestinationRecommenderService(destinationConnectionString, loggerFactory.CreateLogger<DestinationRecommenderService>());
        });
        
        services.AddSingleton<ItineraryPlannerService>((serviceProvider) =>
        {
            var configuration = serviceProvider.GetRequiredService<IConfiguration>();
            var loggerFactory = serviceProvider.GetRequiredService<ILoggerFactory>();
            var itineraryConnectionString = configuration["ITINERARY_PLANNER_CONNECTION"] ?? throw new InvalidOperationException("Itinerary planner connection string is not configured");
            var logger = loggerFactory.CreateLogger<ItineraryPlannerService>();
            return new ItineraryPlannerService(itineraryConnectionString, loggerFactory.CreateLogger<ItineraryPlannerService>());
        });

        services.AddSingleton<LocalRecommendationsService>((serviceProvider) =>
        {
            var configuration = serviceProvider.GetRequiredService<IConfiguration>();
            var loggerFactory = serviceProvider.GetRequiredService<ILoggerFactory>();
            var localRecommendationsConnectionString = configuration["LOCAL_RECOMMENDATIONS_CONNECTION"] ?? throw new InvalidOperationException("Local recommendations connection string is not configured");
            var logger = loggerFactory.CreateLogger<LocalRecommendationsService>();
            return new LocalRecommendationsService(localRecommendationsConnectionString, loggerFactory.CreateLogger<LocalRecommendationsService>());
        });

        services.AddAzureClients(clientBuilder =>
        {
            var tenantId = Environment.GetEnvironmentVariable("AZURE_TENANT_ID");
            var clientId = Environment.GetEnvironmentVariable("AZURE_CLIENT_ID");

            ArgumentNullException.ThrowIfNullOrEmpty(nameof(tenantId), "AZURE_TENANT_ID environment variable is not set.");
            ArgumentNullException.ThrowIfNullOrEmpty(nameof(clientId), "AZURE_CLIENT_ID environment variable is not set.");

            // Use the same credentials for all clients. 
            clientBuilder.UseCredential(new DefaultAzureCredential(new DefaultAzureCredentialOptions
            {
                TenantId = tenantId,
                ManagedIdentityClientId = clientId
            }));

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