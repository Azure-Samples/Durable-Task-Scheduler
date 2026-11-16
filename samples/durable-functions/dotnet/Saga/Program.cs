using Azure.Monitor.OpenTelemetry.Exporter;
using Microsoft.Azure.Functions.Worker.OpenTelemetry;
using Microsoft.Extensions.Azure;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;

var host = new HostBuilder()
    .ConfigureFunctionsWorkerDefaults()
    .ConfigureServices((context, services) =>
    {
        string? connectionString = context.Configuration["APPLICATIONINSIGHTS_CONNECTION_STRING"];
        if (!string.IsNullOrWhiteSpace(connectionString))
        {
            services.AddAzureClientsCore();
            services.AddOptions<AzureMonitorExporterOptions>()
                .Configure<AzureComponentFactory>((options, factory) =>
                {
                    options.ConnectionString = connectionString;
                    options.Credential = factory.CreateTokenCredential(
                        context.Configuration.GetSection("AzureMonitorExporter"));
                });
            services.AddOpenTelemetry()
                .UseFunctionsWorkerDefaults()
                .UseAzureMonitorExporter();
        }
    })
    .Build();

host.Run();
