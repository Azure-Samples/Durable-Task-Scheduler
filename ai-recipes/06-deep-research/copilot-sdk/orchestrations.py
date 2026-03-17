from __future__ import annotations

import re
from datetime import timedelta
from typing import Any

from durabletask import task
from durabletask.task import RetryPolicy

from activities import decompose_query, research_dimension, synthesize_report

COPILOT_RETRY = RetryPolicy(
    first_retry_interval=timedelta(seconds=3),
    max_number_of_attempts=3,
    backoff_coefficient=2.0,
    max_retry_interval=timedelta(seconds=30),
    retry_timeout=timedelta(minutes=2),
)


def _slugify(value: str) -> str:
    return re.sub(r"[^a-z0-9]+", "-", value.lower()).strip("-") or "dimension"


def research_sub_orchestration(ctx: task.OrchestrationContext, input_: dict[str, Any]):
    result = yield ctx.call_activity(research_dimension, input=input_, retry_policy=COPILOT_RETRY)
    return result


def analysis_coordinator(ctx: task.OrchestrationContext, query: str):
    ctx.set_custom_status({"stage": "decomposing", "query": query})
    dimensions = yield ctx.call_activity(decompose_query, input=query, retry_policy=COPILOT_RETRY)
    dimensions = list(dimensions or [])[:5]

    ctx.set_custom_status({"stage": "researching", "dimensions": dimensions})
    tasks = [
        ctx.call_sub_orchestrator(
            research_sub_orchestration,
            input={"query": query, "dimension": dimension},
            instance_id=f"{ctx.instance_id}:dimension:{_slugify(dimension)}",
            retry_policy=COPILOT_RETRY,
        )
        for dimension in dimensions
    ]
    findings = yield task.when_all(tasks)

    ctx.set_custom_status({"stage": "synthesizing", "completed_dimensions": len(findings)})
    report = yield ctx.call_activity(
        synthesize_report,
        input={"query": query, "dimensions": dimensions, "findings": findings},
        retry_policy=COPILOT_RETRY,
    )
    return report
