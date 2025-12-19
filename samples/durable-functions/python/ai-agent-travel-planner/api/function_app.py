"""
AI Travel Planner with Durable Agents using Microsoft Agent Framework.

This application demonstrates how to build durable AI agents that coordinate 
to create comprehensive travel plans. It uses the Durable Task extension for 
Microsoft Agent Framework to provide:
- Automatic session management and state persistence
- Deterministic multi-agent orchestrations
- Human-in-the-loop approval workflows
- Serverless hosting on Azure Functions
"""
import os
import logging
import random
from datetime import timedelta
from typing import cast

import azure.durable_functions as df
from azure.identity import DefaultAzureCredential
from agent_framework.azure import AzureOpenAIChatClient, AgentFunctionApp

from models import (
    TravelRequest,
    DestinationRecommendations,
    Itinerary,
    LocalRecommendations,
    BookingResult,
    TravelPlan,
    TravelPlanResult,
)
from tools import convert_currency, get_exchange_rate


# ================== Configuration ==================

endpoint = os.getenv("AZURE_OPENAI_ENDPOINT")
deployment_name = os.getenv("AZURE_OPENAI_DEPLOYMENT_NAME", "gpt-4o-mini")

if not endpoint:
    raise ValueError("AZURE_OPENAI_ENDPOINT environment variable is not set. "
                     "Please copy local.settings.json.template to local.settings.json "
                     "and configure your Azure OpenAI endpoint.")

# Create the Azure OpenAI chat client
chat_client = AzureOpenAIChatClient(
    endpoint=endpoint,
    deployment_name=deployment_name,
    credential=DefaultAzureCredential()
)

# ================== Agent Definitions ==================

# Destination Recommender Agent - recommends travel destinations based on preferences
destination_recommender_agent = chat_client.create_agent(
    name="DestinationRecommenderAgent",
    instructions="""You are a travel destination expert who recommends destinations based on user preferences.
Based on the user's preferences, budget, duration, travel dates, and special requirements, recommend 3 travel destinations.
Provide a detailed explanation for each recommendation highlighting why it matches the user's preferences.

Return your response as a JSON object with this structure (use PascalCase for property names):
{
    "Recommendations": [
        {
            "DestinationName": "string",
            "Description": "string",
            "Reasoning": "string",
            "MatchScore": number (0-100)
        }
    ]
}"""
)

# Itinerary Planner Agent - creates detailed day-by-day itineraries
itinerary_planner_agent = chat_client.create_agent(
    name="ItineraryPlannerAgent",
    instructions="""You are a travel itinerary planner. Create concise day-by-day travel plans with key activities and timing.

IMPORTANT: Keep responses compact:
- Descriptions MUST be under 50 characters each
- Include 2-4 activities per day maximum
- Use abbreviated formats for times (9AM not 9:00 AM)
- Keep location names short

CURRENCY HANDLING - FOLLOW THESE RULES EXACTLY:

1. First, identify the user's budget currency (from the budget string, e.g., "$3000" = USD)
2. Identify the destination country's local currency (e.g., Japan=JPY, UK=GBP, Spain=EUR)

3. IF SAME CURRENCY (e.g., user has USD budget and destination uses USD):
   - DO NOT call any currency tools
   - Show all costs in USD only (e.g., "25 USD")
   - No conversion needed

4. IF DIFFERENT CURRENCIES (e.g., user has USD budget but destination uses EUR):
   - Call get_exchange_rate EXACTLY ONCE at the start to get the rate
   - Use simple multiplication for all conversions (do NOT call convert_currency for each activity)
   - Show costs as: local currency first, then user currency in parentheses
   - Example: "25 EUR (27 USD)" where 27 = 25 * exchange_rate

COST CALCULATION:
- Add up all numeric activity costs (ignore "Free" and "Varies")
- EstimatedTotalCost = sum of activity costs
- If currencies differ, show both: "162 EUR (177 USD)"

Return your response as a JSON object with this structure:
{
    "DestinationName": "string",
    "TravelDates": "string",
    "DailyPlan": [
        {
            "Day": number,
            "Date": "string",
            "Activities": [
                {
                    "Time": "string",
                    "ActivityName": "string",
                    "Description": "string",
                    "Location": "string",
                    "EstimatedCost": "string"
                }
            ]
        }
    ],
    "EstimatedTotalCost": "string",
    "AdditionalNotes": "string"
}""",
    tools=[get_exchange_rate, convert_currency]
)


