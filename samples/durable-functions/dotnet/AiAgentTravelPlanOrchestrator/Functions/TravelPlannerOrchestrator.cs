using System.Text;
using System.Text.Json;
using Microsoft.Azure.Functions.Worker;
using Microsoft.Agents.AI;
using Microsoft.Extensions.Logging;
using TravelPlannerFunctions.Models;
using Microsoft.Agents.AI.Hosting.AzureFunctions;
using Microsoft.Agents.AI.DurableTask;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;

namespace TravelPlannerFunctions.Functions;

public class TravelPlannerOrchestrator
{
    private readonly ILogger _logger;

    public TravelPlannerOrchestrator(ILoggerFactory loggerFactory)
    {
        _logger = loggerFactory.CreateLogger<TravelPlannerOrchestrator>();
    }

    [Function(nameof(RunTravelPlannerOrchestration))]
    public async Task<TravelPlanResult> RunTravelPlannerOrchestration(
        [OrchestrationTrigger] TaskOrchestrationContext context)
    {
        var travelRequest = context.GetInput<TravelRequest>()
            ?? throw new ArgumentNullException(nameof(context), "Travel request input is required");
            
        var logger = context.CreateReplaySafeLogger<TravelPlannerOrchestrator>();
        logger.LogInformation("Starting travel planning orchestration for user {UserName}", travelRequest.UserName);

        // Set initial status
        context.SetCustomStatus(new {
            step = "Starting",
            message = $"Starting travel planning for {travelRequest.UserName}",
            progress = 0
        });

        // Get the durable agents for the travel planning workflow
        DurableAIAgent destinationAgent = context.GetAgent("DestinationRecommenderAgent");
        DurableAIAgent itineraryAgent = context.GetAgent("ItineraryPlannerAgent");
        DurableAIAgent localRecommendationsAgent = context.GetAgent("LocalRecommendationsAgent");

        // Create new threads for each agent
        AgentThread destinationThread = destinationAgent.GetNewThread();
        AgentThread itineraryThread = itineraryAgent.GetNewThread();
        AgentThread localThread = localRecommendationsAgent.GetNewThread();

        // Step 1: Get destination recommendations
        logger.LogInformation("Step 1: Requesting destination recommendations");
        context.SetCustomStatus(new {
            step = "GetDestinationRecommendations",
            message = "Finding the perfect destinations for your travel preferences...",
            progress = 10
        });
        
        var destinationPrompt = $@"Based on the following preferences, recommend 3 travel destinations:
                                User: {travelRequest.UserName}
                                Preferences: {travelRequest.Preferences}
                                Duration: {travelRequest.DurationInDays} days
                                Budget: {travelRequest.Budget}
                                Travel Dates: {travelRequest.TravelDates}
                                Special Requirements: {travelRequest.SpecialRequirements}";

        var destinationResponse = await destinationAgent.RunAsync<DestinationRecommendations>(
            destinationPrompt,
            destinationThread);
        
        var destinationRecommendations = destinationResponse.Result;
            
        if (destinationRecommendations.Recommendations.Count == 0)
        {
            logger.LogWarning("No destination recommendations were generated");
            return new TravelPlanResult(CreateEmptyTravelPlan(), string.Empty);
        }
        
        // For this example, we'll take the top recommendation
        var topDestination = destinationRecommendations.Recommendations
            .OrderByDescending(r => r.MatchScore)
            .First();
            
        logger.LogInformation("Selected top destination: {DestinationName}", topDestination.DestinationName);

        // Step 2: Create an itinerary for the selected destination
        logger.LogInformation("Step 2: Creating itinerary for {DestinationName}", topDestination.DestinationName);
        context.SetCustomStatus(new {
            step = "CreateItinerary",
            message = $"Creating a detailed itinerary for {topDestination.DestinationName}...",
            progress = 30,
            destination = topDestination.DestinationName
        });
        
        var itineraryPrompt = $@"Create a {travelRequest.DurationInDays}-day itinerary for {topDestination.DestinationName}.
                            Dates: {travelRequest.TravelDates}
                            Budget: {travelRequest.Budget}
                            Requirements: {travelRequest.SpecialRequirements}
                            
                            CRITICAL: Keep ALL descriptions under 50 characters. Include only 2-4 activities per day.
                            Use short names and abbreviated formats.
                            
                            IMPORTANT: Determine the local currency for {topDestination.DestinationName} and use the currency converter 
                            tool to provide costs in both the user's budget currency and the destination's local currency.";
            
        TravelItinerary itinerary;
        try
        {
            logger.LogInformation("Calling itinerary agent with prompt length: {Length}", itineraryPrompt.Length);
            
            var itineraryResponse = await itineraryAgent.RunAsync<TravelItinerary>(
                itineraryPrompt,
                itineraryThread);
            
            if (itineraryResponse == null || itineraryResponse.Result == null)
            {
                throw new InvalidOperationException("Agent returned null response");
            }
            
            itinerary = itineraryResponse.Result;
            logger.LogInformation("Successfully received itinerary for {Destination}", itinerary.DestinationName);
        }
        catch (Exception ex)
        {
            logger.LogError(ex, "Error getting itinerary from agent for {Destination}. Error: {Error}", 
                topDestination.DestinationName, ex.Message);
            throw new InvalidOperationException(
                $"Failed to generate itinerary for {topDestination.DestinationName}. " +
                $"The AI agent returned an invalid or empty response. Please try again.", ex);
        }

        // Validate and correct the cost calculation
        itinerary = ValidateAndFixCostCalculation(itinerary, logger);

        // Step 3: Get local recommendations
        logger.LogInformation("Step 3: Getting local recommendations for {DestinationName}", topDestination.DestinationName);
        context.SetCustomStatus(new {
            step = "GetLocalRecommendations",
            message = $"Finding local hidden gems and recommendations in {topDestination.DestinationName}...",
            progress = 50,
            destination = topDestination.DestinationName
        });
        
        var localPrompt = $@"Provide local recommendations for {topDestination.DestinationName}:
                        Duration: {travelRequest.DurationInDays} days
                        Preferred Cuisine: Any
                        Include Hidden Gems: true
                        Family Friendly: {travelRequest.SpecialRequirements.Contains("family", StringComparison.OrdinalIgnoreCase)}";
            
        var localRecommendationsResponse = await localRecommendationsAgent.RunAsync<LocalRecommendations>(
            localPrompt,
            localThread);
        
        var localRecommendations = localRecommendationsResponse.Result;

        // Combine all results into a comprehensive travel plan
        var travelPlan = new TravelPlan(destinationRecommendations, itinerary, localRecommendations);
        
        // Step 4: Save the travel plan to blob storage
        logger.LogInformation("Step 4: Saving travel plan to blob storage");
        context.SetCustomStatus(new {
            step = "SaveTravelPlan",
            message = "Finalizing your travel plan and preparing documentation...",
            progress = 70,
            destination = topDestination.DestinationName
        });
        var savePlanRequest = new SaveTravelPlanRequest(travelPlan, travelRequest.UserName);
        var documentUrl = await context.CallActivityAsync<string>(
            nameof(TravelPlannerActivities.SaveTravelPlanToBlob),
            savePlanRequest);
        
        // Step 5: Request approval before booking the trip (Human Interaction Pattern)
        logger.LogInformation("Step 5: Requesting approval for travel plan");
        context.SetCustomStatus(new {
            step = "RequestApproval",
            message = "Sending travel plan for your approval...",
            progress = 85,
            destination = topDestination.DestinationName,
            documentUrl = documentUrl
        });
        var approvalRequest = new ApprovalRequest(context.InstanceId, travelPlan, travelRequest.UserName);
        await context.CallActivityAsync(nameof(TravelPlannerActivities.RequestApproval), approvalRequest);
        
        // Step 6: Wait for approval
        logger.LogInformation("Step 6: Waiting for approval from user {UserName}", travelRequest.UserName);
        
        // Define a default response for timeout
        var defaultApprovalResponse = new ApprovalResponse(false, "Timed out waiting for approval");
        
        // Wait for external event with timeout
        ApprovalResponse approvalResponse;
        try
        {
            // Update the waiting for approval status with more structured data including the full dailyPlan and local recommendations
            var waitingStatus = new {
                step = "WaitingForApproval",
                message = "Waiting for your approval of the travel plan...",
                progress = 90,
                destination = topDestination.DestinationName,
                documentUrl = documentUrl,
                travelPlan = new {
                    destination = topDestination.DestinationName,
                    dates = itinerary.TravelDates,
                    cost = itinerary.EstimatedTotalCost,
                    days = itinerary.DailyPlan.Count,
                    dailyPlan = itinerary.DailyPlan, // Include the full dailyPlan in the status update
                    attractions = localRecommendations.Attractions.FirstOrDefault(), // Include attractions
                    restaurants = localRecommendations.Restaurants.FirstOrDefault(), // Include restaurants
                    insiderTips = localRecommendations.InsiderTips // Include insider tips
                }
            };

            // The custom status has a max size of 16KB.
            var waitingStatusSize = Encoding.Unicode.GetByteCount(JsonSerializer.Serialize(waitingStatus));
            
            logger.LogInformation("Waiting status size: {Size} bytes", waitingStatusSize);
            
            context.SetCustomStatus(waitingStatus);
            
            approvalResponse = await context.WaitForExternalEvent<ApprovalResponse>(
                "ApprovalEvent",
                TimeSpan.FromDays(7)); // Wait up to 7 days for a response
        }
        catch (TaskCanceledException)
        {
            // If timeout occurs, use the default response
            logger.LogWarning("Approval request timed out for user {UserName}", travelRequest.UserName);
            approvalResponse = defaultApprovalResponse;
        }
            
        // Check if the trip was approved
        if (approvalResponse.Approved)
        {
            // Step 7: Book the trip if approved
            logger.LogInformation("Step 7: Booking trip to {Destination} for user {UserName}", 
                itinerary.DestinationName, travelRequest.UserName);
                
            context.SetCustomStatus(new {
                step = "BookingTrip",
                message = $"Booking your trip to {topDestination.DestinationName}...",
                progress = 95,
                destination = topDestination.DestinationName,
                documentUrl = documentUrl,
                approved = true
            });
                
            var bookingRequest = new BookingRequest(travelPlan, travelRequest.UserName, approvalResponse.Comments);
            var bookingConfirmation = await context.CallActivityAsync<BookingConfirmation>(
                nameof(TravelPlannerActivities.BookTrip), bookingRequest);
                
            // Return the travel plan with booking confirmation
            logger.LogInformation("Completed travel planning for {UserName} with booking confirmation {BookingId}", 
                travelRequest.UserName, bookingConfirmation.BookingId);
                
            return new TravelPlanResult(
                travelPlan, 
                documentUrl, 
                $"Booking confirmed: {bookingConfirmation.BookingId} - {bookingConfirmation.ConfirmationDetails}");
        }
        else
        {
            // Return the travel plan without booking
            logger.LogInformation("Travel plan for {UserName} was not approved. Comments: {Comments}", 
                travelRequest.UserName, approvalResponse.Comments);
                
            return new TravelPlanResult(
                travelPlan, 
                documentUrl, 
                $"Travel plan was not approved. Comments: {approvalResponse.Comments}");
        }
    }

