# Portable SDK Sample for Eternal Orchestrations

This sample demonstrates how to use the Durable Task SDK, also known as the Portable SDK, with the Durable Task Scheduler to create eternal orchestrations - orhcestrations that run in a loop.

The scenario demonstrates a periodic cleanup orchestration that needs to be executed repeatedly. The cleanup activity itself takes 5 seconds to complete. After the cleanup task finishes, a timer is set to schedule the next orchestration 15 seconds from the current time by calling the continue_as_new method. For demonstration purposes, we limit the number of times the orchestration is repeated to 5.

## Running the Examples
There are two separate ways to run an example:

- Using the Emulator
- Using a deployed Scheduler and Taskhub

### Running with a Deployed Scheduler and Taskhub rResource
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
docker run --name dtsemulator -d -p 8080:8080 mcr.microsoft.com/dts/dts-emulator:v0.0.6
```

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

4. Edit the Examples: Change the token_credential input of both the `DurableTaskSchedulerWorker` and `DurableTaskSchedulerClient` to `None`.

### Running the Examples
You can now execute the sample using Python:

Start the worker and ensure the TASKHUB and ENDPOINT environment variables are set in your shell:
```bash 
python3 ./worker.py
```

Next, start the orchestrator and make sure the TASKHUB and ENDPOINT environment variables are set in your shell:
```bash
python3 ./orchestrator.py
```

You should start seeing logs for processing orders in both shell outputs.

### Review Orchestration History and Status in the Durable Task Scheduler Dashboard
To access the Durable Task Scheduler Dashboard, follow these steps:

- **Using the Emulator**: By default, the dashboard runs on portal 8082. Navigate to http://localhost:8082 and click on the default task hub.

- **Using a Deployed Scheduler**: Navigate to the Scheduler resource. Then, go to the Task Hub subresource that you are using and click on the dashboard URL in the top right corner.
