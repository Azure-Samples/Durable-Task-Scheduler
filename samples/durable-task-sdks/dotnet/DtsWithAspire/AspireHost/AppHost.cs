var builder = DistributedApplication.CreateBuilder(args);

var dts = builder.AddContainer("dts", "mcr.microsoft.com/dts/dts-emulator", "latest")
                 .WithEndpoint(name: "grpc", targetPort: 8080)
                 .WithHttpEndpoint(name: "http", targetPort: 8081)
                 .WithHttpEndpoint(name: "dashboard", targetPort: 8082);

var grpcEndpoint = dts.GetEndpoint("grpc");

var dtsConnectionString = ReferenceExpression.Create($"Endpoint=http://{grpcEndpoint.Property(EndpointProperty.Host)}:{grpcEndpoint.Property(EndpointProperty.Port)};TaskHub=default;Authentication=None");

builder.AddProject<Projects.Client>("client")
    .WithEnvironment("DURABLE_TASK_SCHEDULER_CONNECTION_STRING", dtsConnectionString);

builder.AddProject<Projects.Worker>("worker")
    .WithEnvironment("DURABLE_TASK_SCHEDULER_CONNECTION_STRING", dtsConnectionString);

builder.Build().Run();
