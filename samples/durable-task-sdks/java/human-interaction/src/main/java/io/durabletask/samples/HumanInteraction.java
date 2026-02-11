// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package io.durabletask.samples;

import com.microsoft.durabletask.*;
import com.microsoft.durabletask.azuremanaged.DurableTaskSchedulerClientExtensions;
import com.microsoft.durabletask.azuremanaged.DurableTaskSchedulerWorkerExtensions;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.io.IOException;
import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.time.Duration;
import java.time.Instant;
import java.util.UUID;
import java.util.concurrent.TimeoutException;
import java.util.logging.Logger;

/**
 * Sample demonstrating human interaction pattern with approval workflow.
 */
public class HumanInteraction {
    private static final String ORCHESTRATION_NAME = "ApprovalWorkflow";
    private static final Duration TIMEOUT = Duration.ofHours(1); // Changed from 1 minute to 1 hour
    private static final Logger logger = Logger.getLogger(HumanInteraction.class.getName());

    public static void main(String[] args) throws IOException, InterruptedException, TimeoutException {
        // Get connection string from environment variable
        String connectionString = System.getenv("DURABLE_TASK_CONNECTION_STRING");
        if (connectionString == null) {
            // Default to local development connection string if not set
            connectionString = "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";
        }

        // Create worker and register orchestration and activities
        DurableTaskGrpcWorker worker = createWorker(connectionString);
        worker.start();
        Thread.sleep(5000);

        // Create client
        DurableTaskClient client = DurableTaskSchedulerClientExtensions.createClientBuilder(connectionString).build();

        try {
            logger.info("=== Human Interaction Pattern Sample ===");
            logger.info("This sample demonstrates approval workflow with human interaction.\n");

            // Create approval request
            ApprovalRequest request = new ApprovalRequest(
                "Console User",
                "Vacation Request",
                TIMEOUT.toHours()
            );
            
            String instanceId = UUID.randomUUID().toString();
            logger.info("Creating new approval request...");
            
            // Start orchestration
            client.scheduleNewOrchestrationInstance(
                ORCHESTRATION_NAME,
                new NewOrchestrationInstanceOptions()
                    .setInput(request)
                    .setInstanceId(instanceId));
            
            logger.info("Request created with ID: " + instanceId + "\n");

            // Check initial status
            logger.info("Initial status:");
            printStatus(client.getInstanceMetadata(instanceId, true));

            // Get user decision using System.console()
            logger.info("\nPress Enter to approve the request, or type 'reject' and press Enter to reject: ");
            
            String input = "";
            try {
                BufferedReader reader = new BufferedReader(new InputStreamReader(System.in));
                input = reader.readLine();
            } catch (IOException e) {
                // Silently handle error and default to approval
                logger.warning("Warning: Error reading console input. Defaulting to approval.");
            }
            
            boolean isApproved = input == null || !input.trim().toLowerCase().equals("reject");

            // Send approval response
            ApprovalResponse response = new ApprovalResponse(
                isApproved,
                "Console User",
                "Response from console application",
                Instant.now().toString()
            );

            logger.info("\nSubmitting your response...");
            client.raiseEvent(instanceId, "ApprovalResponse", response);

            // Wait and check final status
            logger.info("\nWaiting for final status...");
            Thread.sleep(2000); // Give orchestration time to process
            
            logger.info("Final status:");
            printStatus(client.getInstanceMetadata(instanceId, true));
            
            logger.info("\nSample completed.");

        } finally {
            worker.stop();
            System.exit(0);
        }
    }

