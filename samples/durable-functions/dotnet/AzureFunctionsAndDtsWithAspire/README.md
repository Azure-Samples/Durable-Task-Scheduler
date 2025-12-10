# Azure Functions and Durable Task Scheduler with Aspire

This sample demonstrates the orchestration of the local startup of an Azure Functions project backed by the Durable Task Scheduler (DTS) using Aspire.

# Prerequisites

 - Docker (for starting Azure Storage and DTS emulators)
 - .NET SDK 10 or later

 # Starting the application

Open a terminal at the `AzureFunctionsAndDtsWithAspire` host folder and run:

```bash
cd AspireHost
dotnet run
```

The Aspire host should start the application and present a URL to its dashboard:

```bash
Using launch settings from /Users/<user>/Source/Azure-Samples/Durable-Task-Scheduler/samples/durable-functions/dotnet/AzureFunctionsAndDtsWithAspire/AspireHost/Properties/launchSettings.json...
Building...
info: Aspire.Hosting.DistributedApplication[0]
      Aspire version: 13.0.2+8924dc6a494bfb53476657dcbb7a7764edb94b43
info: Aspire.Hosting.DistributedApplication[0]
      Distributed application starting.
info: Aspire.Hosting.DistributedApplication[0]
      Application host directory is: /Users/<user>/Source/Azure-Samples/Durable-Task-Scheduler/samples/durable-functions/dotnet/AzureFunctionsAndDtsWithAspire/AspireHost
info: Aspire.Hosting.DistributedApplication[0]
      Distributed application started. Press Ctrl+C to shut down.
info: Aspire.Hosting.DistributedApplication[0]
      Now listening on: https://localhost:17050
info: Aspire.Hosting.DistributedApplication[0]
      Login to the dashboard at https://localhost:17050/login?t=b3977d4d86dbb0c83e0d06a2af2440d7
```

# Starting an orchestration

From the Aspire dashboard, open the logs for the Azure Functions (`funcapp`) resource and start an orchestration using the `SampleOrchestration_HttpStart` link.

# Observing the orchestration

From the Aspire dashboard, open the details panel for the DTS (`dts`) resource and browse to its `dashboard` endpoint and its `default` taskhub. Select the new orchestration to see its event timeline.