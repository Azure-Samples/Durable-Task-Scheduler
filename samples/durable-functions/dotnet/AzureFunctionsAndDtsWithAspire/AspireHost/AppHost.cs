var builder = DistributedApplication.CreateBuilder(args);

var storage = builder.AddAzureStorage("storage").RunAsEmulator();

var dts = builder.AddContainer("dts", "mcr.microsoft.com/dts/dts-emulator", "latest")
                 .WithEndpoint(name: "grpc", targetPort: 8080)
                 .WithHttpEndpoint(name: "http", targetPort: 8081)
                 .WithHttpEndpoint(name: "dashboard", targetPort: 8082);

var grpcEndpoint = dts.GetEndpoint("grpc");

var dtsConnectionString = ReferenceExpression.Create($"Endpoint=http://{grpcEndpoint.Property(EndpointProperty.Host)}:{grpcEndpoint.Property(EndpointProperty.Port)};Authentication=None");

builder.AddAzureFunctionsProject<Projects.AzureFunctions>("funcapp")
    .WithHostStorage(storage)
    .WithEnvironment("DURABLE_TASK_SCHEDULER_CONNECTION_STRING", dtsConnectionString)
    .WithEnvironment("TASKHUB_NAME", "default");

builder.Build().Run();
