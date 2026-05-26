// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

using System.Net;
using Microsoft.Azure.Functions.Worker;
using Microsoft.Azure.Functions.Worker.Http;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.Extensions.Logging;

namespace WorkItemFiltering.AppB;

// =============================================================================
// App B — registers an entirely DIFFERENT set of functions from App A.
// Both apps share the same DTS task hub ("default"). Work item filtering ensures
// each app only receives work items for the functions it has registered.
//
// App A owns:  GreetingOrchestration, FanOutOrchestration, ParentOrchestration,
//              CounterOrchestration, SayHello activity, CounterEntity
// App B owns:  OrdersOrchestration, ShipOrder activity
//
// Either app's client endpoint can SCHEDULE any orchestration name. The
// scheduler routes the work item to the app whose filter matches.
// =============================================================================

public static class OrdersOrchestration
{
    [Function(nameof(OrdersOrchestration))]
    public static async Task<string> Run([OrchestrationTrigger] TaskOrchestrationContext ctx)
    {
        var logger = ctx.CreateReplaySafeLogger(nameof(OrdersOrchestration));
        logger.LogInformation("OrdersOrchestration started on App B");

        string orderId = ctx.GetInput<string>() ?? $"order-{ctx.NewGuid():N}";
        string shipResult = await ctx.CallActivityAsync<string>(nameof(ShipOrder), orderId);
        return shipResult;
    }

    [Function(nameof(OrdersOrchestration) + "_Start")]
    public static async Task<HttpResponseData> Start(
        [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "orchestrators/orders")] HttpRequestData req,
        [DurableClient] DurableTaskClient client)
    {
        string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
            nameof(OrdersOrchestration), input: "order-42");
        return client.CreateCheckStatusResponse(req, instanceId);
    }
}

public static class ShipOrder
{
    [Function(nameof(ShipOrder))]
    public static string Run([ActivityTrigger] string orderId, FunctionContext ctx)
    {
        ctx.GetLogger(nameof(ShipOrder)).LogInformation("App B shipping {OrderId}", orderId);
        return $"Shipped {orderId} from App B";
    }
}

// Generic starter so you can schedule ANY orchestration name from App B's port too.
public static class GenericStarter
{
    [Function("AppB_StartOrchestration")]
    public static async Task<HttpResponseData> Start(
        [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "start/{name}")] HttpRequestData req,
        [DurableClient] DurableTaskClient client,
        string name)
    {
        string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(name);
        return client.CreateCheckStatusResponse(req, instanceId);
    }
}
