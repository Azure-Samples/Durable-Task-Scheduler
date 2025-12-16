"""
Pydantic models for Travel Planner agents structured responses.
"""
from typing import List, Optional
from pydantic import BaseModel, Field


# ================== Travel Request Models ==================

class TravelRequest(BaseModel):
    """User's travel planning request."""
    user_name: str = Field(alias="userName", default="")
    preferences: str = ""
    duration_in_days: int = Field(alias="durationInDays", default=3)
    budget: str = ""
    travel_dates: str = Field(alias="travelDates", default="")
    special_requirements: str = Field(alias="specialRequirements", default="")

    class Config:
        populate_by_name = True


# ================== Destination Recommendation Models ==================

class DestinationRecommendation(BaseModel):
    """A single destination recommendation."""
    destination_name: str = Field(alias="DestinationName", default="")
    description: str = Field(alias="Description", default="")
    reasoning: str = Field(alias="Reasoning", default="")
    match_score: int = Field(alias="MatchScore", default=0)

    class Config:
        populate_by_name = True


class DestinationRecommendations(BaseModel):
    """Collection of destination recommendations from the agent."""
    recommendations: List[DestinationRecommendation] = Field(alias="Recommendations", default_factory=list)

    class Config:
        populate_by_name = True


# ================== Itinerary Models ==================

class Activity(BaseModel):
    """A single activity in the itinerary."""
    time: str = Field(alias="Time", default="")
    activity_name: str = Field(alias="ActivityName", default="")
    description: str = Field(alias="Description", default="")
    location: str = Field(alias="Location", default="")
    estimated_cost: str = Field(alias="EstimatedCost", default="")

    class Config:
        populate_by_name = True


class DayPlan(BaseModel):
    """A single day's plan in the itinerary."""
    day: int = Field(alias="Day", default=1)
    date: str = Field(alias="Date", default="")
    activities: List[Activity] = Field(alias="Activities", default_factory=list)

    class Config:
        populate_by_name = True


class Itinerary(BaseModel):
    """Complete travel itinerary from the agent."""
    destination_name: str = Field(alias="DestinationName", default="")
    travel_dates: str = Field(alias="TravelDates", default="")
    daily_plan: List[DayPlan] = Field(alias="DailyPlan", default_factory=list)
    estimated_total_cost: str = Field(alias="EstimatedTotalCost", default="")
    additional_notes: str = Field(alias="AdditionalNotes", default="")

    class Config:
        populate_by_name = True


# ================== Local Recommendations Models ==================

class Attraction(BaseModel):
    """A local attraction recommendation."""
    name: str = Field(alias="Name", default="")
    category: str = Field(alias="Category", default="")
    description: str = Field(alias="Description", default="")
    location: str = Field(alias="Location", default="")
    visit_duration: str = Field(alias="VisitDuration", default="")
    estimated_cost: str = Field(alias="EstimatedCost", default="")
    rating: float = Field(alias="Rating", default=0.0)

    class Config:
        populate_by_name = True


class Restaurant(BaseModel):
    """A restaurant recommendation."""
    name: str = Field(alias="Name", default="")
    cuisine: str = Field(alias="Cuisine", default="")
    description: str = Field(alias="Description", default="")
    location: str = Field(alias="Location", default="")
    price_range: str = Field(alias="PriceRange", default="")
    rating: float = Field(alias="Rating", default=0.0)

    class Config:
        populate_by_name = True


class LocalRecommendations(BaseModel):
    """Local recommendations from the agent."""
    attractions: List[Attraction] = Field(alias="Attractions", default_factory=list)
    restaurants: List[Restaurant] = Field(alias="Restaurants", default_factory=list)
    insider_tips: str = Field(alias="InsiderTips", default="")

    class Config:
        populate_by_name = True


# ================== Booking Models ==================

class BookingResult(BaseModel):
    """Result of a booking operation."""
    booking_id: str = ""
    status: str = ""
    destination: str = ""
    total_cost: str = ""
    confirmation_number: str = ""
    booking_date: str = ""
    message: str = ""
    next_steps: str = ""


# ================== Travel Plan Result Models ==================

class TravelPlan(BaseModel):
    """Complete travel plan combining all agent outputs."""
    destination_recommendations: Optional[DestinationRecommendations] = Field(
        alias="DestinationRecommendations", default=None
    )
    itinerary: Optional[Itinerary] = Field(alias="Itinerary", default=None)
    local_recommendations: Optional[LocalRecommendations] = Field(
        alias="LocalRecommendations", default=None
    )
    attractions: List[Attraction] = Field(alias="Attractions", default_factory=list)
    restaurants: List[Restaurant] = Field(alias="Restaurants", default_factory=list)
    insider_tips: str = Field(alias="InsiderTips", default="")

    class Config:
        populate_by_name = True


class TravelPlanResult(BaseModel):
    """Final result of the travel planning orchestration."""
    plan: Optional[TravelPlan] = Field(alias="Plan", default=None)
    booking_result: Optional[BookingResult] = Field(alias="BookingResult", default=None)
    booking_confirmation: str = Field(alias="BookingConfirmation", default="")
    document_url: str = Field(alias="DocumentUrl", default="")

    class Config:
        populate_by_name = True
