// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package io.durabletask.samples;

import com.microsoft.durabletask.*;
import com.microsoft.durabletask.azuremanaged.DurableTaskSchedulerClientExtensions;
import com.microsoft.durabletask.azuremanaged.DurableTaskSchedulerWorkerExtensions;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.web.bind.annotation.*;
import org.springframework.context.ConfigurableApplicationContext;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

/**
 * Sample Spring Boot application demonstrating Azure-managed Durable Task integration.
 * This sample shows how to:
 * 1. Configure Azure-managed Durable Task with Spring Boot
 * 2. Create orchestrations and activities
 * 3. Handle REST API endpoints for order processing
 */
@SpringBootApplication
public class WebApi {

    public static void main(String[] args) {
        ConfigurableApplicationContext context = SpringApplication.run(WebApi.class, args);

        // Get the worker bean and start it
        DurableTaskGrpcWorker worker = context.getBean(DurableTaskGrpcWorker.class);
        worker.start();
    }

    @Configuration
    static class DurableTaskConfig {
        @Bean
        public DurableTaskGrpcWorker durableTaskWorker() {

            // Create worker using Azure-managed extensions
            // use system env variable DURABLE_TASK_CONNECTION_STRING or default to local development
            String connectionString = System.getenv("DURABLE_TASK_CONNECTION_STRING");
            if (connectionString == null) {
                connectionString = "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";
            }
            DurableTaskGrpcWorkerBuilder workerBuilder = DurableTaskSchedulerWorkerExtensions.createWorkerBuilder(
                connectionString
            );

            // Add orchestrations using the factory pattern
            workerBuilder.addOrchestration(new TaskOrchestrationFactory() {
                @Override
                public String getName() { return "ProcessOrderOrchestration"; }

                @Override
                public TaskOrchestration create() {
                    return ctx -> {
                        // Get the order input as JSON string
                        String orderJson = ctx.getInput(String.class);

                        // Process the order through multiple activities
                        boolean isValid = ctx.callActivity("ValidateOrder", orderJson, Boolean.class).await();
                        if (!isValid) {
                            ctx.complete("{\"status\": \"FAILED\", \"message\": \"Order validation failed\"}");
                            return;
                        }

                        // Process payment
                        String paymentResult = ctx.callActivity("ProcessPayment", orderJson, String.class).await();
                        if (!paymentResult.contains("\"success\":true")) {
                            ctx.complete("{\"status\": \"FAILED\", \"message\": \"Payment processing failed\"}");
                            return;
                        }

                        // Ship order
                        String shipmentResult = ctx.callActivity("ShipOrder", orderJson, String.class).await();

                        // Return the final result
                        ctx.complete("{\"status\": \"SUCCESS\", " +
                                   "\"payment\": " + paymentResult + ", " +
                                   "\"shipment\": " + shipmentResult + "}");
                    };
                }
            });

            // Add activities using the factory pattern
            workerBuilder.addActivity(new TaskActivityFactory() {
                @Override
                public String getName() { return "ValidateOrder"; }

                @Override
                public TaskActivity create() {
                    return ctx -> {
                        String orderJson = ctx.getInput(String.class);
                        // Simple validation - check if order contains amount and it's greater than 0
                        return orderJson.contains("\"amount\"") && !orderJson.contains("\"amount\":0");
                    };
                }
            });

            workerBuilder.addActivity(new TaskActivityFactory() {
                @Override
                public String getName() { return "ProcessPayment"; }

                @Override
                public TaskActivity create() {
                    return ctx -> {
                        // Simulate payment processing
                        sleep(1000); // Simulate processing time
                        return "{\"success\":true, \"transactionId\":\"TXN" + System.currentTimeMillis() + "\"}";
                    };
                }
            });

            workerBuilder.addActivity(new TaskActivityFactory() {
                @Override
                public String getName() { return "ShipOrder"; }

                @Override
                public TaskActivity create() {
                    return ctx -> {
                        // Simulate shipping process
                        sleep(1000); // Simulate processing time
                        return "{\"trackingNumber\":\"TRACK" + System.currentTimeMillis() + "\"}";
                    };
                }
            });

            return workerBuilder.build();
        }

        @Bean
        public DurableTaskClient durableTaskClient() {
            
            String connectionString = System.getenv("DURABLE_TASK_CONNECTION_STRING");
            if (connectionString == null) {
                connectionString = "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";
            }

            // Create client using Azure-managed extensions
            return DurableTaskSchedulerClientExtensions.createClientBuilder(connectionString).build();
        }
    }

    private static void sleep(int millis) {
        try {
            Thread.sleep(millis);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        }
    }
}

/**
 * REST Controller for handling order-related operations.
 */
@RestController
@RequestMapping("/api/orders")
class OrderController {

    private final DurableTaskClient client;

    public OrderController(DurableTaskClient client) {
        this.client = client;
    }

    @PostMapping
    public String createOrder(@RequestBody String orderJson) throws Exception {
        String instanceId = client.scheduleNewOrchestrationInstance(
            "ProcessOrderOrchestration",
            orderJson
        );

        // Return the instance ID immediately without waiting for completion
        return "{\"instanceId\": \"" + instanceId + "\"}";
    }

    @GetMapping("/{instanceId}")
    public String getOrder(@PathVariable String instanceId) throws Exception {
        OrchestrationMetadata metadata = client.getInstanceMetadata(instanceId, true);
        if (metadata == null) {
            return "{\"error\": \"Order not found\"}";
        }
        return metadata.readOutputAs(String.class);
    }
}