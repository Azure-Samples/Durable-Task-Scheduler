// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package io.durabletask.samples;

import com.microsoft.durabletask.*;
import com.microsoft.durabletask.azuremanaged.DurableTaskSchedulerClientExtensions;
import com.microsoft.durabletask.azuremanaged.DurableTaskSchedulerWorkerExtensions;
import io.opentelemetry.api.trace.Span;
import io.opentelemetry.api.trace.StatusCode;
import io.opentelemetry.api.trace.Tracer;
import io.opentelemetry.context.Scope;
import io.opentelemetry.exporter.otlp.trace.OtlpGrpcSpanExporter;
import io.opentelemetry.sdk.OpenTelemetrySdk;
import io.opentelemetry.sdk.resources.Resource;
import io.opentelemetry.sdk.trace.SdkTracerProvider;
import io.opentelemetry.sdk.trace.export.BatchSpanProcessor;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.time.Duration;
import java.util.concurrent.TimeoutException;

/**
 * Demonstrates OpenTelemetry distributed tracing with the Durable Task SDK.
 * Traces are exported to Jaeger via OTLP/gRPC for visualization.
 */
final class OpenTelemetryTracingPattern {
    private static final Logger logger = LoggerFactory.getLogger(OpenTelemetryTracingPattern.class);

