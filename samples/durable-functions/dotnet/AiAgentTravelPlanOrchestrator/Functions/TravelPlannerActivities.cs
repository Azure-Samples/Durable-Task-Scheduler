using Microsoft.Azure.Functions.Worker;
using Microsoft.Extensions.Logging;
using TravelPlannerFunctions.Models;
using TravelPlannerFunctions.Agents;
using Azure.Storage.Blobs;
using System.Text;

namespace TravelPlannerFunctions.Functions;

public class TravelPlannerActivities
{
    private readonly ILogger _logger;
    private readonly DestinationRecommenderAgent _destinationService;
    private readonly ItineraryPlannerAgent _itineraryService;
    private readonly LocalRecommendationsAgent _localRecommendationsService;
    private readonly BlobServiceClient _blobServiceClient;

    public TravelPlannerActivities(
        ILoggerFactory loggerFactory,
        BlobServiceClient blobServiceClient,
        DestinationRecommenderAgent destinationRecommenderService,
        ItineraryPlannerAgent itineraryPlannerService,
        LocalRecommendationsAgent localRecommendationsService)
    {
        _logger = loggerFactory.CreateLogger<TravelPlannerActivities>();
        _blobServiceClient = blobServiceClient ?? throw new ArgumentNullException(nameof(blobServiceClient));
        _destinationService = destinationRecommenderService ?? throw new ArgumentNullException(nameof(destinationRecommenderService));
        _itineraryService = itineraryPlannerService ?? throw new ArgumentNullException(nameof(itineraryPlannerService));
        _localRecommendationsService = localRecommendationsService ?? throw new ArgumentNullException(nameof(localRecommendationsService));
    }

    [Function(nameof(GetDestinationRecommendations))]
    public async Task<DestinationRecommendations> GetDestinationRecommendations(
        [ActivityTrigger] TravelRequest request)
    {
        _logger.LogInformation("Getting destination recommendations for user {UserName}", request.UserName);
        try
        {
            var recommendations = await _destinationService.GetDestinationRecommendationsAsync(request);
            _logger.LogInformation("Generated {Count} destination recommendations", recommendations.Recommendations.Count);
            return recommendations;
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error getting destination recommendations");
            return new DestinationRecommendations(new List<DestinationRecommendation>());
        }
    }

    [Function(nameof(CreateItinerary))]
    public async Task<TravelItinerary> CreateItinerary(
        [ActivityTrigger] TravelItineraryRequest request)
    {
        _logger.LogInformation("Creating itinerary for {DestinationName}", request.DestinationName);
        try
        {
            var itinerary = await _itineraryService.CreateItineraryAsync(request);
            _logger.LogInformation("Generated itinerary with {Count} days", itinerary.DailyPlan.Count);
            return itinerary;
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error creating itinerary");
            return new TravelItinerary(
                request.DestinationName,
                request.TravelDates,
                new List<ItineraryDay>(),
                "0",
                "Error generating itinerary"
            );
        }
    }

    [Function(nameof(GetLocalRecommendations))]
    public async Task<LocalRecommendations> GetLocalRecommendations(
        [ActivityTrigger] LocalRecommendationsRequest request)
    {
        _logger.LogInformation("Getting local recommendations for {DestinationName}", request.DestinationName);
        try
        {
            var recommendations = await _localRecommendationsService.GetLocalRecommendationsAsync(request);
            _logger.LogInformation("Generated {AttractionCount} attractions and {RestaurantCount} restaurants", 
                recommendations.Attractions.Count, recommendations.Restaurants.Count);
            return recommendations;
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error getting local recommendations");
            return new LocalRecommendations(
                new List<Attraction>(),
                new List<Restaurant>(),
                "Error generating local recommendations"
            );
        }
    }

    [Function(nameof(SaveTravelPlanToBlob))]
    public async Task<string> SaveTravelPlanToBlob(
        [ActivityTrigger] SaveTravelPlanRequest request)
    {
        _logger.LogInformation("Saving travel plan for {UserName} to blob storage", request.UserName);
        
        try
        {
            // Create a unique filename for this travel plan
            string fileName = $"travel-plan-{request.UserName}-{DateTime.UtcNow:yyyy-MM-dd-HH-mm-ss}.txt";
            
            // Format the travel plan as text
            var content = FormatTravelPlanAsText(request.TravelPlan, request.UserName);
            
            // Get a container client using the pre-initialized BlobServiceClient
            var containerClient = _blobServiceClient.GetBlobContainerClient("travel-plans");
            await containerClient.CreateIfNotExistsAsync();
            
            // Upload the travel plan text to blob storage
            var blobClient = containerClient.GetBlobClient(fileName);
            using var stream = new MemoryStream(Encoding.UTF8.GetBytes(content));
            await blobClient.UploadAsync(stream, overwrite: true);
            
            _logger.LogInformation("Successfully saved travel plan to {BlobUrl}", blobClient.Uri);
            
            // Return the URL of the uploaded file
            return blobClient.Uri.ToString();
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error saving travel plan to blob storage");
            return string.Empty;
        }
    }

