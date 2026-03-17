from __future__ import annotations

import os
from pathlib import Path

from dotenv import load_dotenv
from durabletask import task
from openai import OpenAI

load_dotenv(Path(__file__).resolve().parents[3] / ".env")

SYSTEM_PROMPT = """You answer questions using only the supplied retrieval context.
Cite which retrieval source informed the answer and explicitly note when context is simulated.
"""


def generate_answer(ctx: task.ActivityContext, context_and_query: dict) -> str:
    """Generate an answer grounded in combined retrieval context."""
    query = context_and_query["query"]
    context = context_and_query["context"]

    client = OpenAI(
        base_url=os.getenv("OPENAI_BASE_URL"),
        api_key=os.getenv("OPENAI_API_KEY"),
        max_retries=0,
    )
    response = client.responses.create(
        model=os.getenv("OPENAI_MODEL", "gpt-5.4"),
        input=[
            {"role": "system", "content": SYSTEM_PROMPT},
            {
                "role": "user",
                "content": (
                    "Use the following simulated retrieval context to answer the user query.\n\n"
                    f"Query:\n{query}\n\n"
                    f"Context:\n{context}\n"
                ),
            },
        ],
    )
    return response.output_text
