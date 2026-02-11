package com.example;

import com.microsoft.azure.functions.*;
import com.microsoft.azure.functions.annotation.*;
import com.microsoft.durabletask.*;
import com.microsoft.durabletask.azurefunctions.*;

import java.util.ArrayList;
import java.util.List;

public class Functions {

    /**
     * HTTP trigger that starts the function chaining orchestration.
     */
    @FunctionName("StartChaining")
    public HttpResponseMessage startChaining(
            @HttpTrigger(name = "req", methods = {HttpMethod.POST}, authLevel = AuthorizationLevel.ANONYMOUS)
            HttpRequestMessage<Void> request,
            @DurableClientInput(name = "durableContext") DurableClientContext durableContext) {

        DurableTaskClient client = durableContext.getClient();
        String instanceId = client.scheduleNewOrchestrationInstance("ChainingOrchestration");
        return durableContext.createCheckStatusResponse(request, instanceId);
    }

    /**
     * Function chaining orchestration: calls activities sequentially.
     */
    @FunctionName("ChainingOrchestration")
    public String chainingOrchestration(
            @DurableOrchestrationTrigger(name = "ctx") TaskOrchestrationContext ctx) {

        String result = "";
        result += ctx.callActivity("SayHello", "Tokyo", String.class).await();
        result += " " + ctx.callActivity("SayHello", "Seattle", String.class).await();
        result += " " + ctx.callActivity("SayHello", "London", String.class).await();
        return result;
    }

    /**
     * HTTP trigger that starts the fan-out/fan-in orchestration.
     */
    @FunctionName("StartFanOutFanIn")
    public HttpResponseMessage startFanOutFanIn(
            @HttpTrigger(name = "req", methods = {HttpMethod.POST}, authLevel = AuthorizationLevel.ANONYMOUS)
            HttpRequestMessage<Void> request,
            @DurableClientInput(name = "durableContext") DurableClientContext durableContext) {

        DurableTaskClient client = durableContext.getClient();
        String instanceId = client.scheduleNewOrchestrationInstance("FanOutFanInOrchestration");
        return durableContext.createCheckStatusResponse(request, instanceId);
    }

    /**
     * Fan-out/fan-in orchestration: calls activities in parallel.
     */
    @FunctionName("FanOutFanInOrchestration")
    public List<String> fanOutFanInOrchestration(
            @DurableOrchestrationTrigger(name = "ctx") TaskOrchestrationContext ctx) {

        String[] cities = {"Tokyo", "Seattle", "London", "Paris", "Berlin"};
        List<Task<String>> parallelTasks = new ArrayList<>();

        // Fan-out: schedule all activities in parallel
        for (String city : cities) {
            parallelTasks.add(ctx.callActivity("SayHello", city, String.class));
        }

        // Fan-in: wait for all to complete
        List<String> results = new ArrayList<>();
        for (Task<String> task : parallelTasks) {
            results.add(task.await());
        }

        return results;
    }

    /**
     * Activity function that returns a greeting for a city.
     */
    @FunctionName("SayHello")
    public String sayHello(
            @DurableActivityTrigger(name = "city") String city) {
        return "Hello " + city + "!";
    }
}
