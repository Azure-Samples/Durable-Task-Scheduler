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

final class ChainingPattern {
    private static final Logger logger = LoggerFactory.getLogger(ChainingPattern.class);

    public static void main(String[] args) throws IOException, InterruptedException, TimeoutException {
        // Get connection string from environment variable
        String connectionString = System.getenv("DURABLE_TASK_CONNECTION_STRING");
        if (connectionString == null) {
            // Default to local development connection string if not set
            connectionString = "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";
        }

        // Create worker using Azure-managed extensions
        DurableTaskGrpcWorker worker = DurableTaskSchedulerWorkerExtensions.createWorkerBuilder(connectionString)
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

        // Start the worker
        worker.start();

        // Create client using Azure-managed extensions
        DurableTaskClient client = DurableTaskSchedulerClientExtensions.createClientBuilder(connectionString).build();

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