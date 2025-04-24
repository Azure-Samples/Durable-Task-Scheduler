// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package io.durabletask.samples;

import com.microsoft.durabletask.*;
import com.microsoft.durabletask.azuremanaged.DurableTaskSchedulerClientExtensions;
import com.microsoft.durabletask.azuremanaged.DurableTaskSchedulerWorkerExtensions;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.time.Duration;
import java.util.UUID;
import java.util.logging.Logger;

/**
 * Sample demonstrating sub-orchestration patterns.
 * This sample shows how to create and manage sub-orchestrations.
 */
public class SubOrchestrationPattern {
    private static final String PARENT_ORCHESTRATION_NAME = "ParentOrchestration";
    private static final String CHILD_ORCHESTRATION_NAME = "ChildOrchestration";
    private static final Logger logger = Logger.getLogger(SubOrchestrationPattern.class.getName());

    public static void main(String[] args) throws Exception {
        // Get connection string from environment variable
        String connectionString = System.getenv("DURABLE_TASK_CONNECTION_STRING");
        if (connectionString == null) {
            // Default to local development connection string if not set
            connectionString = "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";
        }
        // Create worker and register orchestrations
        DurableTaskGrpcWorker worker = createWorker(connectionString);
        worker.start();

        // Create client
        DurableTaskClient client = DurableTaskSchedulerClientExtensions.createClientBuilder(connectionString).build();

        try {
            logger.info("=== Sub-Orchestration Pattern Sample ===");
            logger.info("This sample demonstrates parent-child orchestration relationships.\n");

            // Create input for parent orchestration
            ParentInput parentInput = new ParentInput(
                "Parent Task",
                3,  // Number of child orchestrations to create
                2   // Delay in seconds between child orchestrations
            );
            
            String instanceId = UUID.randomUUID().toString();
            logger.info("Starting parent orchestration...");
            
            // Start parent orchestration
            client.scheduleNewOrchestrationInstance(
                PARENT_ORCHESTRATION_NAME,
                new NewOrchestrationInstanceOptions()
                    .setInput(parentInput)
                    .setInstanceId(instanceId));
            
            logger.info("Parent orchestration started with ID: " + instanceId);
            
            // Wait for parent orchestration to complete
            logger.info("Waiting for parent orchestration to complete...");
            OrchestrationMetadata state = client.waitForInstanceCompletion(
                instanceId,
                Duration.ofSeconds(30),
                true);
            
            if (state != null) {
                if (state.getRuntimeStatus() == OrchestrationRuntimeStatus.COMPLETED) {
                    ParentResult result = state.readOutputAs(ParentResult.class);
                    logger.info("Parent orchestration completed successfully!");
                    logger.info("Total children completed: " + result.totalChildren);
                    logger.info("Final message: " + result.message);
                } else {
                    logger.warning("Parent orchestration failed: " + state.getFailureDetails());
                }
            } else {
                logger.warning("Parent orchestration state not found");
            }
            
        } finally {
            worker.stop();
        }
    }

    private static DurableTaskGrpcWorker createWorker(String connectionString) {
        return DurableTaskSchedulerWorkerExtensions.createWorkerBuilder(connectionString)
            // Register parent orchestration
            .addOrchestration(new TaskOrchestrationFactory() {
                @Override
                public String getName() {
                    return PARENT_ORCHESTRATION_NAME;
                }

                @Override
                public TaskOrchestration create() {
                    return ctx -> {
                        // Get the parent input
                        ParentInput input = ctx.getInput(ParentInput.class);
                        
                        // Set initial status
                        ctx.setCustomStatus(new Status("Starting parent orchestration"));
                        
                        int totalChildren = 0;
                        for (int i = 0; i < input.childCount; i++) {
                            // Create child input
                            ChildInput childInput = new ChildInput(
                                "Child " + (i + 1),
                                input.delaySeconds
                            );
                            
                            // Start child orchestration
                            String childId = ctx.callSubOrchestrator(
                                CHILD_ORCHESTRATION_NAME,
                                childInput,
                                String.class).await();
                            
                            logger.info("Child orchestration " + (i + 1) + " completed with ID: " + childId);
                            totalChildren++;
                            
                            // Update status
                            ctx.setCustomStatus(new Status("Completed " + totalChildren + " of " + input.childCount + " children"));
                        }
                        
                        // Complete parent orchestration
                        ctx.complete(new ParentResult(
                            totalChildren,
                            "All " + totalChildren + " child orchestrations completed successfully"
                        ));
                    };
                }
            })
            // Register child orchestration
            .addOrchestration(new TaskOrchestrationFactory() {
                @Override
                public String getName() {
                    return CHILD_ORCHESTRATION_NAME;
                }

                @Override
                public TaskOrchestration create() {
                    return ctx -> {
                        // Get the child input
                        ChildInput input = ctx.getInput(ChildInput.class);
                        
                        // Set initial status
                        ctx.setCustomStatus(new Status("Starting child orchestration: " + input.name));
                        
                        // Simulate some work
                        ctx.createTimer(Duration.ofSeconds(input.delaySeconds)).await();
                        
                        // Complete with a result
                        ctx.complete("Child orchestration " + input.name + " completed");
                    };
                }
            })
            .build();
    }

    // Data classes
    public static class ParentInput {
        @JsonProperty public final String name;
        @JsonProperty public final int childCount;
        @JsonProperty public final int delaySeconds;

        public ParentInput(
            @JsonProperty("name") String name,
            @JsonProperty("childCount") int childCount,
            @JsonProperty("delaySeconds") int delaySeconds) {
            this.name = name;
            this.childCount = childCount;
            this.delaySeconds = delaySeconds;
        }
    }

    public static class ChildInput {
        @JsonProperty public final String name;
        @JsonProperty public final int delaySeconds;

        public ChildInput(
            @JsonProperty("name") String name,
            @JsonProperty("delaySeconds") int delaySeconds) {
            this.name = name;
            this.delaySeconds = delaySeconds;
        }
    }

    public static class ParentResult {
        @JsonProperty public final int totalChildren;
        @JsonProperty public final String message;

        public ParentResult(
            @JsonProperty("totalChildren") int totalChildren,
            @JsonProperty("message") String message) {
            this.totalChildren = totalChildren;
            this.message = message;
        }
    }

    public static class Status {
        @JsonProperty public final String status;

        public Status(@JsonProperty("status") String status) {
            this.status = status;
        }
    }
} 