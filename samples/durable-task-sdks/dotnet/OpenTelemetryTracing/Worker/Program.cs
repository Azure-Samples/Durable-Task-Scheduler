using Microsoft.DurableTask;
using Microsoft.DurableTask.Worker;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using OpenTelemetry;
using OpenTelemetry.Resources;
using OpenTelemetry.Trace;

var builder = Host.CreateApplicationBuilder(args);

// Configure OpenTelemetry tracing
builder.Services.AddOpenTelemetry()
    .ConfigureResource(resource => resource.AddService("durable-worker"))
    .WithTracing(tracing =>
    {
        tracing
            .AddSource("Microsoft.DurableTask")
            .AddOtlpExporter(opts =>
            {
                opts.Endpoint = new Uri(
                    Environment.GetEnvironmentVariable("OTEL_EXPORTER_OTLP_ENDPOINT") 
                    ?? "http://localhost:4317");
            });
    });

// Configure Durable Task worker
string endpoint = Environment.GetEnvironmentVariable("ENDPOINT") ?? "http://localhost:8080";
string taskHub = Environment.GetEnvironmentVariable("TASKHUB") ?? "default";
string connectionString = endpoint.Contains("localhost")
    ? $"Endpoint={endpoint};TaskHub={taskHub};Authentication=None"
    : $"Endpoint={endpoint};TaskHub={taskHub};Authentication=DefaultAzure";

builder.Services.AddDurableTaskWorker(builder =>
{
    builder.AddTasks(tasks =>
    {
        tasks.AddOrchestratorFunc<string, string>("OrderProcessingOrchestration", async (ctx, input) =>
        {
            // Step 1: Validate the order
            var validated = await ctx.CallActivityAsync<string>("ValidateOrder", input);

            // Step 2: Process payment
            var payment = await ctx.CallActivityAsync<string>("ProcessPayment", validated);

            // Step 3: Ship order
            var shipment = await ctx.CallActivityAsync<string>("ShipOrder", payment);

            // Step 4: Send notification
            var result = await ctx.CallActivityAsync<string>("SendNotification", shipment);

            return result;
        });

        tasks.AddActivityFunc<string, string>("ValidateOrder", (ctx, input) =>
        {
            Console.WriteLine($"[ValidateOrder] Validating order: {input}");
            Thread.Sleep(100); // Simulate work
            return Task.FromResult($"Validated({input})");
        });

        tasks.AddActivityFunc<string, string>("ProcessPayment", (ctx, input) =>
        {
            Console.WriteLine($"[ProcessPayment] Processing payment for: {input}");
            Thread.Sleep(200); // Simulate work
            return Task.FromResult($"Paid({input})");
        });

        tasks.AddActivityFunc<string, string>("ShipOrder", (ctx, input) =>
        {
            Console.WriteLine($"[ShipOrder] Shipping: {input}");
            Thread.Sleep(150); // Simulate work
            return Task.FromResult($"Shipped({input})");
        });

        tasks.AddActivityFunc<string, string>("SendNotification", (ctx, input) =>
        {
            Console.WriteLine($"[SendNotification] Notifying customer: {input}");
            Thread.Sleep(50); // Simulate work
            return Task.FromResult($"Notified({input})");
        });
    });
})
.UseDurableTaskScheduler(connectionString);

var host = builder.Build();
Console.WriteLine("Worker started. Press Ctrl+C to exit.");
await host.RunAsync();
