// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package io.durabletask.samples;

import com.microsoft.durabletask.*;
import com.microsoft.durabletask.azuremanaged.DurableTaskSchedulerClientExtensions;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;
import java.time.Instant;
import java.util.ArrayList;
import java.util.List;

/**
 * Client — schedules orchestrations in a continuous loop and prints results.
 *
 * <p>Schedules a batch of 3 orchestrations every 30 seconds for 10 minutes.
 * This makes it easy to observe scaling behavior over time.
 */
public final class Client {
    private static final Logger logger = LoggerFactory.getLogger(Client.class);

    public static void main(String[] args) throws Exception {
        String connectionString = ConnectionHelper.getConnectionString();

        logger.info("=== Work Item Filtering Demo — Client ===");
        logger.info("Connection: {}", connectionString);

        // Create the Durable Task client
        DurableTaskClient client = DurableTaskSchedulerClientExtensions
                .createClientBuilder(connectionString)
                .build();

        // Run in a loop: schedule a batch of orchestrations every 30 seconds for 10 minutes
        int orchestrationsPerBatch = 3;
        long intervalSeconds = 30;
        long totalDurationMinutes = 10;
        Instant deadline = Instant.now().plusSeconds(totalDurationMinutes * 60);

        int totalCompleted = 0;
        int totalFailed = 0;
        int batchNumber = 0;

        logger.info("Will schedule {} orchestrations every {}s for {} minutes.",
                orchestrationsPerBatch, intervalSeconds, totalDurationMinutes);
        logger.info("(Make sure the Orchestrator, Validator, and Shipper workers are all running)\n");

        while (Instant.now().isBefore(deadline)) {
            batchNumber++;
            logger.info("--- Batch #{} at {} ---", batchNumber, Instant.now());

            List<String> instanceIds = new ArrayList<>();
            for (int i = 1; i <= orchestrationsPerBatch; i++) {
                String orderId = String.format("ORD-B%03d-%03d", batchNumber, i);
                logger.info("Scheduling orchestration with orderId='{}'...", orderId);

                String instanceId = client.scheduleNewOrchestrationInstance(
                        "OrderProcessingOrchestration",
                        new NewOrchestrationInstanceOptions().setInput(orderId));

                instanceIds.add(instanceId);
                logger.info("  -> Scheduled with InstanceId={}", instanceId);
            }

            // Wait for all orchestrations in this batch to complete
            int batchCompleted = 0;
            int batchFailed = 0;

            for (String id : instanceIds) {
                try {
                    OrchestrationMetadata result = client.waitForInstanceCompletion(
                            id, Duration.ofSeconds(120), true);

                    if (result.getRuntimeStatus() == OrchestrationRuntimeStatus.COMPLETED) {
                        batchCompleted++;
                        logger.info("COMPLETED | InstanceId={} | Output: {}",
                                result.getInstanceId(), result.readOutputAs(String.class));
                    } else {
                        batchFailed++;
                        logger.error("FAILED    | InstanceId={} | Status={}",
                                result.getInstanceId(), result.getRuntimeStatus());
                    }
                } catch (Exception ex) {
                    batchFailed++;
                    logger.error("Error waiting for orchestration {}", id, ex);
                }
            }

            totalCompleted += batchCompleted;
            totalFailed += batchFailed;

            logger.info("Batch #{} results: {} completed, {} failed", batchNumber, batchCompleted, batchFailed);

            // Wait for the next interval (unless we've passed the deadline)
            if (Instant.now().isBefore(deadline)) {
                long remainingSeconds = Duration.between(Instant.now(), deadline).getSeconds();
                long waitSeconds = Math.min(remainingSeconds, intervalSeconds);
                logger.info("Next batch in {}s (deadline in {} min)\n",
                        waitSeconds, remainingSeconds / 60);
                Thread.sleep(waitSeconds * 1000);
            }
        }

        logger.info("\n=== FINAL RESULTS: {} completed, {} failed across {} batches ===",
                totalCompleted, totalFailed, batchNumber);

        // Keep the process alive so Container Apps doesn't mark it as failed
        logger.info("Demo complete. Staying alive — press Ctrl+C to exit.");
        Thread.currentThread().join();
    }
}
