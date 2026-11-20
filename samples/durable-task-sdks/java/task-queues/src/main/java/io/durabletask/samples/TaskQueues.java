// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package io.durabletask.samples;

import com.microsoft.durabletask.DurableTaskClient;
import com.microsoft.durabletask.DurableTaskGrpcClientBuilder;
import com.microsoft.durabletask.DurableTaskGrpcWorker;
import com.microsoft.durabletask.DurableTaskGrpcWorkerBuilder;
import com.microsoft.durabletask.NewOrchestrationInstanceOptions;
import com.microsoft.durabletask.OrchestrationMetadata;
import com.microsoft.durabletask.OrchestrationRuntimeStatus;
import com.microsoft.durabletask.TaskActivity;
import com.microsoft.durabletask.TaskActivityFactory;
import com.microsoft.durabletask.TaskOrchestration;
import com.microsoft.durabletask.TaskOrchestrationFactory;
import com.microsoft.durabletask.azuremanaged.DurableTaskSchedulerWorkerOptions;
import io.grpc.CallOptions;
import io.grpc.Channel;
import io.grpc.ClientCall;
import io.grpc.ClientInterceptor;
import io.grpc.ClientInterceptors;
import io.grpc.Deadline;
import io.grpc.ManagedChannel;
import io.grpc.MethodDescriptor;

import java.time.Duration;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;

public class TaskQueues {
    public static void main(String[] args) throws Exception {
        String connectionString = System.getenv("DURABLE_TASK_CONNECTION_STRING");
        if (connectionString == null || connectionString.isBlank()) {
            throw new IllegalArgumentException("Set DURABLE_TASK_CONNECTION_STRING to your Azure scheduler.");
        }
        int count = args.length == 0 ? 10 : Integer.parseInt(args[0]);
        if (count < 1 || count > 100) {
            throw new IllegalArgumentException("Order count must be between 1 and 100.");
        }

        // The managed SDK supplies an external channel, so this sample owns its lifetime.
        ManagedChannel channel = (ManagedChannel) DurableTaskSchedulerWorkerOptions
                .fromConnectionString(connectionString).createGrpcChannel();
        Deadline batchDeadline = Deadline.after(3, TimeUnit.MINUTES);
        Channel clientChannel = ClientInterceptors.intercept(channel, new ClientInterceptor() {
            @Override
            public <ReqT, RespT> ClientCall<ReqT, RespT> interceptCall(
                    MethodDescriptor<ReqT, RespT> method, CallOptions options, Channel next) {
                Deadline deadline = options.getDeadline() == null ? batchDeadline
                        : batchDeadline.minimum(options.getDeadline());
                return next.newCall(method, options.withDeadline(deadline));
            }
        });
        Thread listener = null;
        try (DurableTaskGrpcWorker worker = createWorker(channel);
             DurableTaskClient client = new DurableTaskGrpcClientBuilder().grpcChannel(clientChannel).build()) {
            listener = new Thread(worker::startAndBlock, "task-queues-listener");
            listener.start();
            List<String> instances = new ArrayList<>();
            for (int i = 1; i <= count; i++) {
                String instanceId = client.scheduleNewOrchestrationInstance(
                        "ProcessOrder",
                        new NewOrchestrationInstanceOptions()
                                .setInput("order-" + i)
                                .setTags(Map.of("DTS_TaskQueue", "orders")));
                instances.add(instanceId);
                System.out.println("Started " + instanceId + " with DTS_TaskQueue=orders");
            }
            for (String instanceId : instances) {
                long remaining = batchDeadline.timeRemaining(TimeUnit.NANOSECONDS);
                if (remaining <= 0) {
                    throw new TimeoutException("Batch did not complete within three minutes.");
                }
                OrchestrationMetadata state = client.waitForInstanceCompletion(
                        instanceId, Duration.ofNanos(remaining), true);
                if (state == null || !state.isInstanceFound()) {
                    throw new IllegalStateException("Instance not found: " + instanceId);
                }
                if (state.getRuntimeStatus() != OrchestrationRuntimeStatus.COMPLETED) {
                    throw new IllegalStateException(instanceId + ": " + state.getRuntimeStatus()
                            + ", failure: " + state.getFailureDetails());
                }
                System.out.println(state.readOutputAs(String.class));
            }
            System.out.println("Completed " + count + " orders.");
        } finally {
            if (listener != null) {
                listener.interrupt();
            }
            channel.shutdownNow();
            channel.awaitTermination(5, TimeUnit.SECONDS);
            if (listener != null) {
                listener.join(5000);
                if (listener.isAlive()) {
                    throw new IllegalStateException("Worker listener did not stop.");
                }
            }
        }
    }

    private static DurableTaskGrpcWorker createWorker(ManagedChannel channel) {
        // Do not call useWorkItemFilters(), even with wildcard filters.
        return new DurableTaskGrpcWorkerBuilder().grpcChannel(channel)
                .addOrchestration(orchestration("ProcessOrder", ctx -> {
                    String order = ctx.getInput(String.class);
                    ctx.callActivity("ValidateOrder", order, String.class).await();
                    String receipt = ctx.callSubOrchestrator("FulfillOrder", order, String.class).await();
                    ctx.complete(receipt);
                }))
                .addOrchestration(orchestration("FulfillOrder", ctx -> {
                    String order = ctx.getInput(String.class);
                    ctx.callActivity("CallPartnerApi", order, String.class).await();
                    ctx.complete(ctx.callActivity("ShipOrder", order, String.class).await());
                }))
                .addActivity(activity("ValidateOrder", ctx -> {
                    String order = ctx.getInput(String.class);
                    System.out.println("Validated " + order);
                    return order;
                }))
                .addActivity(activity("CallPartnerApi", ctx -> {
                    String order = ctx.getInput(String.class);
                    System.out.println("Simulated partner call for " + order);
                    return order;
                }))
                .addActivity(activity("ShipOrder", ctx -> "Shipped " + ctx.getInput(String.class)))
                .build();
    }

    private static TaskOrchestrationFactory orchestration(String name, TaskOrchestration task) {
        return new TaskOrchestrationFactory() {
            public String getName() { return name; }
            public TaskOrchestration create() { return task; }
        };
    }

    private static TaskActivityFactory activity(String name, TaskActivity task) {
        return new TaskActivityFactory() {
            public String getName() { return name; }
            public TaskActivity create() { return task; }
        };
    }
}