    private TravelPlan CreateEmptyTravelPlan()
    {
        return new TravelPlan(
            new DestinationRecommendations(new List<DestinationRecommendation>()),
            new TravelItinerary("None", "N/A", new List<ItineraryDay>(), "0", "No itinerary available"),
            new LocalRecommendations(new List<Attraction>(), new List<Restaurant>(), "No recommendations available")
        );
    }

    private TravelItinerary ValidateAndFixCostCalculation(TravelItinerary itinerary, ILogger logger)
    {
        try
        {
            // Extract all activity costs and sum them up
            decimal totalCost = 0;
            string? localCurrency = null;
            string? userCurrency = null;
            decimal exchangeRate = 1.0m;

            foreach (var day in itinerary.DailyPlan)
            {
                foreach (var activity in day.Activities)
                {
                    var cost = activity.EstimatedCost;
                    if (string.IsNullOrEmpty(cost) || 
                        cost.Equals("Free", StringComparison.OrdinalIgnoreCase) || 
                        cost.Equals("Varies", StringComparison.OrdinalIgnoreCase) ||
                        cost.Equals("0", StringComparison.OrdinalIgnoreCase))
                    {
                        continue;
                    }

                    // Parse cost string like "500 JPY (3.26 USD)" or "25 EUR"
                    var match = System.Text.RegularExpressions.Regex.Match(cost, @"(\d+(?:\.\d+)?)\s*([A-Z]{3})");
                    if (match.Success)
                    {
                        if (decimal.TryParse(match.Groups[1].Value, out decimal amount))
                        {
                            totalCost += amount;
                            if (localCurrency == null)
                            {
                                localCurrency = match.Groups[2].Value;
                            }
                        }

                        // Try to extract the converted currency for exchange rate calculation
                        var convertedMatch = System.Text.RegularExpressions.Regex.Match(cost, @"\((\d+(?:\.\d+)?)\s*([A-Z]{3})\)");
                        if (convertedMatch.Success && userCurrency == null)
                        {
                            userCurrency = convertedMatch.Groups[2].Value;
                            if (decimal.TryParse(convertedMatch.Groups[1].Value, out decimal convertedAmount) && amount > 0)
                            {
                                exchangeRate = convertedAmount / amount;
                            }
                        }
                    }
                }
            }

            // Calculate the corrected total cost
            var correctedLocalCost = Math.Round(totalCost, 2);
            var correctedUserCost = Math.Round(totalCost * exchangeRate, 2);

            // Format the corrected cost string
            string correctedCostString;
            if (!string.IsNullOrEmpty(userCurrency) && localCurrency != userCurrency)
            {
                correctedCostString = $"{correctedLocalCost:N0} {localCurrency} ({correctedUserCost:N2} {userCurrency})";
            }
            else
            {
                correctedCostString = $"{correctedLocalCost:N0} {localCurrency ?? "USD"}";
            }

            // Log the correction if there was a discrepancy
            if (itinerary.EstimatedTotalCost != correctedCostString)
            {
                logger.LogWarning(
                    "Cost calculation corrected. Agent calculated: {AgentCost}, Actual sum: {CorrectedCost}",
                    itinerary.EstimatedTotalCost,
                    correctedCostString);
            }

            // Return a new itinerary with the corrected cost
            return new TravelItinerary(
                itinerary.DestinationName,
                itinerary.TravelDates,
                itinerary.DailyPlan,
                correctedCostString,
                itinerary.AdditionalNotes
            );
        }
        catch (Exception ex)
        {
            logger.LogError(ex, "Error validating cost calculation, using agent's original value");
            return itinerary;
        }
    }
}