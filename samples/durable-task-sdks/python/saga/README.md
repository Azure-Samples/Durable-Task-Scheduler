# Saga Pattern — Durable Task SDK (Python)

Python | Durable Task SDK

## Description

Demonstrates the **Saga** pattern (compensating transactions) using the Durable Task SDK for Python. A travel booking orchestration reserves a flight, hotel, and rental car in sequence. If any step fails, previously completed steps are rolled back using compensating actions.

This pattern is useful for:
- Distributed transactions across multiple services
- Booking systems where all-or-nothing semantics are required
- Any multi-step process requiring rollback on failure

## Prerequisites

1. [Python 3.10+](https://www.python.org/downloads/)
2. [Docker](https://www.docker.com/products/docker-desktop/) (for the emulator)

## Quick Run

1. Start the Durable Task Scheduler emulator:
   ```bash
   docker run -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
   ```

2. Install dependencies:
   ```bash
   pip install durabletask
   ```

3. Start the worker (in one terminal):
   ```bash
   python worker.py
   ```

4. Run the client (in another terminal):
   ```bash
   python client.py
   ```

5. View in the dashboard: http://localhost:8082

## How It Works

1. The orchestration attempts to book a flight, hotel, and car in sequence
2. Each booking activity returns a confirmation ID on success
3. If any step fails (e.g., no cars available), the orchestration enters **compensation mode**
4. Compensation runs in reverse order — cancelling the hotel, then the flight
5. The orchestration returns either a successful booking or a failure with details of what was rolled back

## Learn More

- [Saga Pattern](https://learn.microsoft.com/azure/architecture/reference-architectures/saga/saga)
- [Durable Task Scheduler Documentation](https://aka.ms/dts-documentation)
