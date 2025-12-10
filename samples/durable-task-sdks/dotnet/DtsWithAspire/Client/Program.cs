using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Client.AzureManaged;

var builder = WebApplication.CreateBuilder(args);

// Add services to the container.
// Learn more about configuring OpenAPI at https://aka.ms/aspnet/openapi
builder.Services.AddOpenApi();

builder.Services.AddDurableTaskClient(options =>
{
    string dtsConnectionString =
        builder.Configuration.GetValue<string>("DURABLE_TASK_SCHEDULER_CONNECTION_STRING")
        ?? throw new InvalidOperationException("DURABLE_TASK_SCHEDULER_CONNECTION_STRING is not set");

    options.UseDurableTaskScheduler(dtsConnectionString);
});

var app = builder.Build();

// Configure the HTTP request pipeline.
if (app.Environment.IsDevelopment())
{
    app.MapOpenApi();
}

app.UseHttpsRedirection();

app.MapGet("/start", async (DurableTaskClient client) =>
{
    var instance = await client.ScheduleNewOrchestrationInstanceAsync("SampleOrchestration");

    return instance is not null
     ? Results.Ok(new { instance })
     : Results.InternalServerError("Failed to start orchestration instance");
})
.WithName("Start");

app.Run();
