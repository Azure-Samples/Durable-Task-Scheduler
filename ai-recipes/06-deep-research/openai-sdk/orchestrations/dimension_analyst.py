from __future__ import annotations

import json
from datetime import timedelta
from typing import Any, Union

from durabletask import task
from durabletask.task import RetryPolicy
from pydantic import BaseModel, Field

from activities.llm_activity import llm_activity
from activities.search import search_comparison


LLM_RETRY = RetryPolicy(
    first_retry_interval=timedelta(seconds=5),
    max_number_of_attempts=3,
    backoff_coefficient=2.0,
    max_retry_interval=timedelta(seconds=30),
    retry_timeout=timedelta(minutes=2),
)


class DimensionAnalystInput(BaseModel):
    original_query: str
    products: list[str] = Field(default_factory=list)
    dimension: str
    max_iterations: int = 2


class SearchDecision(BaseModel):
    search_query: str
    focus: str = ""
    done: bool = False
    reason: str = ""


def _parse_decision(raw: str, request: DimensionAnalystInput, iteration: int) -> SearchDecision:
    try:
        return SearchDecision.model_validate(json.loads(raw))
    except Exception:
        return SearchDecision(
            search_query=f"{request.original_query} {request.dimension} iteration {iteration}",
            focus=f"{request.dimension} evidence pass {iteration}",
            done=iteration >= request.max_iterations,
            reason="Fallback parser used because the LLM response was not valid JSON.",
        )


def _capture_product_notes(search_output: str, products: list[str]) -> dict[str, list[str]]:
    notes = {product: [] for product in products}
    for line in search_output.splitlines():
        stripped = line.strip()
        for product in products:
            prefix = f"- {product}:"
            if stripped.lower().startswith(prefix.lower()):
                notes[product].append(stripped.split(":", 1)[1].strip())
    return notes


def dimension_analyst(ctx: task.OrchestrationContext, input_: Union[dict[str, Any], DimensionAnalystInput]):
    request = input_ if isinstance(input_, DimensionAnalystInput) else DimensionAnalystInput.model_validate(input_)
    findings: list[str] = []
    explored_focuses: list[str] = []
    product_notes = {product: [] for product in request.products}

    for iteration in range(1, request.max_iterations + 1):
        ctx.set_custom_status(
            {
                "dimension": request.dimension,
                "iteration": iteration,
                "products": request.products,
            }
        )
        planning_prompt = {
            "system_prompt": "You are a competitive analysis planner. Return JSON with keys search_query, focus, done, and reason.",
            "user_prompt": (
                f"Original query: {request.original_query}\n"
                f"Products: {', '.join(request.products)}\n"
                f"Dimension: {request.dimension}\n"
                f"Iteration: {iteration}\n"
                f"Findings so far:\n"
                + ("\n\n".join(findings) if findings else "None yet.")
                + "\n\nPlan the next focused comparison search for this single dimension."
            ),
            "temperature": 0.1,
        }
        raw_decision = yield ctx.call_activity(llm_activity, input=planning_prompt, retry_policy=LLM_RETRY)
        decision = _parse_decision(raw_decision, request, iteration)

        search_output = yield ctx.call_activity(
            search_comparison,
            input={
                "topic": decision.search_query or request.original_query,
                "dimension": request.dimension,
                "products": request.products,
            },
        )
        explored_focuses.append(decision.focus or decision.search_query)
        findings.append(
            f"Iteration {iteration}\n"
            f"Focus: {decision.focus or 'Not provided.'}\n"
            f"Planner rationale: {decision.reason or 'Not provided.'}\n"
            f"Search output:\n{search_output}"
        )

        for product, notes in _capture_product_notes(search_output, request.products).items():
            product_notes[product].extend(notes)

        if decision.done:
            break

    lines = [
        f"## Dimension: {request.dimension.title()}",
        f"Products: {', '.join(request.products)}",
        "### Focus explored",
    ]
    lines.extend(f"- {focus}" for focus in explored_focuses)
    if not explored_focuses:
        lines.append("- No additional focus areas were explored.")

    lines.append("### Comparison Notes")
    for product in request.products:
        combined = " ".join(product_notes.get(product, [])[:2]) or f"Collect more evidence about {product} on {request.dimension}."
        lines.append(f"- {product}: {combined}")

    lines.extend(
        [
            "### Analyst Verdict",
            f"- {request.dimension.title()} exposes meaningful trade-offs across {', '.join(request.products)}.",
        ]
    )
    return "\n".join(lines)
