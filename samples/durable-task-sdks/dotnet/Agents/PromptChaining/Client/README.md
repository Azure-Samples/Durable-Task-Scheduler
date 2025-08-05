# Testing the Agent Chaining API

This directory contains a `test.http` file that can be used to test the API endpoints using VS Code REST Client extension.

## Prerequisites

1. Install the [REST Client extension](https://marketplace.visualstudio.com/items?itemName=humao.rest-client) for VS Code.

## Using the test.http file

1. Open the `test.http` file in VS Code
2. You'll see several HTTP requests defined in the file
3. Click the "Send Request" link that appears above each request to execute it
4. The response will be displayed in a new tab

## Available Endpoints

- `GET /health` - Health check endpoint
- `POST /api/content` - Create a new content generation request
- `GET /api/content` - List all active content generation requests
- `GET /api/content/{instanceId}` - Get status of a specific request
- `GET /api/content/{instanceId}/wait` - Wait for completion of a specific request with timeout

## Sample Workflow

1. Send a POST request to `/api/content` with a topic
2. Copy the instance ID from the response
3. Use that ID to check status with `GET /api/content/{instanceId}`
4. When the status is "Completed", you can retrieve the full result

## Running Locally

```bash
cd Client
dotnet run
```

The API will be available at http://localhost:5000.