# Local Recommendations Agent - provides restaurant and attraction recommendations
local_recommendations_agent = chat_client.create_agent(
    name="LocalRecommendationsAgent",
    instructions="""You are a local expert who provides recommendations for restaurants and attractions.
Provide specific recommendations with practical details like operating hours, pricing, and tips.

Return your response as a JSON object with this structure:
{
    "Attractions": [
        {
            "Name": "string",
            "Category": "string",
            "Description": "string",
            "Location": "string",
            "VisitDuration": "string",
            "EstimatedCost": "string",
            "Rating": number
        }
    ],
    "Restaurants": [
        {
            "Name": "string",
            "Cuisine": "string",
            "Description": "string",
            "Location": "string",
            "PriceRange": "string",
            "Rating": number
        }
    ],
    "InsiderTips": "string"
}"""
)

# ================== Configure Function App with Durable Agents ==================

app = AgentFunctionApp(agents=[
    destination_recommender_agent,
    itinerary_planner_agent,
    local_recommendations_agent
])


# ================== Travel Planner Orchestration ==================

@app.orchestration_trigger(context_name="context")
def travel_planner_orchestration(context: df.DurableOrchestrationContext):
    """
    Travel planner orchestration with multi-agent coordination and approval workflow.
    
    This orchestration:
    1. Gets destination recommendations from the Destination Recommender Agent
    2. Creates an itinerary using the Itinerary Planner Agent
    3. Gets local recommendations from the Local Recommendations Agent
    4. Waits for human approval
    5. Books the trip if approved
    """
    travel_request_data = context.get_input()
    travel_request = TravelRequest(**travel_request_data) if isinstance(travel_request_data, dict) else travel_request_data
    
    try:
        # Set initial status
        context.set_custom_status({
            "step": "GettingDestinations"
        })
        
        # Step 1: Get destination recommendations
        destination_agent = app.get_agent(context, "DestinationRecommenderAgent")
        destination_thread = destination_agent.get_new_thread()
        
        destination_prompt = f"""Based on the following preferences, recommend 3 travel destinations:
User: {travel_request.user_name}
Preferences: {travel_request.preferences}
Duration: {travel_request.duration_in_days} days
Budget: {travel_request.budget}
Travel Dates: {travel_request.travel_dates}
Special Requirements: {travel_request.special_requirements}

Provide detailed explanations for each recommendation highlighting why it matches the user's preferences."""

        destinations_result = yield destination_agent.run(
            messages=destination_prompt,
            thread=destination_thread,
            response_format=DestinationRecommendations
        )
        
        destinations = cast(
            DestinationRecommendations,
            destinations_result.value
        )
        
        if not destinations or not destinations.recommendations:
            return {"error": "No destinations found"}
        
        # Get top destination
        top_destination = destinations.recommendations[0]
        
        # Update status
        context.set_custom_status({
            "step": "CreatingItinerary",
            "destination": top_destination.destination_name
        })
        
        # Step 2: Create itinerary for top destination
        itinerary_agent = app.get_agent(context, "ItineraryPlannerAgent")
        itinerary_thread = itinerary_agent.get_new_thread()
        
        itinerary_prompt = f"""Create a detailed daily itinerary for a trip to {top_destination.destination_name}:
Duration: {travel_request.duration_in_days} days
Budget: {travel_request.budget}
Travel Dates: {travel_request.travel_dates}
Special Requirements: {travel_request.special_requirements}

Include a mix of sightseeing, cultural activities, and relaxation time with realistic costs."""

        itinerary_result = yield itinerary_agent.run(
            messages=itinerary_prompt,
            thread=itinerary_thread,
            response_format=Itinerary
        )
        
        itinerary = cast(Itinerary, itinerary_result.value)
        
        # Update status
        context.set_custom_status({
            "step": "GettingLocalRecommendations",
            "destination": top_destination.destination_name
        })
        
        # Step 3: Get local recommendations
        local_agent = app.get_agent(context, "LocalRecommendationsAgent")
        local_thread = local_agent.get_new_thread()
        
        local_prompt = f"""Provide local recommendations for {top_destination.destination_name}:
Duration of Stay: {travel_request.duration_in_days} days
Include: Hidden gems, family-friendly options, authentic local experiences

Provide authentic local attractions, restaurants, and insider tips."""

        local_result = yield local_agent.run(
            messages=local_prompt,
            thread=local_thread,
            response_format=LocalRecommendations
        )
        
        local_recs = cast(LocalRecommendations, local_result.value)
        
        logging.info(f"Local recommendations received: {local_recs is not None}")
        
        # Update status to waiting for approval
        context.set_custom_status({
            "step": "WaitingForApproval",
            "destination": top_destination.destination_name,
            "travelPlan": {
                "dates": itinerary.travel_dates if itinerary else "TBD",
                "cost": itinerary.estimated_total_cost if itinerary else "TBD",
                "dailyPlan": [day.model_dump(by_alias=True) for day in itinerary.daily_plan] if itinerary else [],
                "attractions": [a.model_dump(by_alias=True) for a in local_recs.attractions] if local_recs else [],
                "restaurants": [r.model_dump(by_alias=True) for r in local_recs.restaurants] if local_recs else [],
                "insiderTips": local_recs.insider_tips if local_recs else ""
            }
        })
        
        logging.info("Set custom status to WaitingForApproval, now waiting for external event...")
        
        # Step 4: Wait for approval event with timeout
        approval_task = context.wait_for_external_event("ApprovalEvent")
        timeout_task = context.create_timer(context.current_utc_datetime + timedelta(hours=24))
        
        logging.info("Created approval task and timeout task, yielding task_any...")
        
        winner = yield context.task_any([approval_task, timeout_task])
        
        logging.info(f"task_any returned, winner is approval_task: {winner == approval_task}")
        
        if winner == approval_task:
            timeout_task.cancel()
            approval_result = approval_task.result
            
            # Handle both string and dict approval results
            if isinstance(approval_result, str):
                import json
                try:
                    approval_result = json.loads(approval_result)
                except:
                    approval_result = {"approved": False, "comments": "Invalid approval format"}
            
            if approval_result.get("approved", False):
                # Step 5: Book the trip
                context.set_custom_status({
                    "step": "BookingTrip",
                    "destination": top_destination.destination_name
                })
                
                booking_request = {
                    "destination_name": top_destination.destination_name,
                    "estimated_cost": itinerary.estimated_total_cost if itinerary else "TBD",
                    "travel_dates": itinerary.travel_dates if itinerary else "TBD",
                    "user_name": travel_request.user_name,
                    "approval_comments": approval_result.get("comments", "")
                }
                
                booking_result = yield context.call_activity("book_trip", booking_request)
                
                context.set_custom_status({
                    "step": "Completed",
                    "destination": top_destination.destination_name,
                    "booking_id": booking_result.get("booking_id", "N/A")
                })
                
                # Build final result
                result = TravelPlanResult(
                    plan=TravelPlan(
                        destination_recommendations=destinations,
                        itinerary=itinerary,
                        local_recommendations=local_recs,
                        attractions=local_recs.attractions if local_recs else [],
                        restaurants=local_recs.restaurants if local_recs else [],
                        insider_tips=local_recs.insider_tips if local_recs else ""
                    ),
                    booking_result=BookingResult(**booking_result),
                    booking_confirmation=f"Booking confirmed for your trip to {top_destination.destination_name}! Confirmation ID: {booking_result.get('booking_id', 'N/A')}",
                    document_url=f"https://example.com/booking/{context.instance_id}"
                )
                
                return result.model_dump(by_alias=True)
            else:
                # Not approved
                result = TravelPlanResult(
                    plan=TravelPlan(
                        destination_recommendations=destinations,
                        itinerary=itinerary,
                        local_recommendations=local_recs
                    ),
                    booking_confirmation=f"Travel plan was not approved. Comments: {approval_result.get('comments', 'No comments provided')}"
                )
                return result.model_dump(by_alias=True)
        else:
            # Timeout - escalate for review
            logging.info("Timeout task won - travel plan timed out")
            result = TravelPlanResult(
                plan=TravelPlan(
                    destination_recommendations=destinations,
                    itinerary=itinerary,
                    local_recommendations=local_recs
                ),
                booking_confirmation="Travel plan timed out waiting for approval."
            )
            return result.model_dump(by_alias=True)
            
    except Exception as ex:
        import traceback
        logging.error(f"Orchestration error: {ex}")
        logging.error(f"Traceback: {traceback.format_exc()}")
        return {"error": str(ex)}


# ================== Activity Functions ==================

@app.activity_trigger(input_name="request")
def book_trip(request: dict) -> dict:
    """Book the trip - simulates a booking process."""
    try:
        destination = request.get("destination_name", "Unknown")
        estimated_cost = request.get("estimated_cost", "TBD")
        
        # Generate booking confirmation
        booking_id = f"TRV-{random.randint(100000, 999999)}"
        
        return {
            "booking_id": booking_id,
            "status": "confirmed",
            "destination": destination,
            "total_cost": estimated_cost,
            "confirmation_number": booking_id,
            "booking_date": "2025-08-07",
            "message": f"Trip to {destination} successfully booked!",
            "next_steps": "You will receive confirmation emails shortly with detailed itinerary and vouchers."
        }
    except Exception as ex:
        logging.error(f"Error in book_trip: {ex}")
        return {"status": "failed", "error": str(ex)}


# ================== Custom HTTP Endpoints ==================

# The AgentFunctionApp automatically creates these HTTP endpoints for each agent:
# - POST /api/agents/{agentName}/run - Run a single agent interaction
# - POST /api/agents/{agentName}/threads - Create a new thread and run
# - POST /api/agents/{agentName}/threads/{threadId} - Continue an existing thread
#
# For the orchestration-based workflow, we add custom endpoints below:

import azure.functions as func
import json


@app.function_name(name="StartTravelPlanning")
@app.route(route="travel-planner", methods=["POST"])
@app.durable_client_input(client_name="client")
async def start_travel_planning(req: func.HttpRequest, client) -> func.HttpResponse:
    """Start travel planning orchestration."""
    try:
        req_body = req.get_json()
        instance_id = await client.start_new("travel_planner_orchestration", client_input=req_body)
        
        return func.HttpResponse(
            json.dumps({"id": instance_id}),
            status_code=202,
            mimetype="application/json"
        )
    except Exception as ex:
        logging.error(f"Error starting travel planning: {ex}")
        return func.HttpResponse(
            json.dumps({"error": str(ex)}),
            status_code=500,
            mimetype="application/json"
        )


@app.function_name(name="GetTravelPlanningStatus")
@app.route(route="travel-planner/status/{instance_id}", methods=["GET"])
@app.durable_client_input(client_name="client")
async def get_travel_planning_status(req: func.HttpRequest, client) -> func.HttpResponse:
    """Get planning status."""
    try:
        instance_id = req.route_params.get("instance_id")
        status = await client.get_status(instance_id)
        
        if not status:
            return func.HttpResponse(
                json.dumps({"error": "Orchestration not found"}),
                status_code=404,
                mimetype="application/json"
            )
        
        # Log status for debugging
        logging.info(f"Orchestration {instance_id} status: {status.runtime_status.name}, custom_status: {status.custom_status}")
        
        return func.HttpResponse(
            json.dumps({
                "id": status.instance_id,
                "runtimeStatus": status.runtime_status.name,
                "output": status.output,
                "customStatus": status.custom_status
            }),
            status_code=200,
            mimetype="application/json"
        )
    except Exception as ex:
        logging.error(f"Error getting status: {ex}")
        return func.HttpResponse(
            json.dumps({"error": str(ex)}),
            status_code=500,
            mimetype="application/json"
        )


@app.function_name(name="ApproveTravelPlan")
@app.route(route="travel-planner/approve/{instance_id}", methods=["POST"])
@app.durable_client_input(client_name="client")
async def approve_travel_plan(req: func.HttpRequest, client) -> func.HttpResponse:
    """Approve or reject travel plan."""
    try:
        instance_id = req.route_params.get("instance_id")
        req_body = req.get_json()
        
        # Check if the orchestration is still running
        status = await client.get_status(instance_id)
        if status is None:
            return func.HttpResponse(
                json.dumps({"error": "Travel plan not found"}),
                status_code=404,
                mimetype="application/json"
            )
        
        # Allow approval for Running, Pending, or Suspended orchestrations
        active_statuses = ["Running", "Pending", "Suspended"]
        if status.runtime_status.name not in active_statuses:
            return func.HttpResponse(
                json.dumps({
                    "error": f"Travel plan is no longer active (status: {status.runtime_status.name})",
                    "status": status.runtime_status.name
                }),
                status_code=400,
                mimetype="application/json"
            )
        
        await client.raise_event(instance_id, "ApprovalEvent", req_body)
        
        return func.HttpResponse(
            json.dumps({"message": "Approval processed"}),
            status_code=200,
            mimetype="application/json"
        )
    except Exception as ex:
        logging.error(f"Error processing approval: {ex}")
        return func.HttpResponse(
            json.dumps({"error": str(ex)}),
            status_code=500,
            mimetype="application/json"
        )
