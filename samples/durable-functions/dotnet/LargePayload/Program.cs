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
                    var credentialSettings = context.Configuration.GetSection("AzureMonitorExporter");
                    if (!string.IsNullOrWhiteSpace(context.Configuration["AzureMonitorExporter:credential"]))
                    {
                        options.Credential = factory.CreateTokenCredential(credentialSettings);
                    }
                });
            services.AddOpenTelemetry()
                .UseFunctionsWorkerDefaults()
                .UseAzureMonitorExporter();
        }
    })
    .Build();

host.Run();
