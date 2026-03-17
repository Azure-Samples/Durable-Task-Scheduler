from __future__ import annotations

import json
import os

from durabletask.azuremanaged.client import DurableTaskSchedulerClient

from orchestrations.clean_data import clean_data


LOCAL_ENDPOINT = "http://localhost:8080"
SAMPLE_DATA = """
1. TrailSprint Headlamp | reviewer Maya K. | 5 stars | "Battery lasted all weekend and the package arrived a day early." | verified buyer
2. HomeBlend Mixer / reviewer: J. Ortiz / rating four out of five / dough hook snapped after two months / shipping box was dented / verification unclear
3. Northwind Coffee Pods - Sam - loved the flavor but half the box was crushed in transit - maybe 3/5 - bought it with my monthly subscription
"""



def get_connection_config() -> dict:
    endpoint = os.getenv("ENDPOINT", LOCAL_ENDPOINT)
    taskhub = os.getenv("TASKHUB", "default")
    is_local = endpoint == LOCAL_ENDPOINT

    credential = None
    if not is_local:
        from azure.identity import DefaultAzureCredential

        credential = DefaultAzureCredential()

    return {
        "host_address": endpoint,
        "taskhub": taskhub,
        "secure_channel": not is_local,
        "token_credential": credential,
    }



def main() -> None:
    client = DurableTaskSchedulerClient(**get_connection_config())
    instance_id = client.schedule_new_orchestration(clean_data, input=SAMPLE_DATA)
    print(f"Started structured-output orchestration with instance ID: {instance_id}")

    state = client.wait_for_orchestration_completion(instance_id, timeout=60)
    if state is None:
        raise TimeoutError("Timed out waiting for the structured-output orchestration to complete.")

    cleaned = json.loads(state.serialized_output)
    print("\nCleaned reviews:\n")
    print(json.dumps(cleaned, indent=2))


if __name__ == "__main__":
    main()
