namespace TravelPlannerFunctions.Models;

public record TravelRequest(
    string UserName,
    string Preferences,
    int DurationInDays,
    string Budget,
    string TravelDates,
    string SpecialRequirements
);

public record DestinationRecommendation(
    string DestinationName,
    string Description,
    string Reasoning,
    double MatchScore
);

public record DestinationRecommendations(
    List<DestinationRecommendation> Recommendations
);

public record TravelItineraryRequest(
    string DestinationName,
    int DurationInDays,
    string Budget,
    string TravelDates,
    string SpecialRequirements
);

public record ItineraryDay(
    int Day,
    string Date,
    List<ItineraryActivity> Activities
);

public record ItineraryActivity(
    string Time,
    string ActivityName,
    string Description,
    string Location,
    string EstimatedCost
);

public record TravelItinerary(
    string DestinationName,
    string TravelDates,
    List<ItineraryDay> DailyPlan,
    string EstimatedTotalCost,
    string AdditionalNotes
);

public record LocalRecommendationsRequest(
    string DestinationName,
    int DurationInDays,
    string PreferredCuisine,
    bool IncludeHiddenGems,
    bool FamilyFriendly
);

public record Attraction(
    string Name,
    string Category,
    string Description,
    string Location,
    string VisitDuration,
    string EstimatedCost,
    double Rating
);

public record Restaurant(
    string Name,
    string Cuisine,
    string Description,
    string Location,
    string PriceRange,
    double Rating
);

public record LocalRecommendations(
    List<Attraction> Attractions,
    List<Restaurant> Restaurants,
    string InsiderTips
);

public record TravelPlan(
    DestinationRecommendations DestinationRecommendations,
    TravelItinerary Itinerary,
    LocalRecommendations LocalRecommendations
);

public record SaveTravelPlanRequest(
    TravelPlan TravelPlan,
    string UserName
);

public record TravelPlanResult(
    TravelPlan Plan,
    string DocumentUrl,
    string? BookingConfirmation = null
);

public record ApprovalRequest(
    string InstanceId,
    TravelPlan TravelPlan, 
    string UserName
);

public record ApprovalResponse(
    bool Approved,
    string Comments
);

public record BookingRequest(
    TravelPlan TravelPlan,
    string UserName,
    string ApproverComments
);

public record BookingConfirmation(
    string BookingId,
    string ConfirmationDetails,
    DateTime BookingDate
);