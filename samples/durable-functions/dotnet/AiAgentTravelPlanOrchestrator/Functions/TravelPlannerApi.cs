using System.Net;
using Microsoft.Azure.Functions.Worker;
using Microsoft.Azure.Functions.Worker.Http;
using Microsoft.DurableTask.Client;
using Microsoft.Extensions.Logging;
using TravelPlannerFunctions.Models;

namespace TravelPlannerFunctions.Functions;

public class TravelPlannerApi
{
    private readonly ILogger _logger;

    public TravelPlannerApi(ILoggerFactory loggerFactory)
    {
        _logger = loggerFactory.CreateLogger<TravelPlannerApi>();
    }

    [Function(nameof(StartTravelPlanning))]
    public async Task<HttpResponseData> StartTravelPlanning(
        [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "travel-planner")] HttpRequestData req,
        [DurableClient] DurableTaskClient client)
    {
        _logger.LogInformation("Travel planning request received");

        // Parse the request
        TravelRequest travelRequest;
        try
        {
            travelRequest = await req.ReadFromJsonAsync<TravelRequest>() 
                ?? throw new InvalidOperationException("Invalid request body");
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Failed to parse travel request");
            var errorResponse = req.CreateResponse(HttpStatusCode.BadRequest);
            await errorResponse.WriteStringAsync("Invalid request format");
            return errorResponse;
        }

        // Start the orchestration
        string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
            nameof(TravelPlannerOrchestrator.RunTravelPlannerOrchestration), travelRequest);

        _logger.LogInformation("Started orchestration with ID = {instanceId}", instanceId);

        // Return a response with the status URL
        var response = req.CreateResponse(HttpStatusCode.Accepted);
        
        // Add a Location header that points to the status endpoint
        response.Headers.Add("Location", $"/api/travel-planner/status/{instanceId}");
        
        // Add CORS headers
        response.Headers.Add("Access-Control-Allow-Origin", "*");
        response.Headers.Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS");
        response.Headers.Add("Access-Control-Allow-Headers", "Content-Type");
        response.Headers.Add("Access-Control-Allow-Credentials", "true");
        
        await response.WriteAsJsonAsync(new
        {
            id = instanceId,
            statusQueryUrl = $"/api/travel-planner/status/{instanceId}"
        });

        return response;
    }

    [Function(nameof(GetTravelPlanningStatus))]
    public async Task<HttpResponseData> GetTravelPlanningStatus(
        [HttpTrigger(AuthorizationLevel.Anonymous, "get", Route = "travel-planner/status/{instanceId}")] HttpRequestData req,
        string instanceId,
        [DurableClient] DurableTaskClient client)
    {
        _logger.LogInformation("Getting status for orchestration with ID = {instanceId}", instanceId);

        // Get the orchestration status
        var status = await client.GetInstanceAsync(instanceId, true);
        _logger.LogInformation("Status for instance = {instanceId}", status); 

        if (status == null)
        {
            var notFoundResponse = req.CreateResponse(HttpStatusCode.NotFound);
            await notFoundResponse.WriteStringAsync($"No orchestration found with ID = {instanceId}");
            return notFoundResponse;
        }

        var response = req.CreateResponse(HttpStatusCode.OK);
        
        // Add CORS headers
        response.Headers.Add("Access-Control-Allow-Origin", "*");
        response.Headers.Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS");
        response.Headers.Add("Access-Control-Allow-Headers", "Content-Type");
        response.Headers.Add("Access-Control-Allow-Credentials", "true");
        
        await response.WriteAsJsonAsync(status);
        return response;
    }

    [Function(nameof(HandleApprovalResponse))]
    public async Task<HttpResponseData> HandleApprovalResponse(
        [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "travel-planner/approve/{instanceId}")] HttpRequestData req,
        string instanceId,
        [DurableClient] DurableTaskClient client)
    {
        _logger.LogInformation("Received approval response for orchestration with ID = {instanceId}", instanceId);

        // Parse the approval response
        ApprovalResponse approvalResponse;
        try
        {
            approvalResponse = await req.ReadFromJsonAsync<ApprovalResponse>() 
                ?? throw new InvalidOperationException("Invalid approval response");
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Failed to parse approval response");
            var errorResponse = req.CreateResponse(HttpStatusCode.BadRequest);
            await errorResponse.WriteStringAsync("Invalid approval response format");
            return errorResponse;
        }

        // Send the approval response to the orchestration
        _logger.LogInformation("Sending approval response to orchestration: Approved = {approved}", approvalResponse.Approved);
        await client.RaiseEventAsync(instanceId, "ApprovalEvent", approvalResponse);

        // Return a success response
        var response = req.CreateResponse(HttpStatusCode.OK);
        
        // Add CORS headers
        response.Headers.Add("Access-Control-Allow-Origin", "*");
        response.Headers.Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS");
        response.Headers.Add("Access-Control-Allow-Headers", "Content-Type");
        response.Headers.Add("Access-Control-Allow-Credentials", "true");
        
        await response.WriteAsJsonAsync(new
        {
            instanceId,
            message = "Approval response processed successfully",
            approved = approvalResponse.Approved
        });

        return response;
    }

    [Function(nameof(TravelPlannerOptionsHandler))]
    public HttpResponseData TravelPlannerOptionsHandler(
        [HttpTrigger(AuthorizationLevel.Anonymous, "options", Route = "travel-planner")] HttpRequestData req)
    {
        _logger.LogInformation("CORS preflight request received for travel-planner endpoint");
        
        var response = req.CreateResponse(HttpStatusCode.OK);
        response.Headers.Add("Access-Control-Allow-Origin", "*");
        response.Headers.Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS");
        response.Headers.Add("Access-Control-Allow-Headers", "Content-Type");
        response.Headers.Add("Access-Control-Allow-Credentials", "true");
        
        return response;
    }
    
    [Function(nameof(TravelPlannerStatusOptionsHandler))]
    public HttpResponseData TravelPlannerStatusOptionsHandler(
        [HttpTrigger(AuthorizationLevel.Anonymous, "options", Route = "travel-planner/status/{instanceId}")] HttpRequestData req,
        string instanceId)
    {
        _logger.LogInformation("CORS preflight request received for travel-planner/status endpoint");
        
        var response = req.CreateResponse(HttpStatusCode.OK);
        response.Headers.Add("Access-Control-Allow-Origin", "*");
        response.Headers.Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS");
        response.Headers.Add("Access-Control-Allow-Headers", "Content-Type");
        response.Headers.Add("Access-Control-Allow-Credentials", "true");
        
        return response;
    }
    
    [Function(nameof(TravelPlannerApproveOptionsHandler))]
    public HttpResponseData TravelPlannerApproveOptionsHandler(
        [HttpTrigger(AuthorizationLevel.Anonymous, "options", Route = "travel-planner/approve/{instanceId}")] HttpRequestData req,
        string instanceId)
    {
        _logger.LogInformation("CORS preflight request received for travel-planner/approve endpoint");
        
        var response = req.CreateResponse(HttpStatusCode.OK);
        response.Headers.Add("Access-Control-Allow-Origin", "*");
        response.Headers.Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS");
        response.Headers.Add("Access-Control-Allow-Headers", "Content-Type");
        response.Headers.Add("Access-Control-Allow-Credentials", "true");
        
        return response;
    }

    [Function(nameof(GetTripConfirmationStatus))]
    public async Task<HttpResponseData> GetTripConfirmationStatus(
        [HttpTrigger(AuthorizationLevel.Anonymous, "get", Route = "travel-planner/confirmation/{instanceId}")] HttpRequestData req,
        string instanceId,
        [DurableClient] DurableTaskClient client)
    {
        _logger.LogInformation("Getting confirmation status for orchestration with ID = {instanceId}", instanceId);

        // Get the orchestration status
        var status = await client.GetInstanceAsync(instanceId, true);
        _logger.LogInformation("Confirmation status for instance = {instanceId}", status); 

        if (status == null)
        {
            var notFoundResponse = req.CreateResponse(HttpStatusCode.NotFound);
            await notFoundResponse.WriteStringAsync($"No orchestration found with ID = {instanceId}");
            return notFoundResponse;
        }

        var response = req.CreateResponse(HttpStatusCode.OK);
        
        // Add CORS headers
        response.Headers.Add("Access-Control-Allow-Origin", "*");
        response.Headers.Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS");
        response.Headers.Add("Access-Control-Allow-Headers", "Content-Type");
        response.Headers.Add("Access-Control-Allow-Credentials", "true");
        
        // Check if the booking has been confirmed
        bool isConfirmed = false;
        bool isRejected = false;
        string confirmationMessage = "";
        
        // Check if the orchestration is completed and has output with booking confirmation
        if (status.RuntimeStatus == OrchestrationRuntimeStatus.Completed && status.ReadOutputAs<object>() != null)
        {
            try
            {
                // First try to read as JSON element
                var jsonOutput = status.ReadOutputAs<System.Text.Json.JsonElement>();
                
                // Check for booking confirmation properties
                if (jsonOutput.TryGetProperty("bookingConfirmation", out var bookingConfirmationElement) || 
                    jsonOutput.TryGetProperty("BookingConfirmation", out bookingConfirmationElement))
                {
                    if (bookingConfirmationElement.ValueKind == System.Text.Json.JsonValueKind.String)
                    {
                        string? bookingConfirmation = bookingConfirmationElement.GetString();
                        if (bookingConfirmation != null)
                        {
                            if (bookingConfirmation.Contains("Booking confirmed"))
                            {
                                isConfirmed = true;
                                confirmationMessage = bookingConfirmation;
                            }
                            else if (bookingConfirmation.Contains("not approved"))
                            {
                                isRejected = true;
                                confirmationMessage = bookingConfirmation;
                            }
                        }
                    }
                }
                // If we couldn't find the booking confirmation property, check other properties or the whole object
                else
                {
                    string outputString = jsonOutput.ToString();
                    if (!string.IsNullOrEmpty(outputString))
                    {
                        if (outputString.Contains("Booking confirmed"))
                        {
                            isConfirmed = true;
                            confirmationMessage = outputString;
                        }
                        else if (outputString.Contains("not approved"))
                        {
                            isRejected = true;
                            confirmationMessage = outputString;
                        }
                    }
                }
            }
            catch (Exception ex)
            {
                _logger.LogError(ex, "Error processing orchestration output");
            }
        }
        
        await response.WriteAsJsonAsync(new
        {
            instanceId,
            isConfirmed,
            isRejected,
            confirmationMessage,
            status.RuntimeStatus
        });

        return response;
    }
    
    [Function(nameof(TripConfirmationOptionsHandler))]
    public HttpResponseData TripConfirmationOptionsHandler(
        [HttpTrigger(AuthorizationLevel.Anonymous, "options", Route = "travel-planner/confirmation/{instanceId}")] HttpRequestData req,
        string instanceId)
    {
        _logger.LogInformation("CORS preflight request received for travel-planner/confirmation endpoint");
        
        var response = req.CreateResponse(HttpStatusCode.OK);
        response.Headers.Add("Access-Control-Allow-Origin", "*");
        response.Headers.Add("Access-Control-Allow-Methods", "GET, OPTIONS");
        response.Headers.Add("Access-Control-Allow-Headers", "Content-Type");
        response.Headers.Add("Access-Control-Allow-Credentials", "true");
        
        return response;
    }
}