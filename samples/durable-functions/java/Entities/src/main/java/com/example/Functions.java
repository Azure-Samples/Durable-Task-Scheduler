package com.example;

import com.microsoft.azure.functions.*;
import com.microsoft.azure.functions.annotation.*;
import com.microsoft.durabletask.*;
import com.microsoft.durabletask.azurefunctions.*;

/**
 * Azure Functions with Durable Entities.
 * <p>
 * This sample demonstrates:
 * 1. A counter entity that supports add, subtract, get, and reset operations
 * 2. An orchestration that signals and calls the counter entity
 * 3. HTTP triggers to start orchestrations and signal entities directly
 */
public class Functions {
    private static final String DEFAULT_ENTITY_KEY = "my-counter";


    /**
     * Entity function for the counter entity.
     * The @DurableEntityTrigger binds to incoming entity operation requests.
     */
    @FunctionName("Counter")
    public String counterEntity(
            @DurableEntityTrigger(name = "req", entityName = "Counter") String req) {
        return EntityRunner.loadAndRun(req, CounterEntity::new);
    }

    /**
     * Orchestration that interacts with the counter entity.
     * Signals add/subtract operations and calls get to retrieve the value.
     */
    @FunctionName("CounterOrchestration")
    public String counterOrchestration(
            @DurableOrchestrationTrigger(name = "ctx") TaskOrchestrationContext ctx) {

        String entityKey = ctx.getInput(String.class);
        if (entityKey == null || entityKey.isBlank()) {
            entityKey = DEFAULT_ENTITY_KEY;
        }
        EntityInstanceId entityId = new EntityInstanceId("Counter", entityKey);

        // Signal entity operations (fire-and-forget)
        ctx.signalEntity(entityId, "add", 10);
        ctx.signalEntity(entityId, "add", 5);
        ctx.signalEntity(entityId, "subtract", 3);

        // Call entity and wait for result
        int value = ctx.callEntity(entityId, "get", Integer.class).await();

        return "Counter '" + entityKey + "' final value: " + value;
    }

    /**
     * HTTP trigger that starts the counter orchestration.
     * POST /api/StartCounterOrchestration?key=my-counter
     */
    @FunctionName("StartCounterOrchestration")
    public HttpResponseMessage startCounterOrchestration(
            @HttpTrigger(name = "req", methods = {HttpMethod.POST}, authLevel = AuthorizationLevel.ANONYMOUS)
            HttpRequestMessage<Void> request,
            @DurableClientInput(name = "durableContext") DurableClientContext durableContext) {

        DurableTaskClient client = durableContext.getClient();
        String entityKey = request.getQueryParameters().getOrDefault("key", DEFAULT_ENTITY_KEY);

        String instanceId = client.scheduleNewOrchestrationInstance(
            "CounterOrchestration",
            new NewOrchestrationInstanceOptions().setInput(entityKey));

        return durableContext.createCheckStatusResponse(request, instanceId);
    }

    /**
     * HTTP trigger that signals the counter entity directly.
     * POST /api/SignalCounter?key=my-counter&op=add&value=10
     */
    @FunctionName("SignalCounter")
    public HttpResponseMessage signalCounter(
            @HttpTrigger(name = "req", methods = {HttpMethod.POST}, authLevel = AuthorizationLevel.ANONYMOUS)
            HttpRequestMessage<Void> request,
            @DurableClientInput(name = "durableContext") DurableClientContext durableContext) {

        String entityKey = request.getQueryParameters().getOrDefault("key", DEFAULT_ENTITY_KEY);
        String operation = request.getQueryParameters().getOrDefault("op", "add");
        String valueStr = request.getQueryParameters().getOrDefault("value", "1");

        EntityInstanceId entityId = new EntityInstanceId("Counter", entityKey);

        if ("get".equals(operation)) {
            return request.createResponseBuilder(HttpStatus.BAD_REQUEST)
                .body("Use GET /api/GetCounter?key=<entity-key> to read entity state.")
                .build();
        }

        if ("reset".equals(operation)) {
            durableContext.signalEntity(entityId, operation);
        } else {
            int value = Integer.parseInt(valueStr);
            durableContext.signalEntity(entityId, operation, value);
        }

        return request.createResponseBuilder(HttpStatus.ACCEPTED)
            .body("Signal sent: " + operation + " on entity '" + entityKey + "'")
            .build();
    }

    /**
     * HTTP trigger that gets the current state of a counter entity.
     * GET /api/GetCounter?key=my-counter
     */
    @FunctionName("GetCounter")
    public HttpResponseMessage getCounter(
            @HttpTrigger(name = "req", methods = {HttpMethod.GET}, authLevel = AuthorizationLevel.ANONYMOUS)
            HttpRequestMessage<Void> request,
            @DurableClientInput(name = "durableContext") DurableClientContext durableContext) {

        String entityKey = request.getQueryParameters().getOrDefault("key", DEFAULT_ENTITY_KEY);
        EntityInstanceId entityId = new EntityInstanceId("Counter", entityKey);

        EntityMetadata metadata = durableContext.getEntityMetadata(entityId, true);

        if (metadata == null) {
            return request.createResponseBuilder(HttpStatus.NOT_FOUND)
                .body("Entity '" + entityKey + "' not found")
                .build();
        }

        Integer state = metadata.readStateAs(Integer.class);
        return request.createResponseBuilder(HttpStatus.OK)
            .header("Content-Type", "application/json")
            .body("{\"key\": \"" + entityKey + "\", \"value\": " + state + "}")
            .build();
    }
}
