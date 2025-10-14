using System.Text.Json;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.AI;
using Microsoft.Agents.AI;
using TravelPlannerFunctions.Models;

namespace TravelPlannerFunctions.Services;

public class DestinationRecommenderService
{
    private readonly AIAgent _agent;
    private readonly ILogger<DestinationRecommenderService> _logger;
    private readonly JsonSerializerOptions _jsonOptions;

    private const string Instructions = @"You are a travel destination expert who recommends destinations based on user preferences.
Based on the user's preferences, budget, duration, travel dates, and special requirements, recommend 3 travel destinations.
Provide a detailed explanation for each recommendation highlighting why it matches the user's preferences.";

    public DestinationRecommenderService(IChatClient chatClient, ILogger<DestinationRecommenderService> logger)
    {
        _logger = logger;
        _agent = new ChatClientAgent(chatClient, new ChatClientAgentOptions
        {
            Name = "DestinationRecommender",
            Instructions = Instructions,
            ChatOptions = new()
            {
                ResponseFormat = ChatResponseFormat.ForJsonSchema(
                    AIJsonUtilities.CreateJsonSchema(typeof(DestinationRecommendations))
                )
            }
        });
        _jsonOptions = new JsonSerializerOptions
        {
            PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
            PropertyNameCaseInsensitive = true
        };
    }

    public async Task<DestinationRecommendations> GetDestinationRecommendationsAsync(TravelRequest request)
    {
        try
        {
            var prompt = $@"Based on the following preferences, recommend 3 travel destinations:
User: {request.UserName}
Preferences: {request.Preferences}
Duration: {request.DurationInDays} days
Budget: {request.Budget}
Travel Dates: {request.TravelDates}
Special Requirements: {request.SpecialRequirements}";

            var response = await _agent.RunAsync(prompt);
            return JsonSerializer.Deserialize<DestinationRecommendations>(response.Text ?? "{}", _jsonOptions) 
                   ?? new DestinationRecommendations(new List<DestinationRecommendation>());
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error calling destination recommender agent");
            return new DestinationRecommendations(new List<DestinationRecommendation>());
        }
    }
}

public class ItineraryPlannerService
{
    private readonly AIAgent _agent;
    private readonly ILogger<ItineraryPlannerService> _logger;

    private const string Instructions = "You are a travel itinerary planner. Create concise day-by-day travel plans with key activities and timing.";

    public ItineraryPlannerService(IChatClient chatClient, ILogger<ItineraryPlannerService> logger)
    {
        _logger = logger;
        _agent = new ChatClientAgent(chatClient, new ChatClientAgentOptions
        {
            Name = "ItineraryPlanner",
            Instructions = Instructions,
            ChatOptions = new()
            {
                ResponseFormat = ChatResponseFormat.ForJsonSchema<TravelItinerary>(),
                MaxOutputTokens = 8000,
                Temperature = 0.7f
            }
        });
    }

    public async Task<TravelItinerary> CreateItineraryAsync(TravelItineraryRequest request)
    {
        try
        {
            var prompt = $@"Create {request.DurationInDays}-day itinerary for {request.DestinationName}.
Dates: {request.TravelDates}
Budget: {request.Budget}
Requirements: {request.SpecialRequirements}

Keep descriptions brief and focused on essential details.";

            var response = await _agent.RunAsync(prompt);
            
            // Deserialize using standard JsonSerializer with reflection
            var jsonText = response.Text ?? "{}";
            var result = JsonSerializer.Deserialize<TravelItinerary>(jsonText, new JsonSerializerOptions 
            { 
                PropertyNameCaseInsensitive = true 
            });
            
            // Ensure we have a valid result with non-null DailyPlan
            if (result == null || result.DailyPlan == null)
            {
                _logger.LogWarning("Deserialized itinerary has null values, returning default");
                return new TravelItinerary(request.DestinationName, request.TravelDates, new List<ItineraryDay>(), "$0", "");
            }
            
            return result;
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error creating itinerary");
            return new TravelItinerary(request.DestinationName, request.TravelDates, new List<ItineraryDay>(), "$0", 
                "Error generating itinerary");
        }
    }
}

public class LocalRecommendationsService
{
    private readonly AIAgent _agent;
    private readonly ILogger<LocalRecommendationsService> _logger;
    private readonly JsonSerializerOptions _jsonOptions;

    private const string Instructions = @"You are a local expert who provides recommendations for restaurants and attractions.
Provide specific recommendations with practical details like operating hours, pricing, and tips.";

    public LocalRecommendationsService(IChatClient chatClient, ILogger<LocalRecommendationsService> logger)
    {
        _logger = logger;
        _agent = new ChatClientAgent(chatClient, new ChatClientAgentOptions
        {
            Name = "LocalRecommendations",
            Instructions = Instructions,
            ChatOptions = new()
            {
                ResponseFormat = ChatResponseFormat.ForJsonSchema(
                    AIJsonUtilities.CreateJsonSchema(typeof(LocalRecommendations))
                )
            }
        });
        _jsonOptions = new JsonSerializerOptions
        {
            PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
            PropertyNameCaseInsensitive = true
        };
    }

    public async Task<LocalRecommendations> GetLocalRecommendationsAsync(LocalRecommendationsRequest request)
    {
        try
        {
            var prompt = $@"Provide local recommendations for {request.DestinationName}:
Duration: {request.DurationInDays} days
Preferred Cuisine: {request.PreferredCuisine}
Include Hidden Gems: {request.IncludeHiddenGems}
Family Friendly: {request.FamilyFriendly}";

            var response = await _agent.RunAsync(prompt);
            return JsonSerializer.Deserialize<LocalRecommendations>(response.Text ?? "{}", _jsonOptions) 
                   ?? new LocalRecommendations(new List<Attraction>(), new List<Restaurant>(), "");
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error calling local recommendations agent");
            return new LocalRecommendations(new List<Attraction>(), new List<Restaurant>(), "");
        }
    }
}
