package com.example;

import com.microsoft.azure.functions.*;
import com.microsoft.azure.functions.annotation.*;
import com.microsoft.durabletask.*;
import com.microsoft.durabletask.azurefunctions.*;

/**
 * Azure Functions with Durable Entities.
 * <p>
 * This sample demonstrates:
 * 1. A counter entity with add, subtract, get, and reset operations
 * 2. An account entity with deposit, withdraw, getBalance, and reset operations
 * 3. An audit entity that records events signaled from other entities
 * 4. Locking entities: a TransferFunds orchestration that locks two account entities
 *    to perform an atomic balance transfer in a critical section
 * 5. Entities signaling other entities: the account entity signals an audit entity
 *    when a large deposit occurs
 * 6. Entities starting orchestrations: the account entity starts an audit orchestration
 *    when a large withdrawal occurs
 * 7. HTTP triggers for direct entity signals and reads
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

    // =========================================================================
    // Account Entity + Audit Entity (entity signaling + entity starting orchestrations)
    // =========================================================================

    /**
     * Entity function for the account entity.
     * Demonstrates entity-to-entity signaling (large deposits signal the audit entity)
     * and entity-starting-orchestrations (large withdrawals start an AuditOrchestration).
     */
    @FunctionName("Account")
    public String accountEntity(
            @DurableEntityTrigger(name = "req", entityName = "Account") String req) {
        return EntityRunner.loadAndRun(req, AccountEntity::new);
    }

    /**
     * Entity function for the audit entity.
     * This entity is signaled by AccountEntity to record large transaction events.
     * Demonstrates entity-to-entity signaling as the target of signals.
     */
    @FunctionName("Audit")
    public String auditEntity(
            @DurableEntityTrigger(name = "req", entityName = "Audit") String req) {
        return EntityRunner.loadAndRun(req, AuditEntity::new);
    }

    /**
     * Orchestration that transfers funds between two account entities with locking.
     * Acquires exclusive locks on both accounts to ensure an atomic transfer (critical section).
     * Demonstrates locking entities.
     */
    @FunctionName("TransferFundsOrchestration")
    public String transferFundsOrchestration(
            @DurableOrchestrationTrigger(name = "ctx") TaskOrchestrationContext ctx) {

        // Input format: "sourceAccount,destinationAccount,amount"
        String input = ctx.getInput(String.class);
        String[] parts = input.split(",");
        String sourceKey = parts[0];
        String destKey = parts[1];
        int amount = Integer.parseInt(parts[2]);

        EntityInstanceId sourceId = new EntityInstanceId("Account", sourceKey);
        EntityInstanceId destId = new EntityInstanceId("Account", destKey);

        // Lock both entities to ensure atomic transfer (critical section).
        // Use getEntities().lockEntities() for entity locking in orchestrations.
        ctx.getEntities().lockEntities(sourceId, destId).await();
        // Check balance of source account
        int sourceBalance = ctx.getEntities().callEntity(sourceId, "getBalance", Integer.class).await();
        if (sourceBalance >= amount) {
            // Withdraw from source and deposit into destination
            ctx.getEntities().callEntity(sourceId, "withdraw", amount, Void.class).await();
            ctx.getEntities().callEntity(destId, "deposit", amount, Void.class).await();
            return String.format("Transferred %d from '%s' to '%s'", amount, sourceKey, destKey);
        } else {
            return String.format("Insufficient funds in '%s': balance=%d, requested=%d",
                sourceKey, sourceBalance, amount);
        }
    }

    /**
     * Orchestration started by AccountEntity when a large withdrawal occurs.
     * Demonstrates entities starting orchestrations.
     */
    @FunctionName("AuditOrchestration")
    public String auditOrchestration(
            @DurableOrchestrationTrigger(name = "ctx") TaskOrchestrationContext ctx) {

        String auditMessage = ctx.getInput(String.class);
        String result = ctx.callActivity("ProcessAudit", auditMessage, String.class).await();
        return result;
    }

    /**
     * Activity function that processes an audit event.
     */
    @FunctionName("ProcessAudit")
    public String processAudit(
            @DurableActivityTrigger(name = "message") String message) {
        return "Audit processed: " + message;
    }

    // =========================================================================
    // HTTP triggers for Account operations
    // =========================================================================

    /**
     * HTTP trigger that signals the account entity.
     * POST /api/SignalAccount?key=account-A&op=deposit&value=100
     */
    @FunctionName("SignalAccount")
    public HttpResponseMessage signalAccount(
            @HttpTrigger(name = "req", methods = {HttpMethod.POST}, authLevel = AuthorizationLevel.ANONYMOUS)
            HttpRequestMessage<Void> request,
            @DurableClientInput(name = "durableContext") DurableClientContext durableContext) {

        String entityKey = request.getQueryParameters().getOrDefault("key", "account-A");
        String operation = request.getQueryParameters().getOrDefault("op", "deposit");
        String valueStr = request.getQueryParameters().getOrDefault("value", "1");

        EntityInstanceId entityId = new EntityInstanceId("Account", entityKey);

        if ("getBalance".equals(operation)) {
            return request.createResponseBuilder(HttpStatus.BAD_REQUEST)
                .body("Use GET /api/GetAccount?key=<entity-key> to read account balance.")
                .build();
        }

        if ("reset".equals(operation)) {
            durableContext.signalEntity(entityId, operation);
        } else {
            int value = Integer.parseInt(valueStr);
            durableContext.signalEntity(entityId, operation, value);
        }

        return request.createResponseBuilder(HttpStatus.ACCEPTED)
            .body("Signal sent: " + operation + " on account '" + entityKey + "'")
            .build();
    }

    /**
     * HTTP trigger that gets the current balance of an account entity.
     * GET /api/GetAccount?key=account-A
     */
    @FunctionName("GetAccount")
    public HttpResponseMessage getAccount(
            @HttpTrigger(name = "req", methods = {HttpMethod.GET}, authLevel = AuthorizationLevel.ANONYMOUS)
            HttpRequestMessage<Void> request,
            @DurableClientInput(name = "durableContext") DurableClientContext durableContext) {

        String entityKey = request.getQueryParameters().getOrDefault("key", "account-A");
        EntityInstanceId entityId = new EntityInstanceId("Account", entityKey);

        EntityMetadata metadata = durableContext.getEntityMetadata(entityId, true);

        if (metadata == null) {
            return request.createResponseBuilder(HttpStatus.NOT_FOUND)
                .body("Account '" + entityKey + "' not found")
                .build();
        }

        Integer balance = metadata.readStateAs(Integer.class);
        return request.createResponseBuilder(HttpStatus.OK)
            .header("Content-Type", "application/json")
            .body("{\"account\": \"" + entityKey + "\", \"balance\": " + balance + "}")
            .build();
    }

    /**
     * HTTP trigger that starts a transfer funds orchestration.
     * POST /api/TransferFunds?from=account-A&to=account-B&amount=100
     * Demonstrates locking entities for atomic transfers.
     */
    @FunctionName("TransferFunds")
    public HttpResponseMessage transferFunds(
            @HttpTrigger(name = "req", methods = {HttpMethod.POST}, authLevel = AuthorizationLevel.ANONYMOUS)
            HttpRequestMessage<Void> request,
            @DurableClientInput(name = "durableContext") DurableClientContext durableContext) {

        DurableTaskClient client = durableContext.getClient();
        String sourceKey = request.getQueryParameters().getOrDefault("from", "account-A");
        String destKey = request.getQueryParameters().getOrDefault("to", "account-B");
        String amountStr = request.getQueryParameters().getOrDefault("amount", "100");

        String input = sourceKey + "," + destKey + "," + amountStr;

        String instanceId = client.scheduleNewOrchestrationInstance(
            "TransferFundsOrchestration",
            new NewOrchestrationInstanceOptions().setInput(input));

        return durableContext.createCheckStatusResponse(request, instanceId);
    }
}
