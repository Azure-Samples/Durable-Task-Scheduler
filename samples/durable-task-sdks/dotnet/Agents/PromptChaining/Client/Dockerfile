FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build
WORKDIR /app

# Copy csproj and restore as distinct layers
COPY ["Client.csproj", "./"]
COPY ["*.config", "./"]
# Create NuGet.config if it doesn't exist in the build context
RUN if [ ! -f "NuGet.config" ] && [ ! -f "nuget.config" ]; then echo '<?xml version="1.0" encoding="utf-8"?>\n<configuration>\n  <packageSources>\n    <clear />\n    <add key="nuget.org" value="https://api.nuget.org/v3/index.json" />\n  </packageSources>\n</configuration>' > NuGet.config; fi

# Copy any props files and create Directory.Packages.props if needed
COPY ["*.props", "./"]
# Create Directory.Packages.props if it doesn't exist
RUN if [ ! -f "Directory.Packages.props" ]; then echo '<Project>\n  <PropertyGroup>\n    <ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>\n  </PropertyGroup>\n  <ItemGroup>\n    <PackageVersion Include="Azure.Identity" Version="1.13.1" />\n    <PackageVersion Include="Azure.AI.Projects" Version="1.0.0-beta.9" />\n    <PackageVersion Include="Azure.AI.Agents.Persistent" Version="1.0.0" />\n    <PackageVersion Include="Azure.Core" Version="1.46.1" />\n    <PackageVersion Include="Microsoft.DurableTask.Client" Version="1.10.0" />\n    <PackageVersion Include="Microsoft.DurableTask.Worker" Version="1.10.0" />\n    <PackageVersion Include="Microsoft.DurableTask.Worker.Abstractions" Version="1.10.0" />\n    <PackageVersion Include="Microsoft.DurableTask.Worker.AzureManaged" Version="1.10.0-preview.1" />\n    <PackageVersion Include="Microsoft.DurableTask.Client.AzureManaged" Version="1.10.0-preview.1" />\n    <PackageVersion Include="Microsoft.DurableTask.Generators" Version="1.0.0-preview.1" />\n    <PackageVersion Include="Microsoft.Extensions.DependencyInjection" Version="8.0.0" />\n    <PackageVersion Include="Microsoft.Extensions.Configuration" Version="8.0.0" />\n    <PackageVersion Include="Microsoft.Extensions.Configuration.EnvironmentVariables" Version="8.0.0" />\n    <PackageVersion Include="Microsoft.Extensions.Hosting" Version="8.0.0" />\n    <PackageVersion Include="Microsoft.Extensions.Logging" Version="8.0.0" />\n    <PackageVersion Include="Microsoft.Extensions.Logging.Console" Version="8.0.0" />\n    <PackageVersion Include="Microsoft.Extensions.Http" Version="8.0.0" />\n    <PackageVersion Include="Grpc.Net.Client" Version="2.67.0" />\n    <PackageVersion Include="Microsoft.AspNetCore.OpenApi" Version="8.0.0" />\n    <PackageVersion Include="Swashbuckle.AspNetCore" Version="6.5.0" />\n  </ItemGroup>\n</Project>' > Directory.Packages.props; fi

RUN dotnet restore "Client.csproj"

# Copy source code and models
COPY ["Program.cs", "./"]
COPY ["Models/", "Models/"]

# Build the application
RUN dotnet build -c Release

# Publish the application
RUN dotnet publish -c Release -o /app/publish

# Build runtime image
FROM mcr.microsoft.com/dotnet/aspnet:8.0 AS final
WORKDIR /app
COPY --from=build /app/publish .

# Expose port 5000 explicitly for web API access
EXPOSE 5000

# Configure web server to listen on port 5000
ENV ASPNETCORE_URLS=http://+:5000
ENV ASPNETCORE_ENVIRONMENT=Production

# Set the entrypoint
ENTRYPOINT ["dotnet", "AgentChainingSample.Client.dll"]
