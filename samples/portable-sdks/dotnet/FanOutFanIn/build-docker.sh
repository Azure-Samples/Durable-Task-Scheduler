#!/bin/bash

# Script to build Docker images with the correct context
echo "Building Docker images with proper context..."

# Build the ClientService Docker image
echo "Building ClientService Docker image..."
docker build -t client-service -f ./ClientService/Dockerfile --no-cache .

# Build the WorkerService Docker image
echo "Building WorkerService Docker image..."
docker build -t worker-service -f ./WorkerService/Dockerfile --no-cache .

echo "Docker build completed successfully!"

# If we're in an azd environment, tag images appropriately
if [ -n "$AZURE_CONTAINER_REGISTRY_ENDPOINT" ]; then
  echo "Tagging images for Azure Container Registry..."
  docker tag client-service "$AZURE_CONTAINER_REGISTRY_ENDPOINT/client-service:latest"
  docker tag worker-service "$AZURE_CONTAINER_REGISTRY_ENDPOINT/worker-service:latest"
fi