using Azure.Monitor.OpenTelemetry.Exporter;
using Microsoft.Azure.Functions.Worker.OpenTelemetry;
using Microsoft.Extensions.Azure;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.DependencyInjection;

var host = new HostBuilder()
    .ConfigureFunctionsWebApplication()
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
