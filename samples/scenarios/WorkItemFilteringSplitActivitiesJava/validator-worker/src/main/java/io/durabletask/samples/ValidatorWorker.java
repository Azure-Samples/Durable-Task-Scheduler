// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package io.durabletask.samples;

import com.microsoft.durabletask.*;
import com.microsoft.durabletask.azuremanaged.DurableTaskSchedulerWorkerExtensions;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;

/**
 * Validator Worker — runs only the {@code ValidateOrder} activity.
 *
 * <p>This worker registers a single activity and enables auto-generated work item filters.
 * DTS will route only {@code ValidateOrder} activity work items to this worker.
 * It will never receive orchestration or other activity work items.
 */
public final class ValidatorWorker {
    private static final Logger logger = LoggerFactory.getLogger(ValidatorWorker.class);

    public static void main(String[] args) throws IOException, InterruptedException {
        String connectionString = ConnectionHelper.getConnectionString();

        logger.info("[Validator] Connection: {}", connectionString);
        logger.info("[Validator] This worker registers ONLY the ValidateOrder activity.");

        // Build the worker with only the ValidateOrder activity registered.
        // useWorkItemFilters() auto-generates filters from the registry, so this worker
        // will ONLY receive ValidateOrder activity work items.
        DurableTaskGrpcWorker worker = DurableTaskSchedulerWorkerExtensions
                .createWorkerBuilder(connectionString)
                .addActivity(new TaskActivityFactory() {
                    @Override
                    public String getName() {
                        return "ValidateOrder";
                    }

                    @Override
                    public TaskActivity create() {
                        return ctx -> {
                            String orderId = ctx.getInput(String.class);

                            logger.info("[Validator] Activity | Name=ValidateOrder | Validating order '{}'...", orderId);

                            // Simulate validation
                            String result = "Order " + orderId + " is valid";

                            logger.info("[Validator] Activity | Name=ValidateOrder | Result: {}", result);

                            return result;
                        };
                    }
                })
                .useWorkItemFilters() // auto-generate from registered tasks
                .build();

        logger.info("[Validator] Starting... waiting for ValidateOrder activity work items only.");
        worker.start();

        // Keep the process alive
        logger.info("[Validator] Worker started. Press Ctrl+C to exit.");
        Thread.currentThread().join();
    }
}
