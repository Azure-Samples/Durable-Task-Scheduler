using Azure.Monitor.OpenTelemetry.Exporter;
using Microsoft.Azure.Functions.Worker.OpenTelemetry;
using Microsoft.Extensions.Azure;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using OpenTelemetry;
using OpenTelemetry.Trace;

var host = new HostBuilder()
    .ConfigureFunctionsWebApplication()
    .ConfigureServices((context, services) =>
    {
        var connectionString = context.Configuration["APPLICATIONINSIGHTS_CONNECTION_STRING"];
        if (!string.IsNullOrWhiteSpace(connectionString))
        {
            services.AddAzureClientsCore();
            services.AddOptions<AzureMonitorExporterOptions>()
                .Configure<AzureComponentFactory>((options, factory) =>
                {
                    options.ConnectionString = connectionString;
                    var credentialSettings = context.Configuration.GetSection("AzureMonitorExporter");
                    if (!string.IsNullOrWhiteSpace(context.Configuration["AzureMonitorExporter:credential"]))
                    {
                        options.Credential = factory.CreateTokenCredential(credentialSettings);
                    }
                });
            services.AddOpenTelemetry()
                .UseFunctionsWorkerDefaults()
                .WithTracing(tracing => tracing.AddSource("Microsoft.DurableTask"))
                .UseAzureMonitorExporter();
        }
    })
    .Build();

host.Run();
