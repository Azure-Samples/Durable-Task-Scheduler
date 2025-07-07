using System.Text;
using System.Text.Json;
using Microsoft.Azure.Functions.Worker;
using Microsoft.DurableTask;
using Microsoft.Extensions.Logging;
using TravelPlannerFunctions.Models;

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

        // Step 1: Get destination recommendations
        logger.LogInformation("Step 1: Requesting destination recommendations");
        context.SetCustomStatus(new {
            step = "GetDestinationRecommendations",
            message = "Finding the perfect destinations for your travel preferences...",
            progress = 10
        });
        
        var destinationRecommendations = await context.CallActivityAsync<DestinationRecommendations>(
            nameof(TravelPlannerActivities.GetDestinationRecommendations),
            travelRequest);
            
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
        var itineraryRequest = new TravelItineraryRequest(
            topDestination.DestinationName,
            travelRequest.DurationInDays,
            travelRequest.Budget,
            travelRequest.TravelDates,
            travelRequest.SpecialRequirements);
            
        var itinerary = await context.CallActivityAsync<TravelItinerary>(
            nameof(TravelPlannerActivities.CreateItinerary),
            itineraryRequest);

        // Step 3: Get local recommendations
        logger.LogInformation("Step 3: Getting local recommendations for {DestinationName}", topDestination.DestinationName);
        context.SetCustomStatus(new {
            step = "GetLocalRecommendations",
            message = $"Finding local hidden gems and recommendations in {topDestination.DestinationName}...",
            progress = 50,
            destination = topDestination.DestinationName
        });
        var localRecommendationsRequest = new LocalRecommendationsRequest(
            topDestination.DestinationName,
            travelRequest.DurationInDays,
            "Any", // Default value for preferred cuisine
            true,  // Include hidden gems
            travelRequest.SpecialRequirements.Contains("family", StringComparison.OrdinalIgnoreCase)); // Check if family-friendly
            
        var localRecommendations = await context.CallActivityAsync<LocalRecommendations>(
            nameof(TravelPlannerActivities.GetLocalRecommendations),
            localRecommendationsRequest);

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
}