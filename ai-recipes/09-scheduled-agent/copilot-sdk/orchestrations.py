from __future__ import annotations

from datetime import timedelta
from typing import Any

from durabletask import task
from durabletask.task import RetryPolicy

from activities import run_scheduled_review, store_report


RUN_RETRY = RetryPolicy(
    first_retry_interval=timedelta(seconds=5),
    max_number_of_attempts=3,
)


def scheduled_agent_orchestration(
    ctx: task.OrchestrationContext, input_: dict[str, Any] | None = None
) -> None:
    """Run a Copilot SDK agent on a durable schedule."""
    config = dict(input_ or {})
    interval_seconds = int(config.get("interval_seconds", 3600))
    max_runs = int(config.get("max_runs", 0))
    run_count = int(config.get("_run_count", 0))

    ctx.set_custom_status(
        {
            "stage": "running",
            "run_number": run_count + 1,
            "interval_seconds": interval_seconds,
            "max_runs": max_runs,
        }
    )

    result = yield ctx.call_activity(
        run_scheduled_review,
        input=config,
        retry_policy=RUN_RETRY,
    )

    yield ctx.call_activity(
        store_report,
        input={"content": result, "timestamp": ctx.current_utc_datetime.isoformat()},
        retry_policy=RUN_RETRY,
    )

    run_count += 1
    if max_runs > 0 and run_count >= max_runs:
        ctx.set_custom_status({"stage": "completed", "run_count": run_count})
        return

    ctx.set_custom_status(
        {
            "stage": "sleeping",
            "completed_runs": run_count,
            "interval_seconds": interval_seconds,
        }
    )
    yield ctx.create_timer(timedelta(seconds=interval_seconds))

    next_config = dict(config)
    next_config["_run_count"] = run_count
    ctx.continue_as_new(next_config)
