#!/bin/bash

# Script to push Docker images with automatic retries
# Usage: ./docker-push-with-retry.sh <image-name>

MAX_RETRIES=5
RETRY_DELAY=10
IMAGE_NAME=$1

if [ -z "$IMAGE_NAME" ]; then
  echo "Error: Image name is required"
  echo "Usage: $0 <image-name>"
  exit 1
fi

echo "Pushing image $IMAGE_NAME with up to $MAX_RETRIES retries..."

for ((i=1; i<=MAX_RETRIES; i++)); do
  echo "Attempt $i of $MAX_RETRIES..."
  
  # Try pushing the image
  docker push $IMAGE_NAME
  
  # Check if push was successful
  if [ $? -eq 0 ]; then
    echo "Successfully pushed $IMAGE_NAME on attempt $i"
    exit 0
  fi
  
  echo "Push failed. Waiting $RETRY_DELAY seconds before retrying..."
  sleep $RETRY_DELAY
  
  # Increase delay for next attempt (exponential backoff)
  RETRY_DELAY=$((RETRY_DELAY * 2))
done

echo "Failed to push $IMAGE_NAME after $MAX_RETRIES attempts"
exit 1
