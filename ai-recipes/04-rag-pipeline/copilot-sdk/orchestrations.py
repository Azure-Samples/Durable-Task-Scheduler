from __future__ import annotations

from datetime import timedelta

from durabletask import task
from durabletask.task import RetryPolicy

from activities import generate_answer_copilot, search_document_store, search_knowledge_graph, search_vector_db

RETRY = RetryPolicy(
    first_retry_interval=timedelta(seconds=2),
    max_number_of_attempts=3,
    backoff_coefficient=2.0,
    max_retry_interval=timedelta(seconds=30),
    retry_timeout=timedelta(minutes=2),
)


def _wait_for_all(ctx: task.OrchestrationContext, tasks_to_wait_on):
    if hasattr(ctx, 'task_all'):
        return ctx.task_all(tasks_to_wait_on)
    return task.when_all(tasks_to_wait_on)


def rag_orchestration(ctx: task.OrchestrationContext, query: str) -> str:
    tasks = [
        ctx.call_activity(search_vector_db, input=query),
        ctx.call_activity(search_document_store, input=query),
        ctx.call_activity(search_knowledge_graph, input=query),
    ]
    results: list[str] = yield _wait_for_all(ctx, tasks)
    combined_context = '\n\n'.join(results)

    answer = yield ctx.call_activity(
        generate_answer_copilot,
        input={'query': query, 'context': combined_context},
        retry_policy=RETRY,
    )
    return answer
