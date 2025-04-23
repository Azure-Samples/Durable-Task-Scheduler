using System.Text.Json.Serialization;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Worker;
using Microsoft.DurableTask.Worker.AzureManaged;
using Microsoft.DurableTask.Client.AzureManaged;

WebApplicationBuilder builder = WebApplication.CreateBuilder(args);

string connectionString = builder.Configuration["DURABLE_TASK_SCHEDULER_CONNECTION_STRING"]
    ?? throw new InvalidOperationException("Missing required configuration 'DURABLE_TASK_SCHEDULER_CONNECTION_STRING'");

// Add all the generated orchestrations and activities automatically
builder.Services.AddDurableTaskWorker(builder =>
{
    builder.AddTasks(r => r.AddAllGeneratedTasks());
    builder.UseDurableTaskScheduler(connectionString);
});

// Register the client, which can be used to start orchestrations
builder.Services.AddDurableTaskClient(builder =>
{
    builder.UseDurableTaskScheduler(connectionString);
});

// Configure console logging using the simpler, more compact format
builder.Services.AddLogging(logging =>
{
    logging.AddSimpleConsole(options =>
    {
        options.SingleLine = true;
        options.UseUtcTimestamp = true;
        options.TimestampFormat = "yyyy-MM-ddTHH:mm:ss.fffZ ";
    });
});

// Configure the HTTP request pipeline
builder.Services.AddControllers().AddJsonOptions(options =>
{
    options.JsonSerializerOptions.Converters.Add(new JsonStringEnumConverter());
    options.JsonSerializerOptions.DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull;
});

// The actual listen URL can be configured in environment variables named "ASPNETCORE_URLS" or "ASPNETCORE_URLS_HTTPS"
WebApplication app = builder.Build();
app.MapControllers();
app.Run();
