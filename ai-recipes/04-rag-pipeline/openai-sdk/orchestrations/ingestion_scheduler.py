from __future__ import annotations

from datetime import timedelta

from durabletask import task

from activities.ingest import get_documents_to_ingest, ingest_document


DEFAULT_INTERVAL_MINUTES = 60


def _wait_for_all(ctx: task.OrchestrationContext, tasks_to_wait_on: list[task.Task]):
    if hasattr(ctx, "task_all"):
        return ctx.task_all(tasks_to_wait_on)
    return task.when_all(tasks_to_wait_on)



def ingestion_scheduler(ctx: task.OrchestrationContext, schedule: dict | None):
    """Eternal orchestration that periodically ingests documents."""
    schedule = schedule or {}
    interval_minutes = int(schedule.get("interval_minutes", DEFAULT_INTERVAL_MINUTES))

    doc_urls: list[str] = yield ctx.call_activity(get_documents_to_ingest, input=schedule)
    ingest_tasks = [ctx.call_activity(ingest_document, input=doc_url) for doc_url in doc_urls]
    ingest_results: list[str] = yield _wait_for_all(ctx, ingest_tasks)

    ctx.set_custom_status(
        {
            "documents_seen": len(doc_urls),
            "last_run_results": ingest_results,
            "next_poll_minutes": interval_minutes,
        }
    )

    yield ctx.create_timer(timedelta(minutes=interval_minutes))
    ctx.continue_as_new(schedule)
