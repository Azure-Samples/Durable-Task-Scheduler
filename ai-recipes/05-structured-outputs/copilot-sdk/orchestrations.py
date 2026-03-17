from __future__ import annotations

from datetime import timedelta

from durabletask import task
from durabletask.task import RetryPolicy

from activities import parse_reviews_with_copilot

DEFAULT_REVIEW_DATA = """
1. TrailSprint Headlamp | reviewer Maya K. | 5 stars | "Battery lasted all weekend and the package arrived a day early."
2. HomeBlend Mixer / reviewer: J. Ortiz / rating four out of five / dough hook snapped after two months / shipping box was dented
3. Northwind Coffee Pods - Sam - loved the flavor but half the box was crushed in transit - maybe 3/5
""".strip()
RETRY = RetryPolicy(first_retry_interval=timedelta(seconds=3), max_number_of_attempts=3, backoff_coefficient=2.0)


def structured_reviews_orchestration(ctx: task.OrchestrationContext, raw_review_data: str | None) -> dict:
    raw_review_data = raw_review_data or DEFAULT_REVIEW_DATA
    result = yield ctx.call_activity(
        parse_reviews_with_copilot,
        input=raw_review_data,
        retry_policy=RETRY,
    )
    return result
