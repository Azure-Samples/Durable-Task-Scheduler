// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package io.durabletask.samples;

import com.microsoft.durabletask.*;
import com.microsoft.durabletask.azuremanaged.DurableTaskSchedulerClientExtensions;
import com.microsoft.durabletask.azuremanaged.DurableTaskSchedulerWorkerExtensions;
import io.opentelemetry.api.trace.Span;
import io.opentelemetry.api.trace.StatusCode;
import io.opentelemetry.api.trace.Tracer;
import io.opentelemetry.context.Context;
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
 *
 * <p>The Java SDK's client automatically propagates W3C trace context
 * (traceparent/tracestate) when scheduling orchestrations. This sample
 * additionally shares the parent span context with activities running in
 * the same process so that all spans appear under a single trace in Jaeger.
 */
final class OpenTelemetryTracingPattern {
    private static final Logger logger = LoggerFactory.getLogger(OpenTelemetryTracingPattern.class);

    // Shared parent context so activity spans become children of the orchestration span.
    // In a distributed setup the SDK would propagate this via W3C trace context headers.
    private static volatile Context orchestrationContext;

    /** Create an activity span as a child of the orchestration span. */
    private static Span startActivitySpan(Tracer tracer, String activityName, int taskId) {
        Context parent = orchestrationContext != null ? orchestrationContext : Context.current();
        return tracer.spanBuilder("activity:" + activityName)
                .setParent(parent)
                .setAttribute("durabletask.task.name", activityName)
                .setAttribute("durabletask.type", "activity")
                .setAttribute("durabletask.task.task_id", taskId)
                .startSpan();
    }

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
                .put("service.name", "DistributedTracingSample")
                .build();

        SdkTracerProvider tracerProvider = SdkTracerProvider.builder()
                .setResource(resource)
                .addSpanProcessor(BatchSpanProcessor.builder(spanExporter).build())
                .build();

        OpenTelemetrySdk openTelemetry = OpenTelemetrySdk.builder()
                .setTracerProvider(tracerProvider)
                .buildAndRegisterGlobal();

        Tracer tracer = openTelemetry.getTracer("Microsoft.DurableTask");
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
                            String validated = ctx.callActivity("ValidateOrder", orderId, String.class).await();
                            String paid = ctx.callActivity("ProcessPayment", validated, String.class).await();
                            String shipped = ctx.callActivity("ShipOrder", paid, String.class).await();
                            String result = ctx.callActivity("SendNotification", shipped, String.class).await();
                            ctx.complete(result);
                        };
                    }
                })
                .addActivity(new TaskActivityFactory() {
                    @Override public String getName() { return "ValidateOrder"; }
                    @Override public TaskActivity create() {
                        return ctx -> {
                            String orderId = ctx.getInput(String.class);
                            Span span = startActivitySpan(tracer, "ValidateOrder", 1);
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
                    @Override public String getName() { return "ProcessPayment"; }
                    @Override public TaskActivity create() {
                        return ctx -> {
                            String input = ctx.getInput(String.class);
                            Span span = startActivitySpan(tracer, "ProcessPayment", 2);
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
                    @Override public String getName() { return "ShipOrder"; }
                    @Override public TaskActivity create() {
                        return ctx -> {
                            String input = ctx.getInput(String.class);
                            Span span = startActivitySpan(tracer, "ShipOrder", 3);
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
                    @Override public String getName() { return "SendNotification"; }
                    @Override public TaskActivity create() {
                        return ctx -> {
                            String input = ctx.getInput(String.class);
                            Span span = startActivitySpan(tracer, "SendNotification", 4);
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

        // Create a parent span for the orchestration. The SDK automatically propagates
        // W3C trace context (traceparent/tracestate) when scheduling orchestrations.
        Span orchestrationSpan = tracer.spanBuilder("create_orchestration:OrderProcessingOrchestration")
                .setAttribute("durabletask.task.name", "OrderProcessingOrchestration")
                .setAttribute("durabletask.type", "orchestration")
                .startSpan();

        // Share the orchestration context with the worker thread so activity spans
        // become children of this orchestration span in the trace.
        orchestrationContext = Context.current().with(orchestrationSpan);

        String instanceId;
        try (Scope scope = orchestrationSpan.makeCurrent()) {
            logger.info("Scheduling order processing orchestration...");
            instanceId = client.scheduleNewOrchestrationInstance(
                    "OrderProcessingOrchestration",
                    new NewOrchestrationInstanceOptions().setInput("Order-12345"));
            orchestrationSpan.setAttribute("durabletask.task.instance_id", instanceId);
            logger.info("Started orchestration: {}", instanceId);
        }

        // Wait for completion
        logger.info("Waiting for completion...");
        OrchestrationMetadata result = client.waitForInstanceCompletion(
                instanceId, Duration.ofSeconds(60), true);
        orchestrationSpan.end();

        logger.info("Status: {}", result.getRuntimeStatus());
        logger.info("Result: {}", result.readOutputAs(String.class));
        if (result.getFailureDetails() != null
                && result.getFailureDetails().getErrorMessage() != null
                && !result.getFailureDetails().getErrorMessage().isEmpty()) {
            logger.error("Failure: {} - {}", result.getFailureDetails().getErrorType(),
                    result.getFailureDetails().getErrorMessage());
        }
        logger.info("");
        logger.info("View traces in Jaeger UI: http://localhost:16686");
        logger.info("  Search for service: DistributedTracingSample");
        logger.info("View orchestration in DTS Dashboard: http://localhost:8082");

        // Flush traces and shut down
        tracerProvider.forceFlush();
        Thread.sleep(2000);
        tracerProvider.shutdown();
        worker.stop();
        System.exit(0);
    }
}
