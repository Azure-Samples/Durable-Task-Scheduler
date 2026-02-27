FROM mcr.microsoft.com/dotnet/sdk:10.0 AS build
WORKDIR /src

# Copy csproj and restore dependencies
COPY ["Worker.csproj", "./"]
RUN dotnet restore

# Copy all files and build
COPY . .
RUN dotnet publish -c Release -o /app/publish

# Build runtime image
FROM mcr.microsoft.com/dotnet/runtime:10.0 AS final
WORKDIR /app
COPY --from=build /app/publish .

EXPOSE 8080
# Set the entrypoint
ENTRYPOINT ["dotnet", "Worker.dll"]
