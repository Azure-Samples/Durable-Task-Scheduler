import logging
import json
import asyncio
import azure.functions as func
import azure.durable_functions as df
from ai_services.agent_service import get_destination_recommendations, create_itinerary as create_itinerary_service, get_local_recommendations

# Initialize the Durable Functions app
app = df.DFApp(http_auth_level=func.AuthLevel.ANONYMOUS)

# ================== HTTP ENDPOINTS ==================

@app.route(route="travel-planner", methods=["POST"])
@app.durable_client_input(client_name="client")
async def travel_planner(req: func.HttpRequest, client) -> func.HttpResponse:
    """Start travel planning"""
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
        return func.HttpResponse("Error starting travel planning", status_code=500)

@app.route(route="travel-planner/status/{instance_id}", methods=["GET"])
@app.durable_client_input(client_name="client")
async def travel_planner_status(req: func.HttpRequest, client) -> func.HttpResponse:
    """Get planning status"""
    try:
        instance_id = req.route_params.get("instance_id")
        status = await client.get_status(instance_id)
        
        if not status:
            return func.HttpResponse("Orchestration not found", status_code=404)
        
        return func.HttpResponse(
            json.dumps({
                "id": status.instance_id,
                "runtimeStatus": str(status.runtime_status),
                "output": status.output,
                "customStatus": status.custom_status
            }),
            status_code=200,
            mimetype="application/json"
        )
    except Exception as ex:
        logging.error(f"Error getting status: {ex}")
        return func.HttpResponse("Error getting status", status_code=500)

@app.route(route="travel-planner/approve/{instance_id}", methods=["POST"])
@app.durable_client_input(client_name="client")
async def travel_planner_approve(req: func.HttpRequest, client) -> func.HttpResponse:
    """Approve or reject travel plan"""
    try:
        instance_id = req.route_params.get("instance_id")
        req_body = req.get_json()
        
        await client.raise_event(instance_id, "ApprovalEvent", req_body)
        
        return func.HttpResponse(
            json.dumps({"message": "Approval processed"}),
            status_code=200,
            mimetype="application/json"
        )
    except Exception as ex:
        logging.error(f"Error processing approval: {ex}")
        return func.HttpResponse("Error processing approval", status_code=500)

# ================== ORCHESTRATOR ==================

