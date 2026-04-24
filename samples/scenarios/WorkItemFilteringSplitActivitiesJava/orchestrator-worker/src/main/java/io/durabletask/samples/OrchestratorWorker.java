// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package io.durabletask.samples;

import com.microsoft.durabletask.*;
import com.microsoft.durabletask.azuremanaged.DurableTaskSchedulerWorkerExtensions;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;

/**
 * Orchestrator Worker — runs only the {@code OrderProcessingOrchestration}.
 *
 * <p>This worker registers a single orchestration and enables auto-generated work item filters.
 * DTS will route only orchestration work items for {@code OrderProcessingOrchestration} to this worker.
 * It will never receive activity work items.
 */
public final class OrchestratorWorker {
    private static final Logger logger = LoggerFactory.getLogger(OrchestratorWorker.class);

    public static void main(String[] args) throws IOException, InterruptedException {
        String connectionString = ConnectionHelper.getConnectionString();

        logger.info("[Orchestrator] Connection: {}", connectionString);
        logger.info("[Orchestrator] This worker registers ONLY the orchestration. No activities.");

        // Build the worker with only the orchestration registered.
        // useWorkItemFilters() auto-generates filters from the registry, so this worker
        // will ONLY receive orchestration work items — never activity work items.
        DurableTaskGrpcWorker worker = DurableTaskSchedulerWorkerExtensions
                .createWorkerBuilder(connectionString)
                .addOrchestration(new TaskOrchestrationFactory() {
                    @Override
                    public String getName() {
                        return "OrderProcessingOrchestration";
                    }

                    @Override
                    public TaskOrchestration create() {
                        return ctx -> {
                            String orderId = ctx.getInput(String.class);

                            logger.info("[Orchestrator] Orchestration | Name=OrderProcessingOrchestration | InstanceId={} | Processing order '{}'",
                                    ctx.getInstanceId(), orderId);

                            // Step 1: Validate the order (routed to Validator Worker)
                            logger.info("[Orchestrator] Orchestration | InstanceId={} | Dispatching ValidateOrder to Validator Worker...",
                                    ctx.getInstanceId());
                            String validationResult = ctx.callActivity("ValidateOrder", orderId, String.class).await();

                            // Step 2: Ship the order (routed to Shipper Worker)
                            logger.info("[Orchestrator] Orchestration | InstanceId={} | Dispatching ShipOrder to Shipper Worker...",
                                    ctx.getInstanceId());
                            String shippingResult = ctx.callActivity("ShipOrder", orderId, String.class).await();

                            String combined = String.format("Order '%s' => Validation: [%s], Shipping: [%s]",
                                    orderId, validationResult, shippingResult);

                            logger.info("[Orchestrator] Orchestration | InstanceId={} | Completed: {}",
                                    ctx.getInstanceId(), combined);

                            ctx.complete(combined);
                        };
                    }
                })
                .useWorkItemFilters() // auto-generate from registered tasks
                .build();

        logger.info("[Orchestrator] Starting... waiting for orchestration work items only.");
        worker.start();

        // Keep the process alive
        logger.info("[Orchestrator] Worker started. Press Ctrl+C to exit.");
        Thread.currentThread().join();
    }
}
