# JavaScript Samples for Durable Task SDK

This directory contains sample applications demonstrating orchestration patterns using the Durable Task JavaScript SDK with Azure Durable Task Scheduler.

## Prerequisites

- [Node.js 22+](https://nodejs.org/)
- [Docker](https://www.docker.com/products/docker-desktop/)

## Running samples with the Durable Task Scheduler Emulator

1. Pull and run the emulator:

	```bash
	docker pull mcr.microsoft.com/dts/dts-emulator:latest
	docker run -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
	```

2. The sample defaults to these local settings when `ENDPOINT` and `TASKHUB` aren't set:
	- Endpoint: `http://localhost:8080`
	- Task hub: `default`

## Available samples

- **fan-out-fan-in**: Parallel processing and aggregation workflow, with both local emulator and `azd` deployment support.

## Running the sample

```bash
cd fan-out-fan-in
npm install
npm run worker
```

In another terminal:

```bash
cd fan-out-fan-in
npm run client
```

## View orchestrations in the dashboard

Open `http://localhost:8082` and select the `default` task hub.