    [Function(nameof(RequestApproval))]
    public ApprovalRequest RequestApproval(
        [ActivityTrigger] ApprovalRequest request)
    {
        _logger.LogInformation("Requesting approval for travel plan for user {UserName}, instance {InstanceId}", 
            request.UserName, request.InstanceId);
            
        // In a real app, you would send an email, SMS, or other notification
        // to the user and store the approval request in a database.
        // For demo purposes, we'll just log the request and return it.
        
        _logger.LogInformation("Approval URL: https://your-approval-app/approve?id={InstanceId}", request.InstanceId);
        
        return request;
    }

    [Function(nameof(BookTrip))]
    public async Task<BookingConfirmation> BookTrip(
        [ActivityTrigger] BookingRequest request)
    {
        _logger.LogInformation("Booking trip to {Destination} for user {UserName}", 
            request.TravelPlan.Itinerary.DestinationName, request.UserName);
            
        try
        {
            // In a real app, this would integrate with a booking system or API
            // For demo purposes, we'll simulate an async booking operation
            await Task.Delay(100); // Simulate an API call to a booking service
            
            // Generate a booking ID
            string bookingId = $"BK-{Guid.NewGuid().ToString().Substring(0, 8)}";
            
            var confirmation = new BookingConfirmation(
                bookingId,
                $"Trip to {request.TravelPlan.Itinerary.DestinationName} booked successfully for {request.UserName}. " +
                $"Travel dates: {request.TravelPlan.Itinerary.TravelDates}. " +
                (string.IsNullOrEmpty(request.ApproverComments) ? "" : $"Notes: {request.ApproverComments}"),
                DateTime.UtcNow
            );
            
            _logger.LogInformation("Trip booked successfully with booking ID {BookingId}", bookingId);
            return confirmation;
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error booking trip");
            throw; // Rethrow to let the orchestrator handle the error
        }
    }
    
    private string FormatTravelPlanAsText(TravelPlan travelPlan, string userName)
    {
        var sb = new StringBuilder();
        
        sb.AppendLine($"TRAVEL PLAN FOR {userName.ToUpper()}");
        sb.AppendLine($"Generated on {DateTime.UtcNow:yyyy-MM-dd HH:mm:ss} UTC");
        sb.AppendLine(new string('-', 80));
        sb.AppendLine();
        
        // Add destination information
        var topDestination = travelPlan.DestinationRecommendations.Recommendations
            .OrderByDescending(r => r.MatchScore)
            .FirstOrDefault();
            
        if (topDestination != null)
        {
            sb.AppendLine("DESTINATION INFORMATION");
            sb.AppendLine("----------------------");
            sb.AppendLine($"Destination: {topDestination.DestinationName}");
            sb.AppendLine($"Match Score: {topDestination.MatchScore}");
            sb.AppendLine($"Description: {topDestination.Description}");
            sb.AppendLine();
        }
        
        // Add itinerary
        if (travelPlan.Itinerary.DailyPlan.Count > 0)
        {
            sb.AppendLine("ITINERARY");
            sb.AppendLine("---------");
            sb.AppendLine($"Destination: {travelPlan.Itinerary.DestinationName}");
            sb.AppendLine($"Travel Dates: {travelPlan.Itinerary.TravelDates}");
            sb.AppendLine($"Estimated Cost: {travelPlan.Itinerary.EstimatedTotalCost}");
            sb.AppendLine();
            
            foreach (var day in travelPlan.Itinerary.DailyPlan)
            {
                sb.AppendLine($"DAY {day.Day}: {day.Date}");
                
                // Format the activities for this day
                foreach (var activity in day.Activities)
                {
                    sb.AppendLine($"  {activity.Time}: {activity.ActivityName}");
                    sb.AppendLine($"      {activity.Description}");
                    sb.AppendLine($"      Location: {activity.Location}");
                    sb.AppendLine($"      Est. Cost: {activity.EstimatedCost}");
                    sb.AppendLine();
                }
            }
        }
        
        // Add local recommendations
        sb.AppendLine("LOCAL RECOMMENDATIONS");
        sb.AppendLine("--------------------");
        
        // Add attractions
        sb.AppendLine("Top Attractions:");
        if (travelPlan.LocalRecommendations.Attractions.Count > 0)
        {
            foreach (var attraction in travelPlan.LocalRecommendations.Attractions)
            {
                sb.AppendLine($"- {attraction.Name}: {attraction.Description}");
            }
        }
        else
        {
            sb.AppendLine("No attractions found.");
        }
        sb.AppendLine();
        
        // Add restaurants
        sb.AppendLine("Recommended Restaurants:");
        if (travelPlan.LocalRecommendations.Restaurants.Count > 0)
        {
            foreach (var restaurant in travelPlan.LocalRecommendations.Restaurants)
            {
                sb.AppendLine($"- {restaurant.Name}: {restaurant.Cuisine} cuisine, {restaurant.PriceRange}");
            }
        }
        else
        {
            sb.AppendLine("No restaurants found.");
        }
        sb.AppendLine();
        
        // Add additional notes
        if (!string.IsNullOrEmpty(travelPlan.LocalRecommendations.InsiderTips))
        {
            sb.AppendLine("Insider Tips:");
            sb.AppendLine(travelPlan.LocalRecommendations.InsiderTips);
        }
        
        return sb.ToString();
    }
}