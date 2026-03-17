from __future__ import annotations

import os
import time

from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

from activities.ingest import get_documents_to_ingest, ingest_document
from activities.llm_generate import generate_answer
from activities.retriever import (
    search_document_store,
    search_knowledge_graph,
    search_vector_db,
)
from orchestrations.ingestion_scheduler import ingestion_scheduler
from orchestrations.rag_orchestrator import rag_orchestrator


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
    with DurableTaskSchedulerWorker(**get_connection_config()) as worker:
        worker.add_orchestrator(rag_orchestrator)
        worker.add_orchestrator(ingestion_scheduler)

        worker.add_activity(search_vector_db)
        worker.add_activity(search_document_store)
        worker.add_activity(search_knowledge_graph)
        worker.add_activity(generate_answer)
        worker.add_activity(get_documents_to_ingest)
        worker.add_activity(ingest_document)

        worker.start()
        print("RAG pipeline worker is running. Press Ctrl+C to stop.")

        try:
            while True:
                time.sleep(3600)
        except KeyboardInterrupt:
            print("Stopping worker...")


if __name__ == "__main__":
    main()
