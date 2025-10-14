using Azure.AI.OpenAI;
using Azure.Identity;
using Microsoft.Azure.Functions.DurableAgents;
using Microsoft.Azure.Functions.Worker.Builder;
using Microsoft.DurableTask.Agents;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Azure;
using OpenAI;

string endpoint = Environment.GetEnvironmentVariable("AZURE_OPENAI_ENDPOINT")
    ?? throw new InvalidOperationException("AZURE_OPENAI_ENDPOINT is not set.");
string deploymentName = Environment.GetEnvironmentVariable("AZURE_OPENAI_DEPLOYMENT_NAME")
    ?? "gpt-4o-mini";

// Build the Functions application with the agents registered.
FunctionsApplicationBuilder builder = FunctionsApplication
    .CreateBuilder(args)
    .ConfigureFunctionsWebApplication()
    .ConfigureDurableAgents(configure =>
    {
        // Destination Recommender Agent - recommends travel destinations based on preferences
        configure.AddAIAgentFactory("DestinationRecommenderAgent", sp =>
            new AzureOpenAIClient(new Uri(endpoint), new DefaultAzureCredential())
                .GetChatClient(deploymentName)
                .CreateAIAgent(
                    instructions: @"You are a travel destination expert who recommends destinations based on user preferences.
                        Based on the user's preferences, budget, duration, travel dates, and special requirements, recommend 3 travel destinations.
                        Provide a detailed explanation for each recommendation highlighting why it matches the user's preferences.
                        Return your response as a JSON object with this structure (use PascalCase for property names):
                        {
                            ""Recommendations"": [
                                {
                                    ""DestinationName"": ""string"",
                                    ""Description"": ""string"",
                                    ""Reasoning"": ""string"",
                                    ""MatchScore"": number (0-100)
                                }
                            ]
                        }",
                    name: "DestinationRecommenderAgent",
                    services: sp
                ));

        // Itinerary Planner Agent - creates detailed day-by-day itineraries
        configure.AddAIAgentFactory("ItineraryPlannerAgent", sp =>
            new AzureOpenAIClient(new Uri(endpoint), new DefaultAzureCredential())
                .GetChatClient(deploymentName)
                .CreateAIAgent(
                    instructions: @"You are a travel itinerary planner. Create concise day-by-day travel plans with key activities and timing.
                        Keep descriptions brief and focused on essential details.
                        Return your response as a JSON object with this structure:
                        {
                            ""DestinationName"": ""string"",
                            ""TravelDates"": ""string"",
                            ""DailyPlan"": [
                                {
                                    ""Day"": number,
                                    ""Date"": ""string"",
                                    ""Activities"": [
                                        {
                                            ""Time"": ""string"",
                                            ""ActivityName"": ""string"",
                                            ""Description"": ""string"",
                                            ""Location"": ""string"",
                                            ""EstimatedCost"": ""string""
                                        }
                                    ]
                                }
                            ],
                            ""EstimatedTotalCost"": ""string"",
                            ""AdditionalNotes"": ""string""
                        }",
                    name: "ItineraryPlannerAgent",
                    services: sp
                ));

        // Local Recommendations Agent - provides restaurant and attraction recommendations
        configure.AddAIAgentFactory("LocalRecommendationsAgent", sp =>
            new AzureOpenAIClient(new Uri(endpoint), new DefaultAzureCredential())
                .GetChatClient(deploymentName)
                .CreateAIAgent(
                    instructions: @"You are a local expert who provides recommendations for restaurants and attractions.
                        Provide specific recommendations with practical details like operating hours, pricing, and tips.
                        Return your response as a JSON object with this structure:
                        {
                            ""Attractions"": [
                                {
                                    ""Name"": ""string"",
                                    ""Category"": ""string"",
                                    ""Description"": ""string"",
                                    ""Location"": ""string"",
                                    ""VisitDuration"": ""string"",
                                    ""EstimatedCost"": ""string"",
                                    ""Rating"": number
                                }
                            ],
                            ""Restaurants"": [
                                {
                                    ""Name"": ""string"",
                                    ""Cuisine"": ""string"",
                                    ""Description"": ""string"",
                                    ""Location"": ""string"",
                                    ""PriceRange"": ""string"",
                                    ""Rating"": number
                                }
                            ],
                            ""InsiderTips"": ""string""
                        }",
                    name: "LocalRecommendationsAgent",
                    services: sp
                ));
    });

// Configure additional services
builder.Services.AddApplicationInsightsTelemetryWorkerService();
// builder.Services.ConfigureFunctionsApplicationInsights(); // Not available in FunctionsApplication builder pattern

builder.Services.AddAzureClients(clientBuilder =>
{
    // Use DefaultAzureCredential which automatically handles:
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
builder.Services.AddCors(options =>
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

// Build and run the application.
var app = builder.Build();
app.Run();