@app.orchestration_trigger(context_name="context")
def travel_planner_orchestration(context: df.DurableOrchestrationContext):
    """Travel planner orchestration with approval workflow"""
    travel_request = context.get_input()
    
    try:
        # Step 1: Get destination recommendations
        destinations = yield context.call_activity("get_destinations", travel_request)
        
        # Step 2: Create itinerary for top destination
        if destinations and destinations.get("recommendations"):
            top_destination = destinations["recommendations"][0]
            
            # Set custom status to show progress
            context.set_custom_status({
                "step": "CreatingItinerary",
                "destination": top_destination["destination_name"]
            })
            
            itinerary_request = {
                "destination_name": top_destination["destination_name"],
                "duration_in_days": travel_request.get("durationInDays", 3),
                "budget": travel_request.get("budget", "$1000"),
                "travel_dates": travel_request.get("travelDates", "TBD"),
                "special_requirements": travel_request.get("specialRequirements", "")
            }
            
            itinerary = yield context.call_activity("create_itinerary", itinerary_request)
            
            # Step 3: Get local recommendations
            context.set_custom_status({
                "step": "GettingLocalRecommendations",
                "destination": top_destination["destination_name"]
            })
            
            local_request = {
                "destination_name": top_destination["destination_name"],
                "duration_in_days": travel_request.get("durationInDays", 3),
                "preferred_cuisine": "Any",
                "include_hidden_gems": True,
                "family_friendly": True
            }
            
            local_recs = yield context.call_activity("get_local_recs", local_request)
            
            # Step 4: Wait for approval
            context.set_custom_status({
                "step": "WaitingForApproval",
                "destination": top_destination["destination_name"],
                "travelPlan": {
                    "dates": itinerary.get("travel_dates", "TBD"),
                    "cost": itinerary.get("estimated_total_cost", "TBD"),
                    "dailyPlan": itinerary.get("daily_plan", []),
                    "attractions": local_recs.get("attractions", []),
                    "restaurants": local_recs.get("restaurants", []),
                    "insiderTips": local_recs.get("insider_tips", "")
                }
            })
            
            # Wait for approval event
            approval_result = yield context.wait_for_external_event("ApprovalEvent")
            
            # Handle both string and dict approval results
            if isinstance(approval_result, str):
                try:
                    import json
                    approval_result = json.loads(approval_result)
                except:
                    approval_result = {"approved": False, "comments": "Invalid approval format"}
            
            # Format final response based on approval
            if approval_result.get("approved", False):
                # Step 5: Book the trip
                context.set_custom_status({
                    "step": "BookingTrip",
                    "destination": top_destination["destination_name"]
                })
                
                booking_request = {
                    "destination_name": top_destination["destination_name"],
                    "estimated_cost": itinerary.get("estimated_total_cost", "TBD"),
                    "travel_dates": itinerary.get("travel_dates", "TBD"),
                    "user_name": travel_request.get("userName", "Unknown"),
                    "approval_comments": approval_result.get("comments", "")
                }
                
                booking_result = yield context.call_activity("book_trip", booking_request)
                
                context.set_custom_status({
                    "step": "Completed",
                    "destination": top_destination["destination_name"],
                    "booking_id": booking_result.get("booking_id", "N/A")
                })
                
                result = {
                    "Plan": {
                        "DestinationRecommendations": destinations,
                        "Itinerary": itinerary,
                        "LocalRecommendations": local_recs,
                        "Attractions": local_recs.get("attractions", []),
                        "Restaurants": local_recs.get("restaurants", []),
                        "InsiderTips": local_recs.get("insider_tips", "")
                    },
                    "BookingResult": booking_result,
                    "BookingConfirmation": f"Booking confirmed for your trip to {top_destination['destination_name']}! Confirmation ID: {booking_result.get('booking_id', 'N/A')}",
                    "DocumentUrl": f"https://example.com/booking/{context.instance_id}"
                }
            else:
                result = {
                    "Plan": {
                        "DestinationRecommendations": destinations,
                        "Itinerary": itinerary,
                        "LocalRecommendations": local_recs
                    },
                    "BookingConfirmation": f"Travel plan was not approved. Comments: {approval_result.get('comments', 'No comments provided')}"
                }
            
            return result
        else:
            return {"error": "No destinations found"}
            
    except Exception as ex:
        logging.error(f"Orchestration error: {ex}")
        return {"error": str(ex)}

# ================== ACTIVITIES ==================

@app.activity_trigger(input_name="request")
async def get_destinations(request: dict):
    """Get destination recommendations"""
    try:
        return await get_destination_recommendations(request)
    except Exception as ex:
        logging.error(f"Error in get_destinations: {ex}")
        return {"recommendations": []}

@app.activity_trigger(input_name="request")
async def create_itinerary(request: dict):
    """Create itinerary"""
    try:
        return await create_itinerary_service(request)
    except Exception as ex:
        logging.error(f"Error in create_itinerary: {ex}")
        return {"error": str(ex)}

@app.activity_trigger(input_name="request")
async def get_local_recs(request: dict):
    """Get local recommendations"""
    try:
        return await get_local_recommendations(request)
    except Exception as ex:
        logging.error(f"Error in get_local_recs: {ex}")
        return {"attractions": [], "restaurants": [], "insider_tips": ""}

@app.activity_trigger(input_name="request")
async def book_trip(request: dict):
    """Book the trip"""
    try:
        # Simulate booking process
        import time
        import random
        
        destination = request.get("destination_name", "Unknown")
        estimated_cost = request.get("estimated_cost", "TBD")
        
        # Simulate booking time (async sleep instead of time.sleep)
        await asyncio.sleep(2)
        
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