    private static DurableTaskGrpcWorker createWorker(String connectionString) {
        return DurableTaskSchedulerWorkerExtensions.createWorkerBuilder(connectionString)
            .addOrchestration(new TaskOrchestrationFactory() {
                @Override
                public String getName() {
                    return ORCHESTRATION_NAME;
                }

                @Override
                public TaskOrchestration create() {
                    return ctx -> {
                        // Get the approval request
                        ApprovalRequest request = ctx.getInput(ApprovalRequest.class);
                        
                        // Set initial status
                        ctx.setCustomStatus(new WorkflowStatus("Waiting for approval", null));

                        // Create a timeout timer
                        Task timeoutTask = ctx.createTimer(Duration.ofHours((long)request.timeoutHours));
                        
                        // Wait for either approval response or timeout
                        Task<ApprovalResponse> approvalTask = ctx.waitForExternalEvent("ApprovalResponse", ApprovalResponse.class);
                        
                        // Wait for either approval or timeout
                        Task<?> winner = ctx.anyOf(approvalTask, timeoutTask).await();
                        
                        if (winner == approvalTask) {
                            ApprovalResponse response = approvalTask.await();  // safe because we checked
                            String status = response.isApproved ? "APPROVED" : "REJECTED";
                            ctx.setCustomStatus(new WorkflowStatus(status, response));
                            ctx.complete(new WorkflowResult(
                                status,
                                response.isApproved ? "Request approved" : "Request rejected",
                                response
                            ));
                        } else {
                            // Timeout occurred first
                            ctx.setCustomStatus(new WorkflowStatus("Timed out", null));
                            ctx.complete(new WorkflowResult("TIMEOUT", "Request timed out", null));
                        }
                    };
                }
            })
            .build();
    }

    private static void printStatus(OrchestrationMetadata metadata) throws IOException {
        if (metadata == null) {
            logger.info("  Status: Not found");
            return;
        }

        logger.info("  Runtime Status: " + metadata.getRuntimeStatus());
        
        if (metadata.readCustomStatusAs(WorkflowStatus.class) != null) {
            WorkflowStatus status = metadata.readCustomStatusAs(WorkflowStatus.class);
            logger.info("  Status: " + status.status);
            if (status.response != null) {
                logger.info("  Approved: " + status.response.isApproved);
                logger.info("  Approver: " + status.response.approver);
                logger.info("  Comments: " + status.response.comments);
                logger.info("  Response Time: " + status.response.responseTime);
            }
        }

        if (metadata.getRuntimeStatus() == OrchestrationRuntimeStatus.COMPLETED) {
            WorkflowResult result = metadata.readOutputAs(WorkflowResult.class);
            logger.info("  Result: " + result.status);
            logger.info("  Message: " + result.message);
        }
    }

    // Data classes
    public static class ApprovalRequest {
        @JsonProperty public final String requester;
        @JsonProperty public final String item;
        @JsonProperty public final double timeoutHours;

        public ApprovalRequest(
            @JsonProperty("requester") String requester,
            @JsonProperty("item") String item,
            @JsonProperty("timeoutHours") double timeoutHours) {
            this.requester = requester;
            this.item = item;
            this.timeoutHours = timeoutHours;
        }
    }

    public static class ApprovalResponse {
        @JsonProperty public final boolean isApproved;
        @JsonProperty public final String approver;
        @JsonProperty public final String comments;
        @JsonProperty public final String responseTime;

        public ApprovalResponse(
            @JsonProperty("isApproved") boolean isApproved,
            @JsonProperty("approver") String approver,
            @JsonProperty("comments") String comments,
            @JsonProperty("responseTime") String responseTime) {
            this.isApproved = isApproved;
            this.approver = approver;
            this.comments = comments;
            this.responseTime = responseTime;
        }
    }

    public static class WorkflowStatus {
        @JsonProperty public final String status;
        @JsonProperty public final ApprovalResponse response;

        public WorkflowStatus(
            @JsonProperty("status") String status,
            @JsonProperty("response") ApprovalResponse response) {
            this.status = status;
            this.response = response;
        }
    }

    public static class WorkflowResult {
        @JsonProperty public final String status;
        @JsonProperty public final String message;
        @JsonProperty public final ApprovalResponse response;

        public WorkflowResult(
            @JsonProperty("status") String status,
            @JsonProperty("message") String message,
            @JsonProperty("response") ApprovalResponse response) {
            this.status = status;
            this.message = message;
            this.response = response;
        }
    }
} 