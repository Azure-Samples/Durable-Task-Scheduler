using System.Text.Json;
using Microsoft.Extensions.Logging;
using TravelPlannerFunctions.Models;

namespace TravelPlannerFunctions.Services;

/// <summary>
/// Agent service for destination recommendations
/// </summary>
public class DestinationRecommenderService : BaseAgentService
{
    private const string AgentIdEnvVar = "DESTINATION_RECOMMENDER_AGENT_ID";

    public DestinationRecommenderService(string connectionString, ILogger<DestinationRecommenderService> logger)
        : base(GetAgentIdentifier(AgentIdEnvVar), connectionString, logger)
    {
    }

    public async Task<DestinationRecommendations> GetDestinationRecommendationsAsync(TravelRequest request)
    {
        try
        {
            // Create the prompt for the destination recommender agent
            var prompt = $"Based on the following preferences, recommend 3 travel destinations:\n" +
                        $"User: {request.UserName}\n" +
                        $"Preferences: {request.Preferences}\n" +
                        $"Duration: {request.DurationInDays} days\n" +
                        $"Budget: {request.Budget}\n" +
                        $"Travel Dates: {request.TravelDates}\n" +
                        $"Special Requirements: {request.SpecialRequirements}\n\n" +
                        $"Provide a detailed explanation for each recommendation highlighting why it matches the user's preferences." +
                        $"Format the response as a JSON object with a 'recommendations' array containing objects with 'destinationName', " +
                        $"'description', 'reasoning', and 'matchScore' (0-100) properties.";

            // Get agent response
            var output = await GetAgentResponseAsync(prompt);
            
            // Clean the output in case it's wrapped in markdown code blocks
            output = CleanJsonResponse(output);
            
            // Parse the JSON response
            return JsonSerializer.Deserialize<DestinationRecommendations>(output, JsonOptions) 
                   ?? new DestinationRecommendations(new List<DestinationRecommendation>());
        }
        catch (Exception ex)
        {
            // Log the exception in a real-world scenario
            Logger.LogInformation($"Error calling destination recommender agent: {ex.Message}");
            
            // If anything fails, return an empty list
            return new DestinationRecommendations(new List<DestinationRecommendation>());
        }
    }
}

/// <summary>
/// Agent service for itinerary planning
/// </summary>
public class ItineraryPlannerService : BaseAgentService
{
    private const string AgentIdEnvVar = "ITINERARY_PLANNER_AGENT_ID";

    public ItineraryPlannerService(string connectionString, ILogger<ItineraryPlannerService> logger) 
        : base(GetAgentIdentifier(AgentIdEnvVar), connectionString, logger)
    {
    }

    public async Task<TravelItinerary> CreateItineraryAsync(TravelItineraryRequest request)
    {
        try
        {
            // Create the prompt for the itinerary planner agent
            var prompt = $"Create a detailed daily itinerary for a trip to {request.DestinationName}:\n" +
                         $"Duration: {request.DurationInDays} days\n" +
                         $"Budget: {request.Budget}\n" +
                         $"Travel Dates: {request.TravelDates}\n" +
                         $"Special Requirements: {request.SpecialRequirements}\n\n" +
                         $"Include a mix of sightseeing, cultural activities, and relaxation time.\n" +
                         $"For each day, provide a schedule with specific times, activities, locations, and REALISTIC estimated costs in dollars (e.g. '$25', '$75-100').\n" +
                         $"The estimated costs are very important and should reflect reasonable prices for each activity based on the budget of {request.Budget}.\n" +
                         $"Format the response as a JSON object with 'destinationName', 'travelDates', 'dailyPlan' (array of days), " +
                         $"'estimatedTotalCost', and 'additionalNotes' properties. Each day should have 'day', 'date', and 'activities' properties. " +
                         $"Each activity should have 'time', 'activityName', 'description', 'location', and 'estimatedCost' properties (with dollar amounts).";

            Logger.LogInformation($"[DEBUG-ITINERARY] Agent ID: {AgentId}");
            Logger.LogInformation($"[DEBUG-ITINERARY] Env Var: {AgentIdEnvVar} = {Environment.GetEnvironmentVariable(AgentIdEnvVar) ?? "NOT SET"}");
            Logger.LogInformation($"[DEBUG-ITINERARY] Connection available: {!string.IsNullOrEmpty(ConnectionString)}");
            Logger.LogInformation($"[DEBUG-ITINERARY] Sending prompt to agent:\n{prompt}");

            // Get agent response
            var output = await GetAgentResponseAsync(prompt);
            
            Logger.LogInformation($"[DEBUG-ITINERARY] Raw agent response:\n{output}");
            
            // Clean the output in case it's wrapped in markdown code blocks
            output = CleanJsonResponse(output);
            Logger.LogInformation($"[DEBUG-ITINERARY] Cleaned JSON response:\n{output.Substring(0, Math.Min(200, output.Length))}...");
            
            try {
                // Try to parse the JSON response directly
                Logger.LogInformation("[DEBUG-ITINERARY] Attempting direct JSON deserialization");
                var result = JsonSerializer.Deserialize<TravelItinerary>(output, JsonOptions);
                if (result != null) {
                    // Print structure of the deserialized object for debugging
                    Logger.LogInformation($"[DEBUG-ITINERARY] Successfully deserialized! Destination: {result.DestinationName}");
                    Logger.LogInformation($"[DEBUG-ITINERARY] Daily plan count: {result.DailyPlan?.Count ?? 0}");
                    if (result.DailyPlan != null && result.DailyPlan.Count > 0) {
                        Logger.LogInformation($"[DEBUG-ITINERARY] First day has {result.DailyPlan[0].Activities?.Count ?? 0} activities");
                    }
                    return result;
                } else {
                    Logger.LogInformation("[DEBUG-ITINERARY] Deserialization returned null object");
                }
            }
            catch (JsonException ex) {
                // Log the specific JSON error
                Logger.LogInformation($"[DEBUG-ITINERARY] JSON parsing error: {ex.Message}. Attempting to fix the JSON...");
                
                // Try to parse the JSON as a dynamic object to handle potential format issues
                try {
                    Logger.LogInformation("[DEBUG-ITINERARY] Attempting manual JSON document parsing");
                    using (JsonDocument doc = JsonDocument.Parse(output))
                    {
                        // Check if we have a dailyPlan property and debug its structure
                        if (doc.RootElement.TryGetProperty("dailyPlan", out var dailyPlanElement)) {
                            Logger.LogInformation($"[DEBUG-ITINERARY] Found dailyPlan element with type: {dailyPlanElement.ValueKind}");
                            if (dailyPlanElement.ValueKind == JsonValueKind.Array) {
                                Logger.LogInformation($"[DEBUG-ITINERARY] dailyPlan array has {dailyPlanElement.GetArrayLength()} elements");
                            }
                        } else {
                            Logger.LogInformation("[DEBUG-ITINERARY] dailyPlan property not found in JSON");
                        }
                        
                        // Create a new travel itinerary with normalized properties
                        var parsedDailyPlan = ParseDailyPlan(doc.RootElement);
                        Logger.LogInformation($"[DEBUG-ITINERARY] Parsed daily plan count: {parsedDailyPlan.Count}");
                        
                        var result = new TravelItinerary(
                            GetJsonPropertyAsString(doc.RootElement, "destinationName") ?? request.DestinationName,
                            GetJsonPropertyAsString(doc.RootElement, "travelDates") ?? request.TravelDates,
                            parsedDailyPlan,
                            GetJsonPropertyAsString(doc.RootElement, "estimatedTotalCost") ?? "0",
                            GetJsonPropertyAsString(doc.RootElement, "additionalNotes") ?? "Generated itinerary"
                        );
                        
                        Logger.LogInformation($"[DEBUG-ITINERARY] Manual parsing complete: Destination = {result.DestinationName}, Days = {result.DailyPlan.Count}");
                        return result;
                    }
                }
                catch (Exception innerEx) {
                    Logger.LogInformation($"[DEBUG-ITINERARY] Failed to manually parse JSON: {innerEx.Message}");
                    Logger.LogInformation($"[DEBUG-ITINERARY] Exception type: {innerEx.GetType().Name}");
                    Logger.LogInformation($"[DEBUG-ITINERARY] Stack trace: {innerEx.StackTrace}");
                    throw; // Re-throw to be caught by outer catch
                }
            }
            
            // If we got here without returning, create an empty itinerary
            Logger.LogInformation("[DEBUG-ITINERARY] No successful parsing path, creating empty itinerary");
            return CreateEmptyItinerary(request.DestinationName, request.TravelDates);
        }
        catch (Exception ex)
        {
            // Log the exception in a real-world scenario
            Logger.LogInformation($"Error calling itinerary planner agent: {ex.Message}");
            
            // If anything fails, return an empty itinerary
            return CreateEmptyItinerary(request.DestinationName, request.TravelDates);
        }
    }

    private string? GetJsonPropertyAsString(JsonElement element, string propertyName)
    {
        if (element.TryGetProperty(propertyName, out JsonElement prop))
        {
            return prop.ValueKind == JsonValueKind.String ? prop.GetString() : prop.ToString();
        }
        return null;
    }

