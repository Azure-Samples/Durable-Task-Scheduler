from __future__ import annotations

from durabletask import task

from activities.llm_generate import generate_answer
from activities.retriever import (
    search_document_store,
    search_knowledge_graph,
    search_vector_db,
)


def _wait_for_all(ctx: task.OrchestrationContext, tasks_to_wait_on: list[task.Task]):
    if hasattr(ctx, "task_all"):
        return ctx.task_all(tasks_to_wait_on)
    return task.when_all(tasks_to_wait_on)



def rag_orchestrator(ctx: task.OrchestrationContext, query: str):
    """Run a fan-out/fan-in RAG pipeline and return the synthesized answer."""
    task1 = ctx.call_activity(search_vector_db, input=query)
    task2 = ctx.call_activity(search_document_store, input=query)
    task3 = ctx.call_activity(search_knowledge_graph, input=query)

    retrieval_results: list[str] = yield _wait_for_all(ctx, [task1, task2, task3])

    combined_context = "\n\n".join(retrieval_results)
    answer = yield ctx.call_activity(
        generate_answer,
        input={"query": query, "context": combined_context},
    )
    return answer
