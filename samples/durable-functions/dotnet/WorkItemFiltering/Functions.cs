// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

using System.Net;
using Microsoft.Azure.Functions.Worker;
using Microsoft.Azure.Functions.Worker.Http;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Entities;
using Microsoft.Extensions.Logging;

namespace WorkItemFiltering;

// =============================================================================
// Orchestrations
// =============================================================================

/// <summary>
/// A simple orchestration that calls an activity and returns the result.
/// With work item filtering enabled, DTS will only dispatch this orchestration
/// to workers that have it registered.
/// </summary>
public static class GreetingOrchestration
{
    [Function(nameof(GreetingOrchestration))]
    public static async Task<string> Run([OrchestrationTrigger] TaskOrchestrationContext ctx)
    {
        ctx.CreateReplaySafeLogger(nameof(GreetingOrchestration)).LogInformation("GreetingOrchestration started");
        return await ctx.CallActivityAsync<string>(nameof(SayHello), "World");
    }

    [Function(nameof(GreetingOrchestration) + "_Start")]
    public static async Task<HttpResponseData> Start(
        [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "orchestrators/greeting")] HttpRequestData req,
        [DurableClient] DurableTaskClient client)
    {
        string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(nameof(GreetingOrchestration));
        return client.CreateCheckStatusResponse(req, instanceId);
    }
}

/// <summary>
/// A fan-out/fan-in orchestration that calls the same activity in parallel.
/// Demonstrates that activity work items are also filtered.
/// </summary>
public static class FanOutOrchestration
{
    [Function(nameof(FanOutOrchestration))]
    public static async Task<string[]> Run([OrchestrationTrigger] TaskOrchestrationContext ctx)
    {
        ctx.CreateReplaySafeLogger(nameof(FanOutOrchestration)).LogInformation("FanOutOrchestration: fanning out to 3 activities");
        return await Task.WhenAll(
            ctx.CallActivityAsync<string>(nameof(SayHello), "Tokyo"),
            ctx.CallActivityAsync<string>(nameof(SayHello), "London"),
            ctx.CallActivityAsync<string>(nameof(SayHello), "Seattle"));
    }

    [Function(nameof(FanOutOrchestration) + "_Start")]
    public static async Task<HttpResponseData> Start(
        [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "orchestrators/fanout")] HttpRequestData req,
        [DurableClient] DurableTaskClient client)
    {
        string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(nameof(FanOutOrchestration));
        return client.CreateCheckStatusResponse(req, instanceId);
    }
}

/// <summary>
/// A parent orchestration that calls a child orchestration.
/// Sub-orchestration dispatch is also governed by work item filters.
/// </summary>
public static class ParentOrchestration
{
    [Function(nameof(ParentOrchestration))]
    public static async Task<string> Run([OrchestrationTrigger] TaskOrchestrationContext ctx)
    {
        ctx.CreateReplaySafeLogger(nameof(ParentOrchestration)).LogInformation("Calling sub-orchestration");
        string result = await ctx.CallSubOrchestratorAsync<string>(nameof(GreetingOrchestration));
        return $"Parent received: {result}";
    }

    [Function(nameof(ParentOrchestration) + "_Start")]
    public static async Task<HttpResponseData> Start(
        [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "orchestrators/parent")] HttpRequestData req,
        [DurableClient] DurableTaskClient client)
    {
        string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(nameof(ParentOrchestration));
        return client.CreateCheckStatusResponse(req, instanceId);
    }
}

/// <summary>
/// An orchestration that interacts with a durable entity.
/// Entity work items are also filtered.
/// </summary>
public static class CounterOrchestration
{
    [Function(nameof(CounterOrchestration))]
    public static async Task<int> Run([OrchestrationTrigger] TaskOrchestrationContext ctx)
    {
        var logger = ctx.CreateReplaySafeLogger(nameof(CounterOrchestration));
        var entityId = new EntityInstanceId(nameof(CounterEntity), "sample-counter");

        await ctx.Entities.CallEntityAsync(entityId, "Add", 10);
        await ctx.Entities.CallEntityAsync(entityId, "Add", 20);
        int value = await ctx.Entities.CallEntityAsync<int>(entityId, "Get");

        logger.LogInformation("Counter value = {Value}", value);
        return value;
    }

    [Function(nameof(CounterOrchestration) + "_Start")]
    public static async Task<HttpResponseData> Start(
        [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "orchestrators/counter")] HttpRequestData req,
        [DurableClient] DurableTaskClient client)
    {
        string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(nameof(CounterOrchestration));
        return client.CreateCheckStatusResponse(req, instanceId);
    }
}

// =============================================================================
// Activities
// =============================================================================

public static class SayHello
{
    [Function(nameof(SayHello))]
    public static string Run([ActivityTrigger] string name) => $"Hello, {name}!";
}

// =============================================================================
// Entities
// =============================================================================

public class CounterEntity : TaskEntity<int>
{
    public void Add(int amount) => this.State += amount;
    public void Reset() => this.State = 0;
    public int Get() => this.State;

    [Function(nameof(CounterEntity))]
    public static Task Dispatch([EntityTrigger] TaskEntityDispatcher dispatcher)
        => dispatcher.DispatchAsync<CounterEntity>();
}

// =============================================================================
// Generic starter (for cross-app filter isolation tests)
// =============================================================================

public static class GenericStarter
{
    [Function("StartOrchestration")]
    public static async Task<HttpResponseData> Start(
        [HttpTrigger(AuthorizationLevel.Anonymous, "post", Route = "start/{name}")] HttpRequestData req,
        [DurableClient] DurableTaskClient client,
        string name)
    {
        string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(name);
        return client.CreateCheckStatusResponse(req, instanceId);
    }
}
