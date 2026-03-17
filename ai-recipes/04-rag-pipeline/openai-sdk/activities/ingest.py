from __future__ import annotations

from durabletask import task


DEFAULT_DOCUMENTS = [
    "https://contoso.example/docs/durable-task-overview",
    "https://contoso.example/docs/agent-patterns",
    "https://contoso.example/docs/rag-refresh-playbook",
]


def get_documents_to_ingest(ctx: task.ActivityContext, scheduler_input: dict | None) -> list[str]:
    """Return the next batch of documents to ingest."""
    if scheduler_input and scheduler_input.get("documents"):
        return list(scheduler_input["documents"])
    return DEFAULT_DOCUMENTS


def ingest_document(ctx: task.ActivityContext, doc_url: str) -> str:
    """Simulate downloading, chunking, embedding, and storing a document."""
    return (
        f"Ingested {doc_url}: downloaded content, created chunks, generated embeddings, "
        "and upserted vectors into the demo index."
    )
