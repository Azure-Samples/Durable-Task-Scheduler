// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package io.durabletask.samples;

import com.microsoft.durabletask.*;
import com.microsoft.durabletask.azuremanaged.DurableTaskSchedulerClientExtensions;
import com.microsoft.durabletask.azuremanaged.DurableTaskSchedulerWorkerExtensions;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.time.Duration;
import java.util.List;
import java.util.Objects;
import java.util.concurrent.TimeoutException;

import com.azure.core.credential.AccessToken;
import com.azure.core.credential.TokenRequestContext;
import com.azure.core.credential.TokenCredential;
import com.azure.identity.*;

/**
 * Demonstrates the Durable Entities pattern using the Java Durable Task SDK.
 * <p>
 * This sample demonstrates:
 * 1. A counter entity that supports add, subtract, get, and reset operations
 * 2. An account entity with deposit, withdraw, getBalance, and reset operations
 * 3. An audit entity that records events signaled from other entities
 * 4. Locking entities: a TransferFunds orchestration that locks two account entities
 *    to perform an atomic balance transfer in a critical section
 * 5. Entities signaling other entities: the account entity signals an audit entity
 *    when a large deposit occurs
 * 6. Entities starting orchestrations: the account entity starts an audit orchestration
 *    when a large withdrawal occurs
 */
final class EntitiesPattern {
    private static final Logger logger = LoggerFactory.getLogger(EntitiesPattern.class);
    private static final String DEFAULT_ENTITY_KEY = "my-counter";

    public static void main(String[] args) throws IOException, InterruptedException, TimeoutException {
        // Get environment variables for endpoint and taskhub with defaults
        String endpoint = System.getenv("ENDPOINT");
        String taskHubName = System.getenv("TASKHUB");
        String connectionString = System.getenv("DURABLE_TASK_CONNECTION_STRING");

        if (connectionString == null) {
            if (endpoint != null && taskHubName != null) {
                String hostAddress = endpoint;
                if (endpoint.contains(";")) {
                    hostAddress = endpoint.split(";")[0];
                }

                boolean isLocalEmulator = endpoint.equals("http://localhost:8080");

                if (isLocalEmulator) {
                    connectionString = String.format("Endpoint=%s;TaskHub=%s;Authentication=None", hostAddress, taskHubName);
                    logger.info("Using local emulator with no authentication");
                } else {
                    connectionString = String.format("Endpoint=%s;TaskHub=%s;Authentication=DefaultAzure", hostAddress, taskHubName);
                    logger.info("Using Azure endpoint with DefaultAzure authentication");
                }

                logger.info("Using endpoint: {}", endpoint);
                logger.info("Using task hub: {}", taskHubName);
            } else {
                connectionString = "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";
                logger.info("Using default local emulator connection string");
            }
        }

        // Check if we're running in Azure with a managed identity
        String clientId = System.getenv("AZURE_MANAGED_IDENTITY_CLIENT_ID");
        TokenCredential credential = null;
        if (clientId != null && !clientId.isEmpty()) {
            logger.info("Using Managed Identity with client ID: {}", clientId);
            credential = new ManagedIdentityCredentialBuilder().clientId(clientId).build();

            AccessToken token = credential.getToken(
                        new TokenRequestContext().addScopes("https://management.azure.com/.default"))
                        .block(Duration.ofSeconds(10));
            logger.info("Successfully authenticated with Managed Identity, expires at {}", token.getExpiresAt());

        } else if (!connectionString.contains("Authentication=None")) {
            logger.info("No Managed Identity client ID found, using DefaultAzure authentication");
        }

        // Create worker using Azure-managed extensions
        DurableTaskGrpcWorker worker = (credential != null
            ? DurableTaskSchedulerWorkerExtensions.createWorkerBuilder(endpoint, taskHubName, credential)
            : DurableTaskSchedulerWorkerExtensions.createWorkerBuilder(connectionString))
            // Register the counter entity
            .addEntity("counter", CounterEntity::new)
            // Register the account entity (demonstrates entity signaling + entity starting orchestrations)
            .addEntity("account", AccountEntity::new)
            // Register the audit entity (target of entity-to-entity signals)
            .addEntity("audit", AuditEntity::new)
            // Register the orchestration that interacts with the counter entity
            .addOrchestration(new TaskOrchestrationFactory() {
                @Override
                public String getName() { return "CounterWorkflow"; }

                @Override
                public TaskOrchestration create() {
                    return ctx -> {
                        String entityKey = ctx.getInput(String.class);
                        if (entityKey == null || entityKey.isBlank()) {
                            entityKey = DEFAULT_ENTITY_KEY;
                        }
                        EntityInstanceId entityId = new EntityInstanceId("counter", Objects.requireNonNull(entityKey));

                        // Signal entity operations (fire-and-forget)
                        ctx.signalEntity(entityId, "add", 10);
                        ctx.signalEntity(entityId, "add", 5);
                        ctx.signalEntity(entityId, "subtract", 3);

                        // Call entity and wait for result
                        int value = ctx.callEntity(entityId, "get", Integer.class).await();

                        ctx.complete("Counter '" + entityKey + "' final value: " + value);
                    };
                }
            })
            // Register the TransferFunds orchestration that locks two account entities (critical section)
            .addOrchestration(new TaskOrchestrationFactory() {
                @Override
                public String getName() { return "TransferFunds"; }

                @Override
                public TaskOrchestration create() {
                    return ctx -> {
                        // Input format: "sourceAccount,destinationAccount,amount"
                        String input = ctx.getInput(String.class);
                        String[] parts = input.split(",");
                        String sourceKey = parts[0];
                        String destKey = parts[1];
                        int amount = Integer.parseInt(parts[2]);

                        EntityInstanceId sourceId = new EntityInstanceId("account", sourceKey);
                        EntityInstanceId destId = new EntityInstanceId("account", destKey);

                        // Lock both entities to ensure atomic transfer (critical section)
                        AutoCloseable lock = ctx.lockEntities(sourceId, destId).await();
                        // Check balance of source account
                        int sourceBalance = ctx.callEntity(sourceId, "getBalance", Integer.class).await();
                        if (sourceBalance >= amount) {
                            // Withdraw from source and deposit into destination
                            ctx.callEntity(sourceId, "withdraw", amount, Void.class).await();
                            ctx.callEntity(destId, "deposit", amount, Void.class).await();
                            ctx.complete(String.format(
                                "Transferred %d from '%s' to '%s'", amount, sourceKey, destKey));
                        } else {
                            ctx.complete(String.format(
                                "Insufficient funds in '%s': balance=%d, requested=%d",
                                sourceKey, sourceBalance, amount));
                        }
                    };
                }
            })
            // Register the AuditWorkflow (started by AccountEntity on large withdrawals)
            .addOrchestration(new TaskOrchestrationFactory() {
                @Override
                public String getName() { return "AuditWorkflow"; }

                @Override
                public TaskOrchestration create() {
                    return ctx -> {
                        String auditMessage = ctx.getInput(String.class);
                        // Call an activity to process the audit (e.g., log to external system)
                        String result = ctx.callActivity("ProcessAudit", auditMessage, String.class).await();
                        ctx.complete(result);
                    };
                }
            })
            // Register the ProcessAudit activity
            .addActivity(new TaskActivityFactory() {
                @Override
                public String getName() { return "ProcessAudit"; }

                @Override
                public TaskActivity create() {
                    return ctx -> {
                        String message = ctx.getInput(String.class);
                        logger.info("ProcessAudit activity: {}", message);
                        return "Audit processed: " + message;
                    };
                }
            })
            .build();

        // Start the worker and wait for it to connect
        worker.start();
        Thread.sleep(5000);

        // Create client using Azure-managed extensions
        DurableTaskClient client = (credential != null
            ? DurableTaskSchedulerClientExtensions.createClientBuilder(endpoint, taskHubName, credential)
            : DurableTaskSchedulerClientExtensions.createClientBuilder(connectionString)).build();

        String entityKey = DEFAULT_ENTITY_KEY;
        if (args.length > 0 && args[0] != null && !args[0].isBlank()) {
            entityKey = args[0];
        }

        DurableEntityClient entityClient = client.getEntities();

        // =============================================
        // Part 1: Basic counter entity operations
        // =============================================
        logger.info("=== Part 1: Direct Entity Operations (Counter) ===");
        EntityInstanceId entityId = new EntityInstanceId("counter", Objects.requireNonNull(entityKey));

        logger.info("Signaling entity '{}' to add 100", entityKey);
        entityClient.signalEntity(entityId, "add", 100);
        Thread.sleep(2000);

        logger.info("Signaling entity '{}' to subtract 25", entityKey);
        entityClient.signalEntity(entityId, "subtract", 25);
        Thread.sleep(2000);

        // Run orchestrations that interact with counter entities
        logger.info("=== Counter Orchestration-based Entity Operations ===");
        int totalOrchestrations = 5;
        int completedOrchestrations = 0;
        int failedOrTimedOutOrchestrations = 0;

        for (int i = 0; i < totalOrchestrations; i++) {
            String instanceEntityKey = entityKey + "-orch-" + (i + 1);
            logger.info("Scheduling orchestration #{} for entity '{}'", i + 1, instanceEntityKey);

            String instanceId = client.scheduleNewOrchestrationInstance(
                "CounterWorkflow",
                new NewOrchestrationInstanceOptions().setInput(instanceEntityKey));
            logger.info("Orchestration #{} scheduled with ID: {}", i + 1, instanceId);

            OrchestrationMetadata result = client.waitForInstanceCompletion(
                instanceId, Duration.ofSeconds(120), true);

            if (result != null) {
                if (result.isCompleted()) {
                    completedOrchestrations++;
                    logger.info("Orchestration {} completed: {}", instanceId, result.readOutputAs(String.class));
                } else {
                    failedOrTimedOutOrchestrations++;
                    logger.error("Orchestration {} did not complete successfully: {}", instanceId, result.getRuntimeStatus());
                }
            } else {
                failedOrTimedOutOrchestrations++;
                logger.warn("Orchestration {} did not complete within the timeout period", instanceId);
            }
        }

        logger.info("Counter orchestrations completed: {}/{}", completedOrchestrations, totalOrchestrations);
        logger.info("Counter orchestrations failed or timed out: {}", failedOrTimedOutOrchestrations);

        // =============================================
        // Part 2: Entity Locking (Critical Section)
        // =============================================
        logger.info("");
        logger.info("=== Part 2: Locking Entities (TransferFunds with Critical Section) ===");

        // Initialize two accounts with deposits
        EntityInstanceId account1 = new EntityInstanceId("account", "account-A");
        EntityInstanceId account2 = new EntityInstanceId("account", "account-B");

        logger.info("Depositing 1000 into account-A");
        entityClient.signalEntity(account1, "deposit", 1000);
        Thread.sleep(2000);

        logger.info("Depositing 200 into account-B");
        entityClient.signalEntity(account2, "deposit", 200);
        Thread.sleep(2000);

        // Transfer 300 from account-A to account-B using critical section locking
        logger.info("Starting TransferFunds orchestration: 300 from account-A to account-B");
        String transferId = client.scheduleNewOrchestrationInstance(
            "TransferFunds",
            new NewOrchestrationInstanceOptions().setInput("account-A,account-B,300"));
        OrchestrationMetadata transferResult = client.waitForInstanceCompletion(
            transferId, Duration.ofSeconds(120), true);
        if (transferResult != null && transferResult.isCompleted()) {
            logger.info("Transfer completed: {}", transferResult.readOutputAs(String.class));
        } else {
            logger.warn("Transfer did not complete within timeout");
        }

        // =============================================
        // Part 3: Entity Signaling Other Entities
        // =============================================
        logger.info("");
        logger.info("=== Part 3: Entity Signaling Other Entities ===");
        // A large deposit (>= 500) triggers the account entity to signal the audit entity
        logger.info("Making a large deposit of 600 into account-A (triggers entity-to-entity signal to audit entity)");
        entityClient.signalEntity(account1, "deposit", 600);
        Thread.sleep(3000);
        logger.info("The account entity signaled the audit entity to record this large transaction");

        // =============================================
        // Part 4: Entity Starting Orchestrations
        // =============================================
        logger.info("");
        logger.info("=== Part 4: Entity Starting Orchestrations ===");
        // A large withdrawal (>= 500) triggers the account entity to start an AuditWorkflow orchestration
        logger.info("Making a large withdrawal of 500 from account-A (triggers entity to start AuditWorkflow orchestration)");
        entityClient.signalEntity(account1, "withdraw", 500);
        Thread.sleep(5000);
        logger.info("The account entity started an AuditWorkflow orchestration for this large withdrawal");

        logger.info("");
        logger.info("=== Entity Demo Complete ===");
        logger.info("Part 1 - Counter entity signals sent to '{}'", entityKey);
        logger.info("Part 2 - TransferFunds with entity locking demonstrated");
        logger.info("Part 3 - Entity-to-entity signaling demonstrated (AccountEntity -> AuditEntity)");
        logger.info("Part 4 - Entity starting orchestration demonstrated (AccountEntity -> AuditWorkflow)");

        // Shutdown the worker and exit
        worker.stop();
        System.exit(0);
    }
}
