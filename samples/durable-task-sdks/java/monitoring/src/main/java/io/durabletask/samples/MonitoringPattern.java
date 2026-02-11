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
 * Sample demonstrating monitoring pattern with job status tracking.
 */
public class MonitoringPattern {
    private static final String ORCHESTRATION_NAME = "MonitoringJobOrchestrator";
    private static final Logger logger = Logger.getLogger(MonitoringPattern.class.getName());

    public static void main(String[] args) throws Exception {
        // Get connection string from environment variable
        String connectionString = System.getenv("DURABLE_TASK_CONNECTION_STRING");
        if (connectionString == null) {
            // Default to local development connection string if not set
            connectionString = "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";
        }
        // Create worker and register orchestration
        DurableTaskGrpcWorker worker = createWorker(connectionString);
        worker.start();
        Thread.sleep(5000);

        // Create client
        DurableTaskClient client = DurableTaskSchedulerClientExtensions.createClientBuilder(connectionString).build();

        try {
            logger.info("=== Monitoring Pattern Sample ===");
            logger.info("This sample demonstrates job monitoring with status updates.\n");

            // Generate a unique job ID or use one provided as an argument
            String jobId = args.length > 0 ? args[0] : "job-" + UUID.randomUUID().toString();
            
            // Define monitoring parameters with defaults
            int pollingInterval = args.length > 1 ? Integer.parseInt(args[1]) : 5;  // seconds
            int timeout = args.length > 2 ? Integer.parseInt(args[2]) : 30;  // seconds
            
            JobData jobData = new JobData(
                jobId,
                pollingInterval,
                timeout
            );
            
            logger.info("Starting monitoring for job: " + jobId);
            logger.info("Polling interval: " + pollingInterval + " seconds");
            logger.info("Timeout: " + timeout + " seconds");
            
            // Start orchestration
            String instanceId = client.scheduleNewOrchestrationInstance(
                ORCHESTRATION_NAME,
                new NewOrchestrationInstanceOptions()
                    .setInput(jobData)
                    .setInstanceId(jobId));
            
            logger.info("Started monitoring orchestration with ID: " + instanceId);
            
            // Wait for orchestration to complete while showing updates
            logger.info("Waiting for monitoring to complete...");
            logger.info("Status updates will be displayed as they occur.");
            
            // Create a simple timeout for the demonstration
            Duration maxWaitTime = Duration.ofSeconds(timeout + 20);  // Add a buffer to the timeout
            Duration waitInterval = Duration.ofSeconds(2);
            Duration totalWaitTime = Duration.ZERO;
            
            String lastStatus = null;
            while (totalWaitTime.compareTo(maxWaitTime) < 0) {
                // Get the current orchestration state
                OrchestrationMetadata state = client.getInstanceMetadata(instanceId, true);
                
                if (state != null) {
                    // Display custom status updates if available and different from last update
                    if (state.readCustomStatusAs(JobStatus.class) != null) {
                        JobStatus currentStatus = state.readCustomStatusAs(JobStatus.class);
                        if (!currentStatus.status.equals(lastStatus)) {
                            lastStatus = currentStatus.status;
                            logger.info("Status update: " + lastStatus);
                        }
                    }
                    
                    // Check if the orchestration has completed
                    if (state.getRuntimeStatus() == OrchestrationRuntimeStatus.COMPLETED ||
                        state.getRuntimeStatus() == OrchestrationRuntimeStatus.FAILED ||
                        state.getRuntimeStatus() == OrchestrationRuntimeStatus.TERMINATED) {
                        
                        logger.info("Monitoring completed with status: " + state.getRuntimeStatus());
                        
                        if (state.getRuntimeStatus() == OrchestrationRuntimeStatus.COMPLETED) {
                            JobResult result = state.readOutputAs(JobResult.class);
                            logger.info("Final result: " + result.status);
                            logger.info("Message: " + result.message);
                        } else if (state.getRuntimeStatus() == OrchestrationRuntimeStatus.FAILED) {
                            logger.warning("Monitoring failed: " + state.getFailureDetails());
                        }
                        
                        break;
                    }
                }
                
                // Wait before checking again
                Thread.sleep(waitInterval.toMillis());
                totalWaitTime = totalWaitTime.plus(waitInterval);
            }
            
            if (totalWaitTime.compareTo(maxWaitTime) >= 0) {
                logger.warning("Client timed out waiting for orchestration to complete");
            }
            
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
                        // Get the job data
                        JobData jobData = ctx.getInput(JobData.class);
                        
                        // Set initial status
                        ctx.setCustomStatus(new JobStatus("Starting monitoring..."));
                        
                        // Simulate job monitoring with status updates
                        int pollingCount = 0;
                        while (true) {
                            // Update status
                            ctx.setCustomStatus(new JobStatus("Polling job status (attempt " + (++pollingCount) + ")"));
                            
                            // Simulate some work
                            ctx.createTimer(Duration.ofSeconds(jobData.pollingIntervalSeconds)).await();
                            
                            // Check if job is complete (simulated)
                            if (pollingCount >= 3) {
                                ctx.setCustomStatus(new JobStatus("Job completed successfully"));
                                ctx.complete(new JobResult("COMPLETED", "Job completed after " + pollingCount + " attempts"));
                                break;
                            }
                        }
                    };
                }
            })
            .build();
    }

    // Data classes
    public static class JobData {
        @JsonProperty public final String jobId;
        @JsonProperty public final int pollingIntervalSeconds;
        @JsonProperty public final int timeoutSeconds;

        public JobData(
            @JsonProperty("jobId") String jobId,
            @JsonProperty("pollingIntervalSeconds") int pollingIntervalSeconds,
            @JsonProperty("timeoutSeconds") int timeoutSeconds) {
            this.jobId = jobId;
            this.pollingIntervalSeconds = pollingIntervalSeconds;
            this.timeoutSeconds = timeoutSeconds;
        }
    }

    public static class JobStatus {
        @JsonProperty public final String status;

        public JobStatus(@JsonProperty("status") String status) {
            this.status = status;
        }
    }

    public static class JobResult {
        @JsonProperty public final String status;
        @JsonProperty public final String message;

        public JobResult(
            @JsonProperty("status") String status,
            @JsonProperty("message") String message) {
            this.status = status;
            this.message = message;
        }
    }
} 