    public static void main(String[] args) throws IOException, InterruptedException, TimeoutException {
        // Configure OpenTelemetry
        String otlpEndpoint = System.getenv("OTEL_EXPORTER_OTLP_ENDPOINT");
        if (otlpEndpoint == null) {
            otlpEndpoint = "http://localhost:4317";
        }

        OtlpGrpcSpanExporter spanExporter = OtlpGrpcSpanExporter.builder()
                .setEndpoint(otlpEndpoint)
                .build();

        Resource resource = Resource.builder()
                .put("service.name", "durable-worker")
                .build();

        SdkTracerProvider tracerProvider = SdkTracerProvider.builder()
                .setResource(resource)
                .addSpanProcessor(BatchSpanProcessor.builder(spanExporter).build())
                .build();

        OpenTelemetrySdk openTelemetry = OpenTelemetrySdk.builder()
                .setTracerProvider(tracerProvider)
                .buildAndRegisterGlobal();

        Tracer tracer = openTelemetry.getTracer("durable-worker");
        logger.info("OpenTelemetry configured with OTLP exporter at {}", otlpEndpoint);

        // Build connection string
        String connectionString = System.getenv("DURABLE_TASK_CONNECTION_STRING");
        if (connectionString == null) {
            String endpoint = System.getenv("ENDPOINT");
            String taskHub = System.getenv("TASKHUB");

            if (endpoint == null) endpoint = "http://localhost:8080";
            if (taskHub == null) taskHub = "default";

            String authType = endpoint.startsWith("http://localhost") ? "None" : "DefaultAzure";
            connectionString = String.format("Endpoint=%s;TaskHub=%s;Authentication=%s",
                    endpoint, taskHub, authType);
        }
        logger.info("Using connection string: {}", connectionString);

        // Create worker with orchestration and activities
        DurableTaskGrpcWorker worker = DurableTaskSchedulerWorkerExtensions
                .createWorkerBuilder(connectionString)
                .addOrchestration(new TaskOrchestrationFactory() {
                    @Override
                    public String getName() { return "OrderProcessingOrchestration"; }

                    @Override
                    public TaskOrchestration create() {
                        return ctx -> {
                            String orderId = ctx.getInput(String.class);

                            // Step 1: Validate the order
                            String validated = ctx.callActivity(
                                    "ValidateOrder", orderId, String.class).await();

                            // Step 2: Process payment
                            String paid = ctx.callActivity(
                                    "ProcessPayment", validated, String.class).await();

                            // Step 3: Ship order
                            String shipped = ctx.callActivity(
                                    "ShipOrder", paid, String.class).await();

                            // Step 4: Send notification
                            String result = ctx.callActivity(
                                    "SendNotification", shipped, String.class).await();

                            ctx.complete(result);
                        };
                    }
                })
                .addActivity(new TaskActivityFactory() {
                    @Override
                    public String getName() { return "ValidateOrder"; }

                    @Override
                    public TaskActivity create() {
                        return ctx -> {
                            String orderId = ctx.getInput(String.class);
                            Span span = tracer.spanBuilder("ValidateOrder").startSpan();
                            try (Scope scope = span.makeCurrent()) {
                                logger.info("[ValidateOrder] Validating order: {}", orderId);
                                Thread.sleep(100);
                                return "Validated(" + orderId + ")";
                            } catch (InterruptedException e) {
                                span.setStatus(StatusCode.ERROR);
                                Thread.currentThread().interrupt();
                                throw new RuntimeException(e);
                            } finally {
                                span.end();
                            }
                        };
                    }
                })
                .addActivity(new TaskActivityFactory() {
                    @Override
                    public String getName() { return "ProcessPayment"; }

                    @Override
                    public TaskActivity create() {
                        return ctx -> {
                            String input = ctx.getInput(String.class);
                            Span span = tracer.spanBuilder("ProcessPayment").startSpan();
                            try (Scope scope = span.makeCurrent()) {
                                logger.info("[ProcessPayment] Processing payment for: {}", input);
                                Thread.sleep(200);
                                return "Paid(" + input + ")";
                            } catch (InterruptedException e) {
                                span.setStatus(StatusCode.ERROR);
                                Thread.currentThread().interrupt();
                                throw new RuntimeException(e);
                            } finally {
                                span.end();
                            }
                        };
                    }
                })
                .addActivity(new TaskActivityFactory() {
                    @Override
                    public String getName() { return "ShipOrder"; }

                    @Override
                    public TaskActivity create() {
                        return ctx -> {
                            String input = ctx.getInput(String.class);
                            Span span = tracer.spanBuilder("ShipOrder").startSpan();
                            try (Scope scope = span.makeCurrent()) {
                                logger.info("[ShipOrder] Shipping: {}", input);
                                Thread.sleep(150);
                                return "Shipped(" + input + ")";
                            } catch (InterruptedException e) {
                                span.setStatus(StatusCode.ERROR);
                                Thread.currentThread().interrupt();
                                throw new RuntimeException(e);
                            } finally {
                                span.end();
                            }
                        };
                    }
                })
                .addActivity(new TaskActivityFactory() {
                    @Override
                    public String getName() { return "SendNotification"; }

                    @Override
                    public TaskActivity create() {
                        return ctx -> {
                            String input = ctx.getInput(String.class);
                            Span span = tracer.spanBuilder("SendNotification").startSpan();
                            try (Scope scope = span.makeCurrent()) {
                                logger.info("[SendNotification] Notifying customer: {}", input);
                                Thread.sleep(50);
                                return "Notified(" + input + ")";
                            } catch (InterruptedException e) {
                                span.setStatus(StatusCode.ERROR);
                                Thread.currentThread().interrupt();
                                throw new RuntimeException(e);
                            } finally {
                                span.end();
                            }
                        };
                    }
                })
                .build();

        // Start the worker
        worker.start();
        Thread.sleep(5000);
        logger.info("Worker started with OpenTelemetry tracing.");

        // Create client and schedule orchestration
        DurableTaskClient client = DurableTaskSchedulerClientExtensions
                .createClientBuilder(connectionString).build();

        // Create a parent span for the orchestration - the SDK automatically propagates
        // W3C trace context (traceparent/tracestate) when scheduling orchestrations
        Span orchestrationSpan = tracer.spanBuilder("OrderProcessingOrchestration").startSpan();
        String instanceId;
        try (Scope scope = orchestrationSpan.makeCurrent()) {
            logger.info("Scheduling order processing orchestration...");
            instanceId = client.scheduleNewOrchestrationInstance(
                    "OrderProcessingOrchestration",
                    new NewOrchestrationInstanceOptions().setInput("Order-12345"));
            logger.info("Started orchestration: {}", instanceId);
        }

        // Wait for completion
        logger.info("Waiting for completion...");
        OrchestrationMetadata result = client.waitForInstanceCompletion(
                instanceId, Duration.ofSeconds(60), true);
        orchestrationSpan.end();

        logger.info("Status: {}", result.getRuntimeStatus());
        logger.info("Result: {}", result.readOutputAs(String.class));
        logger.info("");
        logger.info("View traces in Jaeger UI: http://localhost:16686");
        logger.info("View orchestration in DTS Dashboard: http://localhost:8082");

        // Shutdown
        tracerProvider.shutdown();
        worker.stop();
        System.exit(0);
    }
}
