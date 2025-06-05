using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;

namespace DurableFunctionsSaga
{
    public class Program
    {
        public static void Main(string[] args)
        {
            var host = new HostBuilder()
                .ConfigureFunctionsWorkerDefaults()
                .ConfigureAppConfiguration((context, config) =>
                {
                    config.AddEnvironmentVariables();
                })
                .ConfigureServices((context, services) =>
                {
                    // Log the DurableTaskSchedulerConnection (not for production)
                    var configuration = services.BuildServiceProvider().GetService<IConfiguration>();
                    var loggerFactory = services.BuildServiceProvider().GetService<ILoggerFactory>();
                    var logger = loggerFactory?.CreateLogger<Program>();
                    
                    string connectionString = configuration?["DurableTaskSchedulerConnection"] ?? "Endpoint=http://localhost:8080;TaskHub=SagaTaskHub;Authentication=None";
                    logger?.LogInformation("Using Durable Task Scheduler connection: {ConnectionString}", connectionString);
                })
                .Build();

            host.Run();
        }
    }
}
