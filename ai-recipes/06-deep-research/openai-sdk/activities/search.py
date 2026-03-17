from __future__ import annotations

import re
from typing import Any, Union

from durabletask import task
from pydantic import BaseModel, Field


class SearchRequest(BaseModel):
    topic: str
    dimension: str
    products: list[str] = Field(default_factory=list)


def _extract_products(topic: str) -> list[str]:
    match = re.search(r"compare\s+(.+?)(?:\s+for\s+|\s*$)", topic, flags=re.IGNORECASE)
    candidate = match.group(1) if match else topic
    parts = [part.strip(" .") for part in re.split(r"\bvs\.?\b|\bversus\b|,", candidate, flags=re.IGNORECASE) if part.strip(" .")]

    unique_products: list[str] = []
    for part in parts:
        if part not in unique_products:
            unique_products.append(part)

    return unique_products[:4]


def _product_note(product: str, dimension: str) -> str:
    lowered_product = product.lower()
    lowered_dimension = dimension.lower()
    specific_notes = {
        ("postgresql", "performance"): "Strong on complex queries and extensibility, but it usually asks for more tuning than simpler database options.",
        ("mysql", "performance"): "Predictable OLTP performance and wide hosting support make it easy to benchmark and operate at scale.",
        ("sqlite", "performance"): "Excellent for single-node and embedded workloads, but write-heavy concurrency becomes the main trade-off.",
        ("postgresql", "operational complexity"): "Richer features give teams more headroom, but backups, tuning, and upgrade planning need deliberate ownership.",
        ("mysql", "operational complexity"): "Operational patterns are well understood and widely documented, which lowers day-two surprises for many teams.",
        ("sqlite", "operational complexity"): "The simplest operational story of the group because there is no server tier, but scaling patterns are narrower.",
        ("react", "ecosystem"): "The broadest ecosystem, deepest hiring pool, and strongest enterprise integration story, with the cost of more choice and framework sprawl.",
        ("vue", "ecosystem"): "A productive ecosystem with good documentation and a cohesive core, though enterprise hiring depth is narrower than React.",
        ("svelte", "ecosystem"): "A smaller ecosystem that feels modern and focused, but fewer enterprise templates and long-lived libraries are available.",
        ("react", "learning curve"): "JSX, state tooling, and architectural freedom create flexibility, but teams need conventions to stay aligned.",
        ("vue", "learning curve"): "Usually the smoothest ramp for frontend teams because the defaults are approachable and opinionated enough to guide structure.",
        ("svelte", "learning curve"): "Component authoring feels lightweight and intuitive, though some teams may need to adapt to a smaller knowledge base.",
    }
    if (lowered_product, lowered_dimension) in specific_notes:
        return specific_notes[(lowered_product, lowered_dimension)]

    dimension_lenses = {
        "performance": "runtime efficiency, throughput, and scaling ceilings",
        "ecosystem": "community depth, integration options, and hiring signal",
        "learning curve": "ramp-up time and developer productivity",
        "enterprise fit": "governance, predictability, and long-term maintainability",
        "operational complexity": "deployment overhead, observability, and maintenance burden",
        "portability": "deployment flexibility and lock-in risk",
        "operational fit": "deployment model and support burden",
    }
    lens = dimension_lenses.get(lowered_dimension, "fit, trade-offs, and adoption signals")
    return f"Mock comparison sources describe {product} in terms of {lens}; validate the strongest claims with primary docs and benchmarks."


def search_comparison(ctx: task.ActivityContext, input_: Union[dict[str, Any], SearchRequest]) -> str:
    request = input_ if isinstance(input_, SearchRequest) else SearchRequest.model_validate(input_)
    topic = request.topic.strip() or "technology comparison"
    products = request.products or _extract_products(topic) or ["Option A", "Option B"]

    lines = [f"Mock comparison scan for \"{request.dimension}\" within \"{topic}\":"]
    for product in products:
        lines.append(f"- {product}: {_product_note(product, request.dimension)}")
    lines.append(
        "- Shared signal: enterprise teams usually weigh migration cost, talent availability, and failure modes alongside raw feature depth."
    )
    return "\n".join(lines)
