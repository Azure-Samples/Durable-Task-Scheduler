from __future__ import annotations

import os
import time

from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

from activities import generate_answer_copilot, search_document_store, search_knowledge_graph, search_vector_db
from orchestrations import rag_orchestration

LOCAL_ENDPOINT = 'http://localhost:8080'


def get_connection_config() -> dict:
    endpoint = os.getenv('ENDPOINT', LOCAL_ENDPOINT)
    taskhub = os.getenv('TASKHUB', 'default')
    return {
        'host_address': endpoint,
        'taskhub': taskhub,
        'secure_channel': endpoint != LOCAL_ENDPOINT,
        'token_credential': None,
    }


def main() -> None:
    with DurableTaskSchedulerWorker(**get_connection_config()) as worker:
        worker.add_orchestrator(rag_orchestration)
        worker.add_activity(search_vector_db)
        worker.add_activity(search_document_store)
        worker.add_activity(search_knowledge_graph)
        worker.add_activity(generate_answer_copilot)
        worker.start()

        print('Copilot SDK RAG worker is running. Press Ctrl+C to stop.')
        try:
            while True:
                time.sleep(3600)
        except KeyboardInterrupt:
            print('Stopping worker...')


if __name__ == '__main__':
    main()
