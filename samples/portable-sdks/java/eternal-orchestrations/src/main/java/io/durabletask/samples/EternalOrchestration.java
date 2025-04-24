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

/**
 * A simple sample demonstrating an eternal orchestration that runs forever
 * and processes work items periodically.
 */
public class EternalOrchestration {
    private static final Logger logger = LoggerFactory.getLogger(EternalOrchestration.class);
    private static final String ORCHESTRATION_NAME = "SimpleEternalOrchestration";
    private static final Duration WORK_INTERVAL = Duration.ofSeconds(10);

    public static void main(String[] args) throws IOException, InterruptedException, TimeoutException {
        // Get connection string from environment variable
        String connectionString = System.getenv("DURABLE_TASK_CONNECTION_STRING");
        if (connectionString == null) {
            // Default to local development connection string if not set
            connectionString = "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";
        }
        
        // Create worker and register orchestration and activities
        DurableTaskGrpcWorker worker = DurableTaskSchedulerWorkerExtensions.createWorkerBuilder(connectionString)
            .addOrchestration(new TaskOrchestrationFactory() {
                @Override
                public String getName() {
                    return ORCHESTRATION_NAME;
                }

                @Override
                public TaskOrchestration create() {
                    return ctx -> {
                        // Get the current counter value from input
                        Integer inputCounter = ctx.getInput(Integer.class);
                        int counter = (inputCounter != null) ? inputCounter : 0;
                        
                        // Do some work
                        String result = ctx.callActivity(
                            "ProcessWorkItem",
                            "Work item " + counter,
                            String.class).await();
                            
                        // Only log when not replaying
                        if (!ctx.getIsReplaying()) {
                            logger.info("Work item processed: {}", result);
                        }
                            
                        // Wait before processing next item
                        ctx.createTimer(WORK_INTERVAL).await();

                        // Continue the orchestration with the incremented counter
                        ctx.continueAsNew(counter + 1);
                    };
                }
            })
            .addActivity(new TaskActivityFactory() {
                @Override
                public String getName() {
                    return "ProcessWorkItem";
                }

                @Override
                public TaskActivity create() {
                    return ctx -> {
                        String workItem = ctx.getInput(String.class);
                        // Simulate some work
                        try {
                            Thread.sleep(1000);
                        } catch (InterruptedException e) {
                            Thread.currentThread().interrupt();
                            throw new RuntimeException("Work interrupted", e);
                        }
                        return "Processed " + workItem + " at " + System.currentTimeMillis();
                    };
                }
            })
            .build();

        // Start the worker
        worker.start();

        // Create client and start orchestration
        DurableTaskClient client = DurableTaskSchedulerClientExtensions.createClientBuilder(connectionString).build();
        String instanceId = client.scheduleNewOrchestrationInstance(ORCHESTRATION_NAME, 0); // Start with counter 0
        logger.info("Started eternal orchestration with ID: {}", instanceId);

        // Keep the main application alive
        // In a real application, you might want to handle shutdown more gracefully
        Thread.sleep(Duration.ofMinutes(5).toMillis());  // Run for 5 minutes then exit

        worker.stop();
    }
} 