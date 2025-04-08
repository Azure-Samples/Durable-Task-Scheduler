FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build
WORKDIR /src

# Copy everything at once
COPY . .

# Try to restore directly if we're already in the project directory
RUN if [ -f "ClientService.csproj" ]; then \
        dotnet restore "ClientService.csproj"; \
    elif [ -f "*.csproj" ]; then \
        dotnet restore "*.csproj"; \
    else \
        # Try to find the csproj file
        dotnet restore $(find . -name "ClientService.csproj" | head -n 1); \
    fi

# Build the project - handle different contexts
RUN if [ -f "ClientService.csproj" ]; then \
        dotnet build "ClientService.csproj" -c Release -o /app/build; \
    elif [ -f "*.csproj" ]; then \
        dotnet build "*.csproj" -c Release -o /app/build; \
    else \
        # Try to find the csproj file
        CSPROJ=$(find . -name "ClientService.csproj" | head -n 1); \
        dotnet build "$CSPROJ" -c Release -o /app/build; \
    fi

FROM build AS publish
# Publish the project - handle different contexts
RUN if [ -f "ClientService.csproj" ]; then \
        dotnet publish "ClientService.csproj" -c Release -o /app/publish /p:UseAppHost=false /p:CopyOutputSymbolsToPublishDirectory=false; \
    elif [ -f "*.csproj" ]; then \
        dotnet publish "*.csproj" -c Release -o /app/publish /p:UseAppHost=false /p:CopyOutputSymbolsToPublishDirectory=false; \
    else \
        # Try to find the csproj file
        CSPROJ=$(find . -name "ClientService.csproj" | head -n 1); \
        dotnet publish "$CSPROJ" -c Release -o /app/publish /p:UseAppHost=false /p:CopyOutputSymbolsToPublishDirectory=false; \
    fi

FROM mcr.microsoft.com/dotnet/aspnet:8.0 AS final
WORKDIR /app
COPY --from=publish /app/publish .
EXPOSE 8080
ENV ASPNETCORE_URLS=http://+:8080
ENTRYPOINT ["dotnet", "ClientService.dll"]