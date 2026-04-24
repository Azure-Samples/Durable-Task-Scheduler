// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package io.durabletask.samples;

import com.microsoft.durabletask.*;
import com.microsoft.durabletask.azuremanaged.DurableTaskSchedulerWorkerExtensions;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.util.concurrent.ThreadLocalRandom;

/**
 * Shipper Worker — runs only the {@code ShipOrder} activity.
 *
 * <p>This worker registers a single activity and enables auto-generated work item filters.
 * DTS will route only {@code ShipOrder} activity work items to this worker.
 * It will never receive orchestration or other activity work items.
 */
public final class ShipperWorker {
    private static final Logger logger = LoggerFactory.getLogger(ShipperWorker.class);

    public static void main(String[] args) throws IOException, InterruptedException {
        String connectionString = ConnectionHelper.getConnectionString();

        logger.info("[Shipper] Connection: {}", connectionString);
        logger.info("[Shipper] This worker registers ONLY the ShipOrder activity.");

        // Build the worker with only the ShipOrder activity registered.
        // useWorkItemFilters() auto-generates filters from the registry, so this worker
        // will ONLY receive ShipOrder activity work items.
        DurableTaskGrpcWorker worker = DurableTaskSchedulerWorkerExtensions
                .createWorkerBuilder(connectionString)
                .addActivity(new TaskActivityFactory() {
                    @Override
                    public String getName() {
                        return "ShipOrder";
                    }

                    @Override
                    public TaskActivity create() {
                        return ctx -> {
                            String orderId = ctx.getInput(String.class);

                            logger.info("[Shipper] Activity | Name=ShipOrder | Shipping order '{}'...", orderId);

                            // Simulate shipping
                            String trackingNumber = "TRACK-" + orderId + "-" + ThreadLocalRandom.current().nextInt(1000, 9999);
                            String result = "Shipped with tracking " + trackingNumber;

                            logger.info("[Shipper] Activity | Name=ShipOrder | Result: {}", result);

                            return result;
                        };
                    }
                })
                .useWorkItemFilters() // auto-generate from registered tasks
                .build();

        logger.info("[Shipper] Starting... waiting for ShipOrder activity work items only.");
        worker.start();

        // Keep the process alive
        logger.info("[Shipper] Worker started. Press Ctrl+C to exit.");
        Thread.currentThread().join();
    }
}
