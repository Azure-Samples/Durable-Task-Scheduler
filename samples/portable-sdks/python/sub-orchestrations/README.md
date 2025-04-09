# Portable SDK Sample for Sub Orchestrations and Fan-out / Fan-in

This sample demonstrates how to use the Durable Task SDK, also known as the Portable SDK, with the Durable Task Scheduler to create orchestrations that spin off child orchestrations but also perform parallel processing by leveraging the fan-out/fan-in application pattern.

The scenario showcases an order processing system where orders are processed in batches. 

> Note, for simplicity, this code is contained within a single source file. In real practice, you would have 


## Running the Examples
There are two separate ways to run an example:

- Using the Emulator
- Using a deployed Scheduler and Taskhub

### Running with a Deployed Scheduler and Taskhub Resource
1. To create a taskhub, follow these steps using the Azure CLI commands:

Create a Scheduler:
```bash
az durabletask scheduler create --resource-group --name --location --ip-allowlist "[0.0.0.0/0]" --sku-capacity 1 --sku-name "Dedicated" --tags "{'myattribute':'myvalue'}"
```

Create Your Taskhub:
```bash
az durabletask taskhub create --resource-group <testrg> --scheduler-name <testscheduler> --name <testtaskhub>
```

2. Retrieve the Endpoint for the Scheduler: Locate the taskhub in the Azure portal to find the endpoint.

3. Set the Environment Variables:
Bash:
```bash
export TASKHUB=<taskhubname>
export ENDPOINT=<taskhubEndpoint>
```
Powershell:
```powershell
$env:TASKHUB = "<taskhubname>"
$env:ENDPOINT = "<taskhubEndpoint>"
```

4. Install the Correct Packages
```bash
pip install -r requirements.txt
```

4. Grant your developer credentials the `Durable Task Data Contributor` Role.

### Running with the Emulator
The emulator simulates a scheduler and taskhub, packaged into an easy-to-use Docker container. For these steps, it is assumed that you are using port 8080.

1. Install Docker: If it is not already installed.

2. Pull the Docker Image for the Emulator:
```bash
docker pull mcr.microsoft.com/dts/dts-emulator:v0.0.6
```

3. Run the Emulator: Wait a few seconds for the container to be ready.
```bash
docker run --name dtsemulator -d -p 8080:8080 mcr.microsoft.com/dts/dts-emulator:v0.0.4
```

4. Set the Environment Variables:
Bash:
```bash
export TASKHUB=<taskhubname>
export ENDPOINT=http://localhost:8080
```
Powershell:
```powershell
$env:TASKHUB = "<taskhubname>"
$env:ENDPOINT = "http://localhost:8080"
```

5. Edit the Examples: Change the `token_credential` input of both the `DurableTaskSchedulerWorker` and `DurableTaskSchedulerClient` to `None`.

## Running the Sample

Once you have set up either the emulator or deployed scheduler, follow these steps to run the sample:

1. First, activate your Python virtual environment:
```bash
python -m venv venv
source venv/bin/activate  # On Windows, use: venv\Scripts\activate
```

2. Install the required packages:
```bash
pip install -r requirements.txt
```

3. Start the worker in a terminal:
```bash
python worker.py
```
You should see output indicating the worker has started and registered the orchestration and activities.

4. In a new terminal (with the virtual environment activated), run the orchestrator:
```bash
python orchestrator.py
```

The orchestrator will create a main orchestration that processes orders in batches with child orchestrations.

### What Happens When You Run the Sample

When you run the sample:

1. The orchestrator creates a new main orchestration instance with a batch of simulated order data.

2. The worker executes the `main_orchestration` function, which:
   - Splits the orders into batches
   - Creates child orchestrations (sub-orchestrations) for each batch
   - Waits for all child orchestrations to complete using Task.all()
   - Aggregates the results from all child orchestrations
   - Returns the summary of processed orders

3. Each child `process_order_batch` orchestration:
   - Receives a batch of orders
   - Processes each order in parallel using Task.all()
   - Collects the results
   - Returns a summary of the processed batch

4. The `process_order` activity is called for each individual order and simulates order processing.

5. The client displays the final results showing the total number of orders processed and their status.

This sample demonstrates two important patterns:
- **Sub-orchestrations**: Breaking down complex workflows into smaller, manageable orchestrations
- **Fan-out/Fan-in**: Processing items in parallel and then collecting results

## Sample Explanation

This sample demonstrates two important patterns in durable task orchestrations:

1. **Sub-orchestrations**: The main orchestration delegates work to child orchestrations, which helps:
   - Break down complex workflows into manageable pieces
   - Improve monitoring and debugging by isolating failures
   - Enable reuse of orchestration logic

2. **Fan-out/Fan-in**: Both the main orchestration and the child orchestrations use parallel processing:
   - The main orchestration fans out by creating multiple child orchestrations
   - Each child orchestration fans out by processing multiple orders in parallel
   - Results are aggregated (fan-in) at both levels

These patterns are useful in real-world scenarios such as:
- Order processing systems
- Batch data processing
- Distributed workloads
- Complex approval workflows with multiple stages

The sample simulates an order processing system where orders are batched for efficient processing. Each order goes through its own processing flow in parallel, and the results are aggregated to provide an overall status report.
