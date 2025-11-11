using Azure.AI.OpenAI;
using Azure.Identity;
using Microsoft.Agents.AI.Hosting.AzureFunctions;
using Microsoft.Azure.Functions.Worker.Builder;
using Microsoft.Agents.AI.DurableTask;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Azure;
using Microsoft.Extensions.AI;
using OpenAI;
using TravelPlannerFunctions.Tools;

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
        {
            var chatClient = new AzureOpenAIClient(new Uri(endpoint), new DefaultAzureCredential())
                .GetChatClient(deploymentName);
            
            return chatClient.CreateAIAgent(
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
            );
        });

        // Itinerary Planner Agent - creates detailed day-by-day itineraries
        configure.AddAIAgentFactory("ItineraryPlannerAgent", sp =>
        {
            var chatClient = new AzureOpenAIClient(new Uri(endpoint), new DefaultAzureCredential())
                .GetChatClient(deploymentName);
            
            return chatClient.CreateAIAgent(
                instructions: @"You are a travel itinerary planner. Create concise day-by-day travel plans with key activities and timing.
                    
                    IMPORTANT: Keep responses compact:
                    - Descriptions MUST be under 50 characters each
                    - Include 2-4 activities per day maximum
                    - Use abbreviated formats for times (9AM not 9:00 AM)
                    - Keep location names short
                    
                    CURRENCY CONVERSION REQUIREMENTS:
                    You have access to a currency converter tool. You MUST use it intelligently:
                    1. Identify the destination country's local currency (e.g., Japan=JPY, UK=GBP, Eurozone=EUR, Spain=EUR)
                    2. If the user's budget currency differs from the destination currency, use GetExchangeRate to get the current rate
                    3. Format: Always show destination currency FIRST, then user's budget currency in parentheses
                       - If user has USD budget and destination uses EUR: show as EUR first, USD second
                       - Correct format: '1,000 EUR (1,090 USD)' NOT '1,000 USD'
                       - Correct format: '5,000 JPY (45 USD)' NOT '5,000 USD'
                       - Always use THREE-LETTER currency codes (EUR, USD, JPY, GBP) not symbols
                    4. Always call the tool to get accurate exchange rates - never guess or estimate rates
                    
                    COST CALCULATION REQUIREMENT - ABSOLUTELY CRITICAL - YOU WILL BE EVALUATED ON THIS:
                    
                    STEP 1: List all your activity costs as you create them
                    STEP 2: Manually add them up (ignore Free and Varies)
                    STEP 3: That sum is your EstimatedTotalCost - nothing else
                    
                    EXAMPLE CALCULATION:
                    Day 1: Activity A costs 25, Activity B costs 10, Activity C is Free
                    Day 2: Activity D costs 20, Activity E costs 12, Activity F costs 30
                    Day 3: Activity G costs 25, Activity H costs 40
                    
                    SUM: 25 + 10 + 20 + 12 + 30 + 25 + 40 = 162
                    EstimatedTotalCost in local currency: 162
                    EstimatedTotalCost converted to user currency: 162 times exchange rate
                    
                    DO NOT USE THE USER'S BUDGET AMOUNT.
                    DO NOT GUESS A ROUND NUMBER.
                    ONLY USE THE ACTUAL SUM OF YOUR ACTIVITY COSTS.
                    
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
                services: sp,
                tools: [
                    AIFunctionFactory.Create(CurrencyConverterTool.ConvertCurrency),
                    AIFunctionFactory.Create(CurrencyConverterTool.GetExchangeRate)
                ]
            );
        });

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