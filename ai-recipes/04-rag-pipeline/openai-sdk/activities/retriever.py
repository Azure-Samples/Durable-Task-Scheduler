from __future__ import annotations

from durabletask import task


def search_vector_db(ctx: task.ActivityContext, query: str) -> str:
    """Simulate a vector similarity search for the incoming query."""
    matches = [
        f"semantic chunk: Durable execution persists progress for '{query}'.",
        f"semantic chunk: Fan-out/fan-in helps parallelize retrieval for '{query}'.",
    ]
    return "Vector DB results:\n- " + "\n- ".join(matches)


def search_document_store(ctx: task.ActivityContext, query: str) -> str:
    """Simulate a keyword/document-store lookup."""
    matches = [
        f"document excerpt: The knowledge base contains operational notes about '{query}'.",
        "document excerpt: Source documents can be chunked, embedded, and refreshed on a schedule.",
    ]
    return "Document store results:\n- " + "\n- ".join(matches)


def search_knowledge_graph(ctx: task.ActivityContext, query: str) -> str:
    """Simulate a knowledge-graph traversal."""
    matches = [
        f"entity relation: '{query}' -> durable workflows -> retries -> resilience.",
        "entity relation: retrieval pipelines connect documents, entities, and generated answers.",
    ]
    return "Knowledge graph results:\n- " + "\n- ".join(matches)
