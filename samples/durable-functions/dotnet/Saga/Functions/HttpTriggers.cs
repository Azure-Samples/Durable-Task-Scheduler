using DurableFunctionsSaga.Models;
using Microsoft.Azure.Functions.Worker;
using Microsoft.Azure.Functions.Worker.Http;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.Extensions.Logging;
using System;
using System.Net;
using System.Text.Json;
using System.Threading.Tasks;

namespace DurableFunctionsSaga.Functions
{
    public class HttpTriggers
    {
        private readonly ILogger<HttpTriggers> _logger;

        public HttpTriggers(ILogger<HttpTriggers> logger)
        {
            _logger = logger;
        }

        [Function("StartOrderSaga")]
        public async Task<HttpResponseData> StartOrderSaga(
            [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "orders")] HttpRequestData req,
            [DurableClient] DurableTaskClient client)
        {
            _logger.LogInformation("Received request to start a new order saga");
            
            // Parse the order from the request
            var requestBody = await req.ReadAsStringAsync();
            // Handle potential null request body
            if (string.IsNullOrEmpty(requestBody))
            {
                var badResponse = req.CreateResponse(HttpStatusCode.BadRequest);
                await badResponse.WriteStringAsync("Request body cannot be empty");
                return badResponse;
            }

            var order = JsonSerializer.Deserialize<Order>(requestBody, new JsonSerializerOptions
            {
                PropertyNameCaseInsensitive = true
            });
            
            if (order == null)
            {
                var badResponse = req.CreateResponse(HttpStatusCode.BadRequest);
                await badResponse.WriteStringAsync("Invalid order data");
                return badResponse;
            }
            
            // Generate a unique order ID if not provided
            if (string.IsNullOrEmpty(order.OrderId))
            {
                order.OrderId = Guid.NewGuid().ToString();
            }
            
            // Start the orchestration
            string instanceId = await client.ScheduleNewOrchestrationInstanceAsync("ProcessOrder", order);
            
            _logger.LogInformation("Started orchestration with ID = {InstanceId} for order {OrderId}", instanceId, order.OrderId);
            
            // Return a response with links to check the status
            var response = req.CreateResponse(HttpStatusCode.Accepted);
            response.Headers.Add("Content-Type", "application/json");
            
            var payload = new
            {
                id = instanceId,
                orderId = order.OrderId,
                statusQueryGetUri = $"{req.Url.GetLeftPart(UriPartial.Authority)}/api/orders/{instanceId}",
                terminatePostUri = $"{req.Url.GetLeftPart(UriPartial.Authority)}/api/orders/{instanceId}/terminate",
            };
            
            await response.WriteStringAsync(JsonSerializer.Serialize(payload));
            
            return response;
        }

        [Function("GetOrderStatus")]
        public async Task<HttpResponseData> GetOrderStatus(
            [HttpTrigger(AuthorizationLevel.Anonymous, "get", Route = "orders/{instanceId}")] HttpRequestData req,
            [DurableClient] DurableTaskClient client,
            string instanceId)
        {
            _logger.LogInformation("Getting status for orchestration with ID = {InstanceId}", instanceId);
            
            var instance = await client.GetInstanceAsync(instanceId);
            if (instance == null)
            {
                var notFoundResponse = req.CreateResponse(HttpStatusCode.NotFound);
                await notFoundResponse.WriteStringAsync($"No instance found with ID = {instanceId}");
                return notFoundResponse;
            }
            
            var response = req.CreateResponse(HttpStatusCode.OK);
            response.Headers.Add("Content-Type", "application/json");
            
            var runtimeStatus = instance.RuntimeStatus.ToString();
            // CustomStatus is not available in the current version
            var customStatus = "Not available";
            var output = instance.SerializedOutput;
            
            var statusResponse = new
            {
                id = instanceId,
                runtimeStatus,
                customStatus,
                output,
                createdTime = instance.CreatedAt,
                lastUpdatedTime = instance.LastUpdatedAt
            };
            
            await response.WriteStringAsync(JsonSerializer.Serialize(statusResponse));
            
            return response;
        }

        [Function("TerminateOrderSaga")]
        public async Task<HttpResponseData> TerminateOrderSaga(
            [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "orders/{instanceId}/terminate")] HttpRequestData req,
            [DurableClient] DurableTaskClient client,
            string instanceId)
        {
            _logger.LogInformation("Terminating orchestration with ID = {InstanceId}", instanceId);
            
            await client.TerminateInstanceAsync(instanceId, "Terminated by user");
            
            var response = req.CreateResponse(HttpStatusCode.Accepted);
            await response.WriteStringAsync($"Orchestration with ID = {instanceId} has been terminated.");
            
            return response;
        }
    }
}
