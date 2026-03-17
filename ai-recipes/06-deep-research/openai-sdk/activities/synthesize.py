from __future__ import annotations

import os
import re
from pathlib import Path
from typing import Any, Union

from dotenv import load_dotenv
from durabletask import task
from openai import OpenAI
from pydantic import BaseModel, Field

load_dotenv(Path(__file__).resolve().parents[3] / ".env")


class ComparisonReportRequest(BaseModel):
    dimension_results: list[str] = Field(default_factory=list)
    original_query: str
    products: list[str] = Field(default_factory=list)
    model: str = Field(default="gpt-5.4")


def _extract_products(query: str) -> list[str]:
    match = re.search(r"compare\s+(.+?)(?:\s+for\s+|\s*$)", query, flags=re.IGNORECASE)
    candidate = match.group(1) if match else query
    parts = [part.strip(" .") for part in re.split(r"\bvs\.?\b|\bversus\b|,", candidate, flags=re.IGNORECASE) if part.strip(" .")]

    unique_products: list[str] = []
    for part in parts:
        if part not in unique_products:
            unique_products.append(part)

    return unique_products[:4] or ["Option A", "Option B"]


def _parse_dimension_name(result: str) -> str:
    match = re.search(r"^## Dimension:\s*(.+)$", result, flags=re.MULTILINE)
    return match.group(1).strip() if match else "Dimension"


def _parse_dimension_notes(result: str) -> dict[str, str]:
    notes: dict[str, str] = {}
    for line in result.splitlines():
        stripped = line.strip()
        if stripped.startswith("- ") and ":" in stripped:
            product, note = stripped[2:].split(":", 1)
            notes[product.strip()] = note.strip()
    return notes


def _pros_cons_for(product: str, query: str) -> tuple[str, str]:
    lowered = product.lower()
    if lowered == "postgresql":
        return (
            "Extensible SQL platform with strong growth headroom and rich tooling.",
            "Typically needs more operational discipline than the lightest options.",
        )
    if lowered == "mysql":
        return (
            "Familiar operational model with broad hosting support and large talent pool.",
            "Feature depth and extensibility can feel narrower for advanced workloads.",
        )
    if lowered == "sqlite":
        return (
            "Tiny footprint and simplest deployment story, especially for local or embedded apps.",
            "Write concurrency and multi-node growth are the main limits to validate.",
        )
    if lowered == "react":
        return (
            "Strongest ecosystem and enterprise hiring signal of the frontend options.",
            "Architectural flexibility can create inconsistency without team standards.",
        )
    if lowered == "vue":
        return (
            "Balanced developer experience with approachable conventions and solid ecosystem depth.",
            "Enterprise mindshare and hiring depth are lower than React in many markets.",
        )
    if lowered == "svelte":
        return (
            "Modern and lightweight developer experience with excellent simplicity.",
            "Smaller ecosystem means more validation work for large-enterprise use cases.",
        )
    return (
        f"{product} appears promising for the scenario described in {query}.",
        f"Validate the long-term trade-offs and ecosystem maturity of {product} before deciding.",
    )


def _recommendation(products: list[str], query: str) -> str:
    lowered_products = {product.lower() for product in products}
    if {"postgresql", "mysql", "sqlite"}.intersection(lowered_products):
        return (
            "For a new web application, PostgreSQL is the strongest default when you want long-term flexibility, "
            "MySQL is a good fit when operational familiarity matters most, and SQLite is ideal when the workload "
            "is lightweight or deployment simplicity is the top priority."
        )
    if {"react", "vue", "svelte"}.intersection(lowered_products):
        return (
            "For enterprise apps, React usually wins on ecosystem and hiring depth, Vue is the most balanced choice "
            "for productivity and structure, and Svelte is best when a small, modern stack matters more than ecosystem breadth."
        )
    return (
        "Choose the option whose strengths map most directly to the scenario, then validate the leading contender "
        "with primary-source docs, benchmarks, and operational constraints."
    )


def _mock_report(request: ComparisonReportRequest) -> str:
    products = request.products or _extract_products(request.original_query)
    parsed_results = [(_parse_dimension_name(result), _parse_dimension_notes(result)) for result in request.dimension_results]

    matrix_header = "| Dimension | " + " | ".join(products) + " |"
    matrix_separator = "|" + " --- |" * (len(products) + 1)
    matrix_rows = []
    for dimension, notes in parsed_results:
        row = [dimension]
        for product in products:
            row.append(notes.get(product, "Needs validation."))
        matrix_rows.append("| " + " | ".join(row) + " |")

    pros_cons_rows = []
    for product in products:
        pros, cons = _pros_cons_for(product, request.original_query)
        pros_cons_rows.append(f"| {product} | {pros} | {cons} |")

    return (
        f"# Competitive Analysis Report\n\n"
        f"## Comparison Question\n{request.original_query}\n\n"
        f"## Comparison Matrix\n"
        f"{matrix_header}\n{matrix_separator}\n"
        + "\n".join(matrix_rows)
        + "\n\n"
        + "## Pros / Cons Snapshot\n"
        + "| Product | Advantages | Trade-offs |\n| --- | --- | --- |\n"
        + "\n".join(pros_cons_rows)
        + "\n\n"
        + "## Recommendation\n"
        + _recommendation(products, request.original_query)
        + "\n\n## Dimension Analyst Notes\n"
        + "\n\n".join(request.dimension_results)
    )


def create_comparison_report(ctx: task.ActivityContext, input_: Union[dict[str, Any], ComparisonReportRequest]) -> str:
    request = input_ if isinstance(input_, ComparisonReportRequest) else ComparisonReportRequest.model_validate(input_)
    api_key = os.getenv("OPENAI_API_KEY")
    if not api_key:
        return _mock_report(request)

    products = request.products or _extract_products(request.original_query)
    client = OpenAI(api_key=os.getenv("OPENAI_API_KEY"), base_url=os.getenv("OPENAI_BASE_URL"), max_retries=0)
    prompt = (
        "Create a structured competitive analysis report. Include:\n"
        "1. A short executive summary.\n"
        "2. A markdown comparison matrix with dimensions as rows and products as columns.\n"
        "3. A markdown pros/cons table.\n"
        "4. A clear recommendation tailored to the original query.\n\n"
        f"Original query: {request.original_query}\n"
        f"Products: {', '.join(products)}\n\n"
        "Dimension analyses:\n"
        + "\n\n".join(request.dimension_results)
    )
    response = client.chat.completions.create(
        model=request.model,
        temperature=0.2,
        messages=[
            {
                "role": "system",
                "content": "You are a senior analyst who writes decision-ready competitive analysis reports.",
            },
            {"role": "user", "content": prompt},
        ],
    )
    content = response.choices[0].message.content
    return content.strip() if content else ""
