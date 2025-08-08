import os
import json
import logging
from typing import Dict, Any
from openai import AsyncAzureOpenAI
from agents import Agent, Runner, OpenAIChatCompletionsModel, set_default_openai_client


# Initialize Azure OpenAI client
openai_client = AsyncAzureOpenAI(
    api_key=os.getenv("AZURE_OPENAI_API_KEY"),
    api_version=os.getenv("AZURE_OPENAI_API_VERSION", "2024-02-01"),
    azure_endpoint=os.getenv("AZURE_OPENAI_ENDPOINT"),
    azure_deployment=os.getenv("AZURE_OPENAI_DEPLOYMENT", "gpt-4")
)

# Set the default OpenAI client for the Agents SDK
set_default_openai_client(openai_client)

deployment = os.getenv("AZURE_OPENAI_DEPLOYMENT", "gpt-4")

# Agent prompts 
DESTINATION_AGENT_PROMPT = """You are a travel destination expert. Recommend 3 destinations.
Keep responses brief and focused. For each destination provide:
- Destination name (max 50 chars)
- Brief description (max 150 chars)
- Short reasoning (max 100 chars)  
- Match score (0-100)

Always respond with JSON: {"recommendations": [{"destination_name": "", "description": "", "reasoning": "", "match_score": 0}]}"""

ITINERARY_AGENT_PROMPT = """Create a concise daily itinerary.
Include:
- Maximum 3 activities per day
- Brief activity descriptions (max 80 chars)
- Simple cost estimates (e.g., "$50", "$100-150")

Always respond with JSON: {"destination_name": "", "travel_dates": "", "daily_plan": [{"day": 1, "date": "", "activities": [{"time": "", "activity_name": "", "description": "", "location": "", "estimated_cost": ""}]}], "estimated_total_cost": "", "additional_notes": ""}"""

LOCAL_RECOMMENDATIONS_AGENT_PROMPT = """Provide brief local recommendations.
Include:
- Maximum 3 attractions with short descriptions (max 80 chars each)
- Maximum 3 restaurants with short descriptions (max 80 chars each)
- Brief insider tips (max 150 chars total)

Always respond with JSON: {"attractions": [{"name": "", "category": "", "description": "", "location": "", "visit_duration": "", "estimated_cost": "", "rating": 4.5}], "restaurants": [{"name": "", "cuisine": "", "description": "", "location": "", "price_range": "", "rating": 4.5}], "insider_tips": ""}"""

def clean_json_response(response: str) -> str:
    """Clean and validate JSON response"""
    if not response:
        return "{}"
    
    response = response.strip()
    
    # Remove markdown code blocks if present
    if "```json" in response:
        start = response.find("```json") + 7
        end = response.find("```", start)
        if end > start:
            response = response[start:end].strip()
    elif "```" in response:
        start = response.find("```") + 3
        end = response.rfind("```")
        if end > start:
            response = response[start:end].strip()
    
    # Remove backticks
    response = response.replace("`", "").strip()
    
    try:
        json.loads(response)
        return response
    except json.JSONDecodeError:
        return "{}"


async def get_destination_recommendations(request: Dict[str, Any]) -> Dict[str, Any]:
    """Get destination recommendations using Azure OpenAI Agent SDK"""
    logger = logging.getLogger(__name__)
    logger.info(f"Getting destination recommendations for user {request.get('userName', 'Unknown')}")
    
    try:
        # Create the prompt
        input_prompt = (
            f"Based on the following preferences, recommend 3 travel destinations:\n"
            f"User: {request.get('userName', 'Unknown')}\n"
            f"Preferences: {request.get('preferences', '')}\n"
            f"Duration: {request.get('durationInDays', 7)} days\n"
            f"Budget: {request.get('budget', '')}\n"
            f"Travel Dates: {request.get('travelDates', '')}\n"
            f"Special Requirements: {request.get('specialRequirements', '')}\n\n"
            f"Provide detailed explanations for each recommendation highlighting why it matches the user's preferences."
        )
        
        # Create destination agent
        agent = Agent(
            name="DestinationExpert",
            instructions=DESTINATION_AGENT_PROMPT,
            model=OpenAIChatCompletionsModel(model=deployment, openai_client=openai_client)
        )
        
        # Use Runner.run to get recommendations
        result = await Runner.run(agent, input=input_prompt)
        
        # Clean and parse response
        output = clean_json_response(result.final_output)
        parsed_result = json.loads(output)
        
        # Convert to expected format
        recommendations = []
        for rec in parsed_result.get("recommendations", []):
            recommendations.append({
                "destination_name": rec.get("destination_name", ""),
                "description": rec.get("description", ""),
                "reasoning": rec.get("reasoning", ""),
                "match_score": rec.get("match_score", 0)
            })
        
        logger.info(f"Generated {len(recommendations)} destination recommendations")
        return {"recommendations": recommendations}
        
    except Exception as ex:
        logger.error(f"Error getting destination recommendations: {ex}")
        return {"recommendations": []}


