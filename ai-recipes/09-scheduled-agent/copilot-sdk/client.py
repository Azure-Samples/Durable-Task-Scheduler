from __future__ import annotations

import argparse
import os

from durabletask.azuremanaged.client import DurableTaskSchedulerClient

from orchestrations import scheduled_agent_orchestration


LOCAL_ENDPOINT = "http://localhost:8080"


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
    parser = argparse.ArgumentParser(description="Start a scheduled Copilot SDK orchestration.")
    parser.add_argument("--interval", type=int, default=3600, help="Number of seconds between runs.")
    parser.add_argument(
        "--prompt",
        default="Summarize recent changes in the codebase",
        help="Prompt to execute on each scheduled run.",
    )
    parser.add_argument("--max-runs", type=int, default=0, help="Maximum number of runs. Use 0 for infinite.")
    args = parser.parse_args()

    config = {
        "interval_seconds": args.interval,
        "prompt": args.prompt,
        "max_runs": args.max_runs,
    }

    client = DurableTaskSchedulerClient(**get_connection_config())
    instance_id = client.schedule_new_orchestration(scheduled_agent_orchestration, input=config)
    print(f"Started scheduled agent orchestration: {instance_id}")

    if args.max_runs > 0:
        timeout = max(60, args.interval * args.max_runs + 60)
        state = client.wait_for_orchestration_completion(instance_id, timeout=timeout)
        if state is None:
            raise TimeoutError("Timed out waiting for the scheduled agent orchestration to complete.")
        print(f"Completed with status: {state.runtime_status}")
    else:
        state = client.wait_for_orchestration_start(instance_id, timeout=60)
        if state is None:
            raise TimeoutError("Timed out waiting for the scheduled agent orchestration to start.")
        print(f"Runtime status: {state.runtime_status}")
        print("This orchestration is designed to keep running until terminated.")


if __name__ == "__main__":
    main()
