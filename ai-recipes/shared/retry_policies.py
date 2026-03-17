"""Standard retry policies for AI workloads."""

from durabletask import RetryPolicy
from datetime import timedelta


# For LLM API calls — generous retries with exponential backoff
LLM_RETRY_POLICY = RetryPolicy(
    first_retry_interval=timedelta(seconds=2),
    max_number_of_attempts=10,
    backoff_coefficient=2.0,
    max_retry_interval=timedelta(minutes=1),
    retry_timeout=timedelta(minutes=5),
)

# For structured output validation — fewer attempts (persistent failure = bad schema)
STRUCTURED_OUTPUT_RETRY_POLICY = RetryPolicy(
    first_retry_interval=timedelta(seconds=1),
    max_number_of_attempts=3,
    backoff_coefficient=1.5,
    max_retry_interval=timedelta(seconds=10),
)

# For external API calls (tool invocations, data retrieval)
TOOL_RETRY_POLICY = RetryPolicy(
    first_retry_interval=timedelta(seconds=1),
    max_number_of_attempts=5,
    backoff_coefficient=2.0,
    max_retry_interval=timedelta(seconds=30),
)
