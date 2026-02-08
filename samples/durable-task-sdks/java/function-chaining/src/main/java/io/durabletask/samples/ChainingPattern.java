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
import java.util.concurrent.TimeoutException;

import com.azure.core.credential.AccessToken;
import com.azure.core.credential.TokenRequestContext;
import com.azure.core.credential.TokenCredential;
import com.azure.identity.*;

final class ChainingPattern {
    private static final Logger logger = LoggerFactory.getLogger(ChainingPattern.class);

    public static void main(String[] args) throws IOException, InterruptedException, TimeoutException {
        // Get environment variables for endpoint and taskhub with defaults
        String endpoint = System.getenv("ENDPOINT");
        String taskHubName = System.getenv("TASKHUB");
        String connectionString = System.getenv("DURABLE_TASK_CONNECTION_STRING");
        
        if (connectionString == null) {
            if (endpoint != null && taskHubName != null) {
                // Use endpoint and taskhub from environment variables
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
                // Default to local development connection string if not set
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
            // If no client ID is found but we're not using local emulator, keep using DefaultAzure
            logger.info("No Managed Identity client ID found, using DefaultAzure authentication");
        }

        // Create worker using Azure-managed extensions
        DurableTaskGrpcWorker worker = (credential != null 
            ? DurableTaskSchedulerWorkerExtensions.createWorkerBuilder(endpoint, taskHubName, credential)
            : DurableTaskSchedulerWorkerExtensions.createWorkerBuilder(connectionString))
            .addOrchestration(new TaskOrchestrationFactory() {
                @Override
                public String getName() { return "ActivityChaining"; }

                @Override
                public TaskOrchestration create() {
                    return ctx -> {
                        String input = ctx.getInput(String.class);
                        String x = ctx.callActivity("Reverse", input, String.class).await();
                        String y = ctx.callActivity("Capitalize", x, String.class).await();
                        String z = ctx.callActivity("ReplaceWhitespace", y, String.class).await();
                        ctx.complete(z);
                    };
                }
            })
            .addActivity(new TaskActivityFactory() {
                @Override
                public String getName() { return "Reverse"; }

                @Override
                public TaskActivity create() {
                    return ctx -> {
                        String input = ctx.getInput(String.class);
                        StringBuilder builder = new StringBuilder(input);
                        builder.reverse();
                        return builder.toString();
                    };
                }
            })
            .addActivity(new TaskActivityFactory() {
                @Override
                public String getName() { return "Capitalize"; }

                @Override
                public TaskActivity create() {
                    return ctx -> ctx.getInput(String.class).toUpperCase();
                }
            })
            .addActivity(new TaskActivityFactory() {
                @Override
                public String getName() { return "ReplaceWhitespace"; }

                @Override
                public TaskActivity create() {
                    return ctx -> {
                        String input = ctx.getInput(String.class);
                        return input.trim().replaceAll("\\s", "-");
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

        // Start a new instance of the registered "ActivityChaining" orchestration
        String instanceId = client.scheduleNewOrchestrationInstance(
                "ActivityChaining",
                new NewOrchestrationInstanceOptions().setInput("Hello, world!"));
        logger.info("Started new orchestration instance: {}", instanceId);

        // Block until the orchestration completes. Then print the final status, which includes the output.
        OrchestrationMetadata completedInstance = client.waitForInstanceCompletion(
                instanceId,
                Duration.ofSeconds(30),
                true);
        logger.info("Orchestration completed: {}", completedInstance);
        logger.info("Output: {}", completedInstance.readOutputAs(String.class));

        // Shutdown the worker and exit
        worker.stop();
        System.exit(0);
    }
}