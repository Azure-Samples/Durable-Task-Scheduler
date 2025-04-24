// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package io.durabletask.samples;

import com.microsoft.durabletask.*;
import com.microsoft.durabletask.azuremanaged.DurableTaskSchedulerClientExtensions;
import com.microsoft.durabletask.azuremanaged.DurableTaskSchedulerWorkerExtensions;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.time.Duration;
import java.util.Arrays;
import java.util.List;
import java.util.StringTokenizer;
import java.util.concurrent.TimeoutException;
import java.util.stream.Collectors;

class FanOutFanInPattern {
    private static final Logger logger = LoggerFactory.getLogger(FanOutFanInPattern.class);

    public static void main(String[] args) throws IOException, InterruptedException, TimeoutException {
        // Get connection string from environment variable
        String connectionString = System.getenv("DURABLE_TASK_CONNECTION_STRING");
        if (connectionString == null) {
            // Default to local development connection string if not set
            "Endpoint=http://localhost:8080;TaskHub=default;Authentication=None";
        }

        // Create worker using Azure-managed extensions
        DurableTaskGrpcWorker worker = DurableTaskSchedulerWorkerExtensions.createWorkerBuilder(connectionString)
            .addOrchestration(new TaskOrchestrationFactory() {
                @Override
                public String getName() { return "FanOutFanIn_WordCount"; }

                @Override
                public TaskOrchestration create() {
                    return ctx -> {
                        List<?> inputs = ctx.getInput(List.class);
                        List<Task<Integer>> tasks = inputs.stream()
                                .map(input -> ctx.callActivity("CountWords", input.toString(), Integer.class))
                                .collect(Collectors.toList());
                        List<Integer> allWordCountResults = ctx.allOf(tasks).await();
                        int totalWordCount = allWordCountResults.stream().mapToInt(Integer::intValue).sum();
                        ctx.complete(totalWordCount);
                    };
                }
            })
            .addActivity(new TaskActivityFactory() {
                @Override
                public String getName() { return "CountWords"; }

                @Override
                public TaskActivity create() {
                    return ctx -> {
                        String input = ctx.getInput(String.class);
                        StringTokenizer tokenizer = new StringTokenizer(input);
                        return tokenizer.countTokens();
                    };
                }
            })
            .build();

        // Start the worker
        worker.start();

        // Create client using Azure-managed extensions
        DurableTaskClient client = DurableTaskSchedulerClientExtensions.createClientBuilder(connectionString).build();

        // The input is an arbitrary list of strings.
        List<String> listOfStrings = Arrays.asList(
                "Hello, world!",
                "The quick brown fox jumps over the lazy dog.",
                "If a tree falls in the forest and there is no one there to hear it, does it make a sound?",
                "The greatest glory in living lies not in never falling, but in rising every time we fall.",
                "Always remember that you are absolutely unique. Just like everyone else.");

        // Schedule an orchestration which will reliably count the number of words in all the given sentences.
        String instanceId = client.scheduleNewOrchestrationInstance(
                "FanOutFanIn_WordCount",
                new NewOrchestrationInstanceOptions().setInput(listOfStrings));
        logger.info("Started new orchestration instance: {}", instanceId);

        // Block until the orchestration completes. Then print the final status, which includes the output.
        OrchestrationMetadata completedInstance = client.waitForInstanceCompletion(
                instanceId,
                Duration.ofSeconds(30),
                true);
        logger.info("Orchestration completed: {}", completedInstance);
        logger.info("Output: {}", completedInstance.readOutputAs(int.class));

        // Shutdown the worker and exit
        worker.stop();
    }
}
