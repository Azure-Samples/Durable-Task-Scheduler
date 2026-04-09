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
import java.util.Objects;
import java.util.concurrent.TimeoutException;

import com.azure.core.credential.AccessToken;
import com.azure.core.credential.TokenRequestContext;
import com.azure.core.credential.TokenCredential;
import com.azure.identity.*;

/**
 * Demonstrates the Durable Entities pattern using the Java Durable Task SDK.
 * <p>
 * This sample:
 * 1. Registers a counter entity that supports add, subtract, get, and reset operations
 * 2. Registers an orchestration that interacts with the counter entity
 * 3. Signals the entity directly from the client
 * 4. Runs an orchestration that signals and calls the entity
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
            // Register the orchestration that interacts with the entity
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

        // Demonstrate direct entity signaling from client
        logger.info("=== Direct Entity Operations ===");
        EntityInstanceId entityId = new EntityInstanceId("counter", Objects.requireNonNull(entityKey));

        logger.info("Signaling entity '{}' to add 100", entityKey);
        entityClient.signalEntity(entityId, "add", 100);
        Thread.sleep(2000);

        logger.info("Signaling entity '{}' to subtract 25", entityKey);
        entityClient.signalEntity(entityId, "subtract", 25);
        Thread.sleep(2000);

        // Run orchestrations that interact with entities
        logger.info("=== Orchestration-based Entity Operations ===");
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

        logger.info("=== Entity Demo Complete ===");
        logger.info("Direct entity signals sent to '{}'", entityKey);
        logger.info("Orchestrations completed successfully: {}/{}", completedOrchestrations, totalOrchestrations);
        logger.info("Orchestrations failed or timed out: {}", failedOrTimedOutOrchestrations);

        // Shutdown the worker and exit
        worker.stop();
        System.exit(0);
    }
}