    private List<ItineraryDay> ParseDailyPlan(JsonElement root)
    {
        var result = new List<ItineraryDay>();
        
        if (root.TryGetProperty("dailyPlan", out JsonElement dailyPlanElement) && 
            dailyPlanElement.ValueKind == JsonValueKind.Array)
        {
            foreach (JsonElement dayElement in dailyPlanElement.EnumerateArray())
            {
                try
                {
                    int day = 1;
                    string date = "Unknown";
                    var activities = new List<ItineraryActivity>();

                    // Parse day number
                    if (dayElement.TryGetProperty("day", out JsonElement dayNumberElement))
                    {
                        if (dayNumberElement.ValueKind == JsonValueKind.Number)
                        {
                            day = dayNumberElement.GetInt32();
                        }
                        else if (dayNumberElement.ValueKind == JsonValueKind.String &&
                                 int.TryParse(dayNumberElement.GetString(), out int parsedDay))
                        {
                            day = parsedDay;
                        }
                    }

                    // Parse date
                    if (dayElement.TryGetProperty("date", out JsonElement dateElement) && 
                        dateElement.ValueKind == JsonValueKind.String)
                    {
                        date = dateElement.GetString() ?? "Unknown";
                    }

                    // Parse activities
                    if (dayElement.TryGetProperty("activities", out JsonElement activitiesElement) && 
                        activitiesElement.ValueKind == JsonValueKind.Array)
                    {
                        foreach (JsonElement activityElement in activitiesElement.EnumerateArray())
                        {
                            string time = "Unknown";
                            string activityName = "Unknown";
                            string description = "";
                            string location = "Unknown";
                            string estimatedCost = "0";

                            if (activityElement.TryGetProperty("time", out JsonElement timeElement) && 
                                timeElement.ValueKind == JsonValueKind.String)
                            {
                                time = timeElement.GetString() ?? "Unknown";
                            }

                            if (activityElement.TryGetProperty("activityName", out JsonElement nameElement) && 
                                nameElement.ValueKind == JsonValueKind.String)
                            {
                                activityName = nameElement.GetString() ?? "Unknown";
                            }

                            if (activityElement.TryGetProperty("description", out JsonElement descElement) && 
                                descElement.ValueKind == JsonValueKind.String)
                            {
                                description = descElement.GetString() ?? "";
                            }

                            if (activityElement.TryGetProperty("location", out JsonElement locElement) && 
                                locElement.ValueKind == JsonValueKind.String)
                            {
                                location = locElement.GetString() ?? "Unknown";
                            }

                            if (activityElement.TryGetProperty("estimatedCost", out JsonElement costElement) && 
                                costElement.ValueKind == JsonValueKind.String)
                            {
                                estimatedCost = costElement.GetString() ?? "0";
                            }

                            activities.Add(new ItineraryActivity(time, activityName, description, location, estimatedCost));
                        }
                    }

                    result.Add(new ItineraryDay(day, date, activities));
                }
                catch (Exception ex)
                {
                    Logger.LogInformation($"Error parsing day element: {ex.Message}");
                    // Continue with the next day instead of failing the entire parsing
                }
            }
        }
        
        return result;
    }

    private TravelItinerary CreateEmptyItinerary(string destinationName, string travelDates)
    {
        return new TravelItinerary(
            destinationName,
            travelDates,
            new List<ItineraryDay>(),
            "0",
            "Unable to generate itinerary"
        );
    }
}

/// <summary>
/// Agent service for local recommendations
/// </summary>
public class LocalRecommendationsService : BaseAgentService
{
    private const string AgentIdEnvVar = "LOCAL_RECOMMENDATIONS_AGENT_ID";

    public LocalRecommendationsService(string connectionString, ILogger<LocalRecommendationsService> logger) 
        : base(GetAgentIdentifier(AgentIdEnvVar), connectionString, logger)
    {
    }

    public async Task<LocalRecommendations> GetLocalRecommendationsAsync(LocalRecommendationsRequest request)
    {
        try
        {
            // Create the prompt for the local recommendations agent
            var prompt = $"Provide local recommendations for {request.DestinationName}:\n" +
                         $"Duration of Stay: {request.DurationInDays} days\n" +
                         $"Preferred Cuisine: {request.PreferredCuisine}\n" +
                         $"Include Hidden Gems: {request.IncludeHiddenGems}\n" +
                         $"Family Friendly: {request.FamilyFriendly}\n\n" +
                         $"Provide a list of local attractions and restaurants that visitors should check out.\n" +
                         $"Include insider tips that only locals would know.\n" +
                         $"Format the response as a JSON object with 'attractions' and 'restaurants' arrays, and an 'insiderTips' string.\n" +
                         $"Each attraction should have 'name', 'category', 'description', 'location', 'visitDuration', 'estimatedCost', and 'rating' properties.\n" +
                         $"Each restaurant should have 'name', 'cuisine', 'description', 'location', 'priceRange', and 'rating' properties.";

            // Get agent response
            var output = await GetAgentResponseAsync(prompt);
            
            // Clean the output in case it's wrapped in markdown code blocks
            output = CleanJsonResponse(output);
            
            // Parse the JSON response
            return JsonSerializer.Deserialize<LocalRecommendations>(output, JsonOptions) 
                   ?? CreateEmptyLocalRecommendations();
        }
        catch (Exception ex)
        {
            // Log the exception in a real-world scenario
            Logger.LogInformation($"Error calling local recommendations agent: {ex.Message}");
            
            // If anything fails, return empty recommendations
            return CreateEmptyLocalRecommendations();
        }
    }

    private LocalRecommendations CreateEmptyLocalRecommendations()
    {
        return new LocalRecommendations(
            new List<Attraction>(),
            new List<Restaurant>(),
            "Unable to generate local recommendations"
        );
    }
}