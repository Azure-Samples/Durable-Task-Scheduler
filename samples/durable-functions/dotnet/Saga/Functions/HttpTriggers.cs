using DurableFunctionsSaga.Models;
using Microsoft.Azure.Functions.Worker;
using Microsoft.Azure.Functions.Worker.Http;
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

        [Function("StartOrderProcessing")]
        public async Task<HttpResponseData> StartOrderProcessing(
            [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "orders")] HttpRequestData req,
            [DurableClient] DurableTaskClient client)
        {
            _logger.LogInformation("Processing order request");

            // Parse the order from the request body
            var requestBody = await req.ReadAsStringAsync();
            
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

            // Generate a new order ID if not provided
            if (string.IsNullOrEmpty(order.OrderId))
            {
                order.OrderId = Guid.NewGuid().ToString();
            }

            // Start the order processing orchestration
            var instanceId = await client.ScheduleNewOrchestrationInstanceAsync("ProcessOrder", order);
            
            _logger.LogInformation("Started order processing with instance ID: {InstanceId}", instanceId);

            // Return the status response with the instance ID
            var response = req.CreateResponse(HttpStatusCode.Accepted);
            await response.WriteAsJsonAsync(new
            {
                id = instanceId,
                statusQueryGetUri = $"/api/orders/{instanceId}"
            });
            
            return response;
        }

        [Function("GetOrderStatus")]
        public async Task<HttpResponseData> GetOrderStatus(
            [HttpTrigger(AuthorizationLevel.Anonymous, "get", Route = "orders/{instanceId}")] HttpRequestData req,
            string instanceId,
            [DurableClient] DurableTaskClient client)
        {
            var instance = await client.GetInstanceAsync(instanceId);
            if (instance == null)
            {
                var notFoundResponse = req.CreateResponse(HttpStatusCode.NotFound);
                await notFoundResponse.WriteStringAsync($"No order processing instance found with ID: {instanceId}");
                return notFoundResponse;
            }

            var response = req.CreateResponse(HttpStatusCode.OK);
            await response.WriteAsJsonAsync(new
            {
                id = instance.InstanceId,
                status = instance.RuntimeStatus.ToString(),
                createdTime = instance.CreatedAt,
                lastUpdatedTime = instance.LastUpdatedAt,
                output = instance.SerializedOutput
            });
            
            return response;
        }

        [Function("TerminateOrder")]
        public async Task<HttpResponseData> TerminateOrder(
            [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "orders/{instanceId}/terminate")] HttpRequestData req,
            string instanceId,
            [DurableClient] DurableTaskClient client)
        {
            await client.TerminateInstanceAsync(instanceId, "Terminated by user");

            var response = req.CreateResponse(HttpStatusCode.Accepted);
            await response.WriteStringAsync($"Order processing {instanceId} has been terminated");
            
            return response;
        }
    }
}
