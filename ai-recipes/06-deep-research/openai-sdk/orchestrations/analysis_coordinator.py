from __future__ import annotations

import json
import re
from datetime import timedelta

from durabletask import task
from durabletask.task import RetryPolicy
from pydantic import BaseModel, Field

from activities.llm_activity import llm_activity
from activities.synthesize import create_comparison_report
from orchestrations.dimension_analyst import dimension_analyst


LLM_RETRY = RetryPolicy(
    first_retry_interval=timedelta(seconds=5),
    max_number_of_attempts=3,
    backoff_coefficient=2.0,
    max_retry_interval=timedelta(seconds=30),
    retry_timeout=timedelta(minutes=2),
)


class ComparisonPlan(BaseModel):
    products: list[str] = Field(default_factory=list)
    dimensions: list[str] = Field(default_factory=list)


def _extract_products(query: str) -> list[str]:
    match = re.search(r"compare\s+(.+?)(?:\s+for\s+|\s*$)", query, flags=re.IGNORECASE)
    candidate = match.group(1) if match else query
    parts = [part.strip(" .") for part in re.split(r"\bvs\.?\b|\bversus\b|,", candidate, flags=re.IGNORECASE) if part.strip(" .")]

    unique_products: list[str] = []
    for part in parts:
        if part not in unique_products:
            unique_products.append(part)

    return unique_products[:4] or ["Option A", "Option B"]


def _default_dimensions(query: str) -> list[str]:
    lowered = query.lower()
    if any(keyword in lowered for keyword in ("database", "postgresql", "mysql", "sqlite")):
        return ["performance", "operational complexity", "ecosystem", "portability"]
    if any(keyword in lowered for keyword in ("react", "vue", "svelte", "frontend")):
        return ["ecosystem", "learning curve", "performance", "enterprise fit"]
    return ["performance", "ecosystem", "learning curve", "operational fit"]


def _parse_plan(raw: str, original_query: str) -> ComparisonPlan:
    try:
        parsed = ComparisonPlan.model_validate(json.loads(raw))
        if parsed.products and parsed.dimensions:
            return ComparisonPlan(products=parsed.products[:4], dimensions=parsed.dimensions[:4])
    except Exception:
        pass

    return ComparisonPlan(
        products=_extract_products(original_query),
        dimensions=_default_dimensions(original_query),
    )


def _slugify(value: str) -> str:
    return re.sub(r"[^a-z0-9]+", "-", value.lower()).strip("-") or "dimension"


def analysis_coordinator(ctx: task.OrchestrationContext, comparison_query: str):
    ctx.set_custom_status({"stage": "planning", "query": comparison_query})
    plan_prompt = {
        "system_prompt": "Identify the products being compared and 3-4 comparison dimensions. Return JSON with products and dimensions.",
        "user_prompt": f"Comparison query: {comparison_query}",
        "temperature": 0.1,
    }
    raw_plan = yield ctx.call_activity(llm_activity, input=plan_prompt, retry_policy=LLM_RETRY)
    plan = _parse_plan(raw_plan, comparison_query)

    ctx.set_custom_status({"stage": "analyzing", "products": plan.products, "dimensions": plan.dimensions})
    parallel_tasks = [
        ctx.call_sub_orchestrator(
            dimension_analyst,
            input={
                "original_query": comparison_query,
                "products": plan.products,
                "dimension": dimension,
                "max_iterations": 2,
            },
            instance_id=f"{ctx.instance_id}:dimension:{_slugify(dimension)}",
            retry_policy=LLM_RETRY,
        )
        for dimension in plan.dimensions
    ]
    dimension_results = yield task.when_all(parallel_tasks)

    ctx.set_custom_status({"stage": "reporting", "dimensions_completed": len(dimension_results)})
    final_report = yield ctx.call_activity(
        create_comparison_report,
        input={
            "dimension_results": dimension_results,
            "original_query": comparison_query,
            "products": plan.products,
        },
        retry_policy=LLM_RETRY,
    )
    return final_report
