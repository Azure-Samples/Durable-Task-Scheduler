from __future__ import annotations

import argparse
import os

from durabletask.azuremanaged.client import DurableTaskSchedulerClient

from orchestrations.ingestion_scheduler import ingestion_scheduler
from orchestrations.rag_orchestrator import rag_orchestrator


LOCAL_ENDPOINT = "http://localhost:8080"
DEFAULT_QUERY = "How does a durable RAG pipeline combine retrieval and generation?"



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



def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Durable Task RAG pipeline client")
    subparsers = parser.add_subparsers(dest="command", required=False)

    query_parser = subparsers.add_parser("query", help="Run the RAG orchestration")
    query_parser.add_argument("query", nargs="?", default=DEFAULT_QUERY)

    ingest_parser = subparsers.add_parser("schedule-ingestion", help="Start the eternal ingestion scheduler")
    ingest_parser.add_argument("--interval-minutes", type=int, default=60)
    ingest_parser.add_argument("documents", nargs="*")

    return parser



def main() -> None:
    parser = build_parser()
    args = parser.parse_args()
    command = args.command or "query"

    client = DurableTaskSchedulerClient(**get_connection_config())

    if command == "schedule-ingestion":
        schedule = {
            "interval_minutes": args.interval_minutes,
            "documents": args.documents,
        }
        instance_id = client.schedule_new_orchestration(ingestion_scheduler, input=schedule)
        print(f"Started ingestion scheduler with instance ID: {instance_id}")
        return

    instance_id = client.schedule_new_orchestration(rag_orchestrator, input=args.query)
    print(f"Started RAG orchestration with instance ID: {instance_id}")

    state = client.wait_for_orchestration_completion(instance_id, timeout=60)
    if state is None:
        raise TimeoutError("Timed out waiting for the RAG orchestration to complete.")

    print("\nAnswer:\n")
    print(state.serialized_output)


if __name__ == "__main__":
    main()
