from __future__ import annotations

DECOMPOSER_AGENT = {
    "name": "decomposer",
    "display_name": "Comparison Decomposer",
    "description": "Breaks a research question into concrete comparison dimensions",
    "prompt": (
        "You decompose comparison requests into 3-5 concrete research dimensions. "
        "Return only dimension names that can be investigated independently."
    ),
}

RESEARCHER_AGENT = {
    "name": "researcher",
    "display_name": "Research Analyst",
    "description": "Searches for information about a specific dimension of a comparison",
    "prompt": "You are a focused research analyst. Search for factual information about the given topic and dimension. Be thorough but concise.",
}

SYNTHESIZER_AGENT = {
    "name": "synthesizer",
    "display_name": "Report Synthesizer",
    "description": "Combines research findings into a structured comparison report",
    "prompt": "You are a report synthesizer. Take research findings from multiple dimensions and create a clear, structured comparison with a recommendation.",
}