async def create_itinerary(request: Dict[str, Any]) -> Dict[str, Any]:
    """Create an itinerary using Azure OpenAI agent"""
    logger = logging.getLogger(__name__)
    logger.info(f"Creating itinerary for {request['destination_name']}")
    
    try:
        # Create the prompt
        input_prompt = (
            f"Create a detailed daily itinerary for a trip to {request['destination_name']}:\n"
            f"Duration: {request['duration_in_days']} days\n"
            f"Budget: {request['budget']}\n"
            f"Travel Dates: {request['travel_dates']}\n"
            f"Special Requirements: {request['special_requirements']}\n\n"
            f"Include a mix of sightseeing, cultural activities, and relaxation time with realistic costs."
        )
        
        # Create itinerary agent
        agent = Agent(
            name="ItineraryPlanner",
            instructions=ITINERARY_AGENT_PROMPT,
            model=OpenAIChatCompletionsModel(model=deployment, openai_client=openai_client)
        )
        
        # Use Runner.run to get itinerary
        result = await Runner.run(agent, input=input_prompt)
        
        # Clean and parse response
        output = clean_json_response(result.final_output)
        parsed_result = json.loads(output)
        
        # Convert to expected format
        formatted_daily_plan = []
        for day in parsed_result.get("daily_plan", []):
            activities = []
            for activity in day.get("activities", []):
                activities.append({
                    "time": activity.get("time", "Unknown"),
                    "activity_name": activity.get("activity_name", "Unknown"),
                    "description": activity.get("description", ""),
                    "location": activity.get("location", "Unknown"),
                    "estimated_cost": activity.get("estimated_cost", "0")
                })
            
            formatted_daily_plan.append({
                "day": day.get("day", 1),
                "date": day.get("date", "Unknown"),
                "activities": activities
            })
        
        itinerary = {
            "destination_name": parsed_result.get("destination_name", request["destination_name"]),
            "travel_dates": parsed_result.get("travel_dates", request["travel_dates"]),
            "daily_plan": formatted_daily_plan,
            "estimated_total_cost": parsed_result.get("estimated_total_cost", "0"),
            "additional_notes": parsed_result.get("additional_notes", "Generated itinerary")
        }
        
        logger.info(f"Generated itinerary with {len(formatted_daily_plan)} days")
        return itinerary
        
    except Exception as ex:
        logger.error(f"Error creating itinerary: {ex}")
        return {
            "destination_name": request["destination_name"],
            "travel_dates": request["travel_dates"],
            "daily_plan": [],
            "estimated_total_cost": "0",
            "additional_notes": "Error generating itinerary"
        }


async def get_local_recommendations(request: Dict[str, Any]) -> Dict[str, Any]:
    """Get local recommendations using Azure OpenAI agent"""
    logger = logging.getLogger(__name__)
    logger.info(f"Getting local recommendations for {request['destination_name']}")
    
    try:
        # Create the prompt
        input_prompt = (
            f"Provide local recommendations for {request['destination_name']}:\n"
            f"Duration of Stay: {request['duration_in_days']} days\n"
            f"Preferred Cuisine: {request['preferred_cuisine']}\n"
            f"Include Hidden Gems: {request['include_hidden_gems']}\n"
            f"Family Friendly: {request['family_friendly']}\n\n"
            f"Provide authentic local attractions, restaurants, and insider tips."
        )
        
        # Create local recommendations agent
        agent = Agent(
            name="LocalExpert",
            instructions=LOCAL_RECOMMENDATIONS_AGENT_PROMPT,
            model=OpenAIChatCompletionsModel(model=deployment, openai_client=openai_client)
        )
        
        # Use Runner.run to get recommendations
        result = await Runner.run(agent, input=input_prompt)
        
        # Clean and parse response
        output = clean_json_response(result.final_output)
        parsed_result = json.loads(output)
        
        # Convert to expected format
        attractions = []
        for attraction in parsed_result.get("attractions", []):
            attractions.append({
                "name": attraction.get("name", ""),
                "category": attraction.get("category", ""),
                "description": attraction.get("description", ""),
                "location": attraction.get("location", ""),
                "visit_duration": attraction.get("visit_duration", ""),
                "estimated_cost": attraction.get("estimated_cost", ""),
                "rating": attraction.get("rating", 0.0)
            })
        
        restaurants = []
        for restaurant in parsed_result.get("restaurants", []):
            restaurants.append({
                "name": restaurant.get("name", ""),
                "cuisine": restaurant.get("cuisine", ""),
                "description": restaurant.get("description", ""),
                "location": restaurant.get("location", ""),
                "price_range": restaurant.get("price_range", ""),
                "rating": restaurant.get("rating", 0.0)
            })
        
        recommendations = {
            "attractions": attractions,
            "restaurants": restaurants,
            "insider_tips": parsed_result.get("insider_tips", "")
        }
        
        logger.info(f"Generated {len(attractions)} attractions and {len(restaurants)} restaurants")
        return recommendations
        
    except Exception as ex:
        logger.error(f"Error getting local recommendations: {ex}")
        return {
            "attractions": [],
            "restaurants": [],
            "insider_tips": "Error generating local recommendations"
        }
