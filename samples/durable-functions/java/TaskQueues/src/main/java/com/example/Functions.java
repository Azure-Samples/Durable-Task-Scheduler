// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package com.example;

import com.microsoft.azure.functions.*;
import com.microsoft.azure.functions.annotation.*;
import com.microsoft.durabletask.*;
import com.microsoft.durabletask.azurefunctions.*;

import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Optional;

public class Functions {
    private static final DataConverter DATA_CONVERTER = new JacksonDataConverter();

    @FunctionName("StartOrder")
    public HttpResponseMessage startOrder(
            @HttpTrigger(name = "req", methods = {HttpMethod.POST}, route = "orders",
                    authLevel = AuthorizationLevel.FUNCTION) HttpRequestMessage<Optional<String>> request,
            @DurableClientInput(name = "durableContext") DurableClientContext durableContext) {
        String orderId = request.getBody().orElse("").trim();
        if (!orderId.matches("[A-Za-z0-9-]{1,64}")) {
            return request.createResponseBuilder(HttpStatus.BAD_REQUEST)
                    .body("Send an order ID with 1 to 64 letters, digits, or hyphens as text/plain.").build();
        }
        // The Functions extension caches this client across invocations.
        DurableTaskClient client = durableContext.getClient();
        String instanceId = client.scheduleNewOrchestrationInstance("ProcessOrder",
                new NewOrchestrationInstanceOptions()
                        .setInput(orderId)
                        .setTags(Map.of("DTS_TaskQueue", "orders")));
        return durableContext.createCheckStatusResponse(request, instanceId);
    }

    @FunctionName("GetOrderStatus")
    public HttpResponseMessage getOrderStatus(
            @HttpTrigger(name = "req", methods = {HttpMethod.GET}, route = "orders/{instanceId}",
                    authLevel = AuthorizationLevel.FUNCTION) HttpRequestMessage<Void> request,
            @BindingName("instanceId") String instanceId,
            @DurableClientInput(name = "durableContext") DurableClientContext durableContext) {
        DurableTaskClient client = durableContext.getClient();
        OrchestrationMetadata state = client.getInstanceMetadata(instanceId, true);
        if (state == null || !state.isInstanceFound()) {
            return request.createResponseBuilder(HttpStatus.NOT_FOUND).body("Instance not found.").build();
        }
        Map<String, Object> body = new LinkedHashMap<>();
        body.put("instanceId", instanceId);
        body.put("runtimeStatus", state.getRuntimeStatus().name());
        HttpStatus status = HttpStatus.ACCEPTED;
        if (state.getRuntimeStatus() == OrchestrationRuntimeStatus.COMPLETED) {
            status = HttpStatus.OK;
            body.put("output", state.readOutputAs(String.class));
        } else if (state.getRuntimeStatus() == OrchestrationRuntimeStatus.FAILED
                || state.getRuntimeStatus() == OrchestrationRuntimeStatus.TERMINATED) {
            status = HttpStatus.INTERNAL_SERVER_ERROR;
            body.put("failure", state.getFailureDetails() == null
                    ? state.getRuntimeStatus().name() : state.getFailureDetails().toString());
        }
        return request.createResponseBuilder(status)
                .header("Content-Type", "application/json").body(body).build();
    }

    @FunctionName("ProcessOrder")
    public String processOrder(@DurableOrchestrationTrigger(name = "ctx") TaskOrchestrationContext ctx) {
        String orderId = ctx.getInput(String.class);
        ctx.callActivity("ValidateOrder", orderId, String.class).await();
        return ctx.callSubOrchestrator("FulfillOrder", orderId, String.class).await();
    }

    @FunctionName("FulfillOrder")
    public String fulfillOrder(@DurableOrchestrationTrigger(name = "ctx") TaskOrchestrationContext ctx) {
        String orderId = ctx.getInput(String.class);
        ctx.callActivity("CallPartnerApi", orderId, String.class).await();
        return ctx.callActivity("ShipOrder", orderId, String.class).await();
    }

    @FunctionName("ValidateOrder")
    public String validateOrder(@DurableActivityTrigger(name = "orderId") String input,
                                ExecutionContext context) {
        String orderId = orderIdFromActivityInput(input);
        context.getLogger().info("Validated " + orderId);
        return orderId;
    }

    @FunctionName("CallPartnerApi")
    public String callPartnerApi(@DurableActivityTrigger(name = "orderId") String input,
                                 ExecutionContext context) {
        String orderId = orderIdFromActivityInput(input);
        context.getLogger().info("Simulated partner call for " + orderId);
        return orderId;
    }

    @FunctionName("ShipOrder")
    public String shipOrder(@DurableActivityTrigger(name = "orderId") String input) {
        return "Shipped " + orderIdFromActivityInput(input);
    }

    // String activity trigger inputs arrive as serialized JSON.
    private static String orderIdFromActivityInput(String input) {
        return DATA_CONVERTER.deserialize(input, String.class);
    }
}
