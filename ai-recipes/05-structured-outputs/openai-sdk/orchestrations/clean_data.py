from __future__ import annotations

from datetime import timedelta

from durabletask import task
from durabletask.task import RetryPolicy

from activities.invoke_model import invoke_model


RETRY_POLICY = RetryPolicy(
    first_retry_interval=timedelta(seconds=5),
    max_number_of_attempts=3,
    backoff_coefficient=2.0,
    max_retry_interval=timedelta(seconds=30),
    retry_timeout=timedelta(seconds=120),
)



def clean_data(ctx: task.OrchestrationContext, raw_review_data: str):
    """Normalize messy product review data with structured outputs and bounded retries."""
    cleaned_payload = yield ctx.call_activity(
        invoke_model,
        input={
            "schema_name": "ProductReviewList",
            "input_text": raw_review_data,
        },
        retry_policy=RETRY_POLICY,
    )
    return cleaned_payload
