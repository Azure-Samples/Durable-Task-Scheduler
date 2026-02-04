"""Activities for the arXiv Research Agent using DurableTask SDK.

Activities are the individual units of work that perform specific tasks
within an orchestration. They are durable and can be retried on failure.
"""

import json
import logging
from typing import Any, Dict, List, Optional, cast

from durabletask import task

from .arxiv_api import search_arxiv
from .llm import call_llm, parse_json_response

logger = logging.getLogger(__name__)

# Configuration
MAX_PAPERS_TO_ANALYZE = 15
DEFAULT_RELEVANCE_SCORE = 5


# =============================================================================
# Helper Functions
# =============================================================================


def _format_paper_for_prompt(index: int, paper: Dict[str, Any]) -> str:
    """Format a single paper for inclusion in an LLM prompt."""
    title = paper.get("title", "No title")
    arxiv_id = paper.get("arxiv_id", "")
    authors_list = paper.get("authors", [])
    authors = ", ".join(authors_list[:3])
    if len(authors_list) > 3:
        authors += " et al."
    summary = paper.get("summary", "")[:500]
    categories = ", ".join(paper.get("categories", [])[:3])
    published = paper.get("published", "")[:10]
    abs_url = paper.get("abs_url", "")

    return (
        f"Paper {index}:\n"
        f"  Title: {title}\n"
        f"  arXiv ID: {arxiv_id}\n"
        f"  Authors: {authors}\n"
        f"  Published: {published}\n"
        f"  Categories: {categories}\n"
        f"  Abstract: {summary}...\n"
        f"  URL: {abs_url}\n"
    )


def _format_finding_for_prompt(finding: Dict[str, Any], include_relevance: bool = False) -> str:
    """Format a research finding for inclusion in an LLM prompt."""
    lines = [
        f"Query: {finding.get('query', 'Unknown')}",
        f"Summary: {finding.get('summary', 'No summary')}",
    ]
    if include_relevance:
        relevance = finding.get("relevance_score", DEFAULT_RELEVANCE_SCORE)
        lines.append(f"Relevance: {relevance}/10")
        lines.append(f"Papers found: {len(finding.get('top_papers', []))}")
    else:
        lines.append(f"Key insights: {finding.get('insights', [])}")
        lines.append(f"Research gaps: {finding.get('research_gaps', [])}")
    return "\n".join(lines)


def _extract_paper_metadata(paper: Dict[str, Any]) -> Dict[str, Any]:
    """Extract standardized metadata from a paper dictionary."""
    return {
        "arxiv_id": paper.get("arxiv_id", ""),
        "title": paper.get("title", ""),
        "authors": paper.get("authors", []),
        "summary": paper.get("summary", ""),
        "published": paper.get("published", ""),
        "primary_category": paper.get("primary_category", ""),
        "categories": paper.get("categories", []),
        "pdf_url": paper.get("pdf_url", ""),
        "abs_url": paper.get("abs_url", ""),
        "comment": paper.get("comment", ""),
        "journal_ref": paper.get("journal_ref", ""),
        "doi": paper.get("doi", ""),
    }


# =============================================================================
# Activities
# =============================================================================


def search_arxiv_activity(ctx: task.ActivityContext, query: str) -> List[Dict[str, Any]]:
    """Activity: Search arXiv for papers about a topic.
    
    Args:
        ctx: Activity context
        query: Search query string
        
    Returns:
        List of paper dictionaries
    """
    logger.info(f"Searching arXiv for: {query}")
    papers = search_arxiv(query, max_results=30)
    logger.info(f"Found {len(papers)} papers")
    return papers


def analyze_papers_activity(ctx: task.ActivityContext, activity_input: Dict[str, Any]) -> Dict[str, Any]:
    """Activity: Analyze arXiv papers and extract academic insights using LLM.

    Args:
        ctx: Activity context
        activity_input: Dictionary with topic, query, and papers

    Returns:
        Analysis result as dictionary
    """
    topic = activity_input["topic"]
    query = activity_input["query"]
    papers = activity_input["papers"]

    logger.info(f"Analyzing papers for topic: {topic}, query: {query}")

    # Format papers for the LLM prompt
    papers_to_analyze = papers[:MAX_PAPERS_TO_ANALYZE]
    papers_lines = [_format_paper_for_prompt(i + 1, p) for i, p in enumerate(papers_to_analyze)]
    top_papers = [_extract_paper_metadata(p) for p in papers_to_analyze]
    papers_text = "\n".join(papers_lines)
    
    prompt = f"""
    You are a research agent evaluating arXiv papers for: {topic}

    Query used: {query}

    Papers found:
    {papers_text}

    Provide a DETAILED analysis of these research papers. Focus on:
    - Key research contributions, methodologies, and techniques
    - Specific experimental results, metrics, or benchmarks
    - Novel approaches, architectures, or algorithms proposed
    - Connections between papers and emerging research themes
    - Identified research gaps or open problems
    - Practical applications and potential impact
    - Most influential or highly relevant papers for this topic

    Return JSON with:
    - "insights": String array of specific, technical insights from the papers
    - "relevance_score": Number 1-10 (how relevant are these papers to the research topic)
    - "summary": Brief summary of the research landscape
    - "key_points": Array of most important research findings
    - "research_gaps": Array of identified gaps or future research directions
    """
    
    messages = [
        {
            "role": "system",
            "content": "You are a research evaluation agent. Analyze arXiv papers and provide structured insights in JSON format. Focus on technical depth and research value.",
        },
        {"role": "user", "content": prompt},
    ]
    
    response = call_llm(messages)
    try:
        evaluation_dict: Dict[str, Any] = parse_json_response(response)
    except json.JSONDecodeError as e:
        logger.warning(f"Failed to parse evaluation JSON: {e}, using defaults")
        evaluation_dict = {
            "insights": [],
            "relevance_score": DEFAULT_RELEVANCE_SCORE,
            "summary": "Failed to parse LLM response",
            "key_points": [],
            "research_gaps": [],
        }
    evaluation_dict["query"] = query
    evaluation_dict["top_papers"] = top_papers

    return evaluation_dict


def identify_research_gaps_activity(ctx: task.ActivityContext, activity_input: Dict[str, Any]) -> Optional[str]:
    """Activity: Identify research gaps and generate follow-up queries.

    Args:
        ctx: Activity context
        activity_input: Dictionary with topic, current_findings, and iteration

    Returns:
        Next query to research, or None if no gaps identified
    """
    topic = activity_input["topic"]
    current_findings = activity_input["current_findings"]
    iteration = activity_input["iteration"]

    logger.info(f"Identifying research gaps for iteration {iteration}")

    findings_lines = [_format_finding_for_prompt(f) for f in current_findings]
    findings_summary = "\n\n".join(findings_lines)
    
    prompt = f"""
    You are a research agent investigating: {topic}
    
    This is iteration {iteration} of your research.
    
    Current findings:
    {findings_summary}
    
    Generate 2-4 SHORT KEYWORD-BASED search queries for arXiv that explore DIVERSE aspects of {topic}.
    
    CRITICAL RULES:
    1. Use SHORT keywords (2-5 words max) - NOT long sentences
    2. Focus on DIFFERENT aspects, methodologies, or applications
    3. Use terms that appear in actual arXiv paper titles
    4. Consider exploring identified research gaps
    5. Avoid repeating previous queries
    
    GOOD examples: ["transformer attention mechanisms", "neural network pruning", "federated learning privacy"]
    BAD examples: ["What are the latest advances in transformer-based architectures for natural language processing?"]
    
    Return only a JSON array of SHORT keyword queries: ["query1", "query2", "query3"]
    """
    
    messages = [
        {
            "role": "system",
            "content": "You are a research agent. Generate focused follow-up queries for arXiv search. Return only JSON array.",
        },
        {"role": "user", "content": prompt},
    ]
    
    response = call_llm(messages)
    try:
        parsed: Any = parse_json_response(response)
        if not isinstance(parsed, list):
            return None
        parsed_list = cast(List[Any], parsed)
        queries: List[str] = [str(q) for q in parsed_list]
    except json.JSONDecodeError as e:
        logger.warning(f"Failed to parse follow-up queries JSON: {e}")
        return None
    if len(queries) > 0:
        return queries[0]
    return None


def decide_continuation_activity(ctx: task.ActivityContext, activity_input: Dict[str, Any]) -> bool:
    """Activity: Decide whether to continue the literature review.

    Args:
        ctx: Activity context
        activity_input: Dictionary with topic, all_findings, current_iteration, max_iterations

    Returns:
        True if literature review should continue, False otherwise
    """
    topic = activity_input["topic"]
    all_findings = activity_input["all_findings"]
    current_iteration = activity_input["current_iteration"]
    max_iterations = activity_input["max_iterations"]

    logger.info(f"Deciding whether to continue (iteration {current_iteration}/{max_iterations})")

    if current_iteration >= max_iterations:
        return False

    # Calculate average relevance and format findings
    findings_lines = [_format_finding_for_prompt(f, include_relevance=True) for f in all_findings]
    findings_summary = "\n\n".join(findings_lines)
    total_relevance = sum(f.get("relevance_score", DEFAULT_RELEVANCE_SCORE) for f in all_findings)

    avg_relevance = total_relevance / len(all_findings) if all_findings else 0
    
    prompt = f"""
    You are a research agent investigating: {topic}
    
    Current iteration: {current_iteration}/{max_iterations}
    
    Findings so far:
    {findings_summary}
    
    Average relevance score: {avg_relevance:.1f}/10
    
    Decide whether to continue research or conclude. Continue if:
    1. Current iteration is less than 75% of max_iterations
    2. Average relevance is above 6.0 and there are likely unexplored aspects
    3. Recent queries found significant new papers with valuable insights
    4. There are identified research gaps worth exploring
    
    Only stop early if:
    - Average relevance is below 5.0 for multiple iterations
    - No new meaningful information in the last 2 iterations
    - The topic has been comprehensively covered
    
    Return JSON with:
    - "should_continue": boolean
    """
    
    messages = [
        {
            "role": "system",
            "content": "You are a research decision agent. Evaluate research completeness and decide whether to continue. Return JSON.",
        },
        {"role": "user", "content": prompt},
    ]
    
    raw_response = call_llm(messages)
    try:
        json_response: Dict[str, Any] = parse_json_response(raw_response)
    except json.JSONDecodeError as e:
        logger.warning(f"Failed to parse should_continue JSON: {e}, defaulting to False")
        json_response = {}
    result: bool = json_response.get("should_continue", False)
    return result


def synthesize_research_activity(ctx: task.ActivityContext, activity_input: Dict[str, Any]) -> str:
    """Activity: Synthesize all research findings into a comprehensive report.

    Args:
        ctx: Activity context
        activity_input: Dictionary with topic and all_findings

    Returns:
        Final research report as markdown string
    """
    topic = activity_input["topic"]
    all_findings = activity_input["all_findings"]

    logger.info(f"Synthesizing findings for topic: {topic}")

    # Build a simple summary of findings for the LLM
    findings_summary = _build_findings_summary(all_findings)
    papers_list = _build_papers_list(all_findings)

    prompt = f"""Write a research summary about: {topic}

Based on these findings:
{findings_summary}

Key papers found:
{papers_list}

Write a clear, well-organized research summary in markdown format with these sections:
- **Summary**: 2-3 sentence overview
- **Key Findings**: Main discoveries and insights (bullet points)
- **Methods & Approaches**: Common techniques used in the research
- **Open Questions**: Gaps and future research directions
- **References**: List the most relevant papers with their arXiv links

Keep it concise and readable. Use markdown formatting."""

    messages = [
        {
            "role": "system",
            "content": "You are a research assistant. Write clear, concise research summaries in markdown format. Do not wrap the response in JSON.",
        },
        {"role": "user", "content": prompt},
    ]

    # Request plain text output (not JSON) for cleaner formatting
    return call_llm(messages, max_tokens=2500, json_output=False)


def _build_findings_summary(all_findings: List[Dict[str, Any]]) -> str:
    """Build a concise summary of all research findings."""
    lines: List[str] = []
    for i, finding in enumerate(all_findings, 1):
        query = finding.get("query", "Unknown")
        summary = finding.get("summary", "No summary")
        key_points = finding.get("key_points", [])

        lines.append(f"Query {i}: {query}")
        lines.append(f"  Summary: {summary}")
        if key_points:
            points_str = "; ".join(str(p) for p in key_points[:3])
            lines.append(f"  Key points: {points_str}")
        lines.append("")

    return "\n".join(lines)


def _build_papers_list(all_findings: List[Dict[str, Any]]) -> str:
    """Build a deduplicated list of papers with links."""
    seen_ids: set[str] = set()
    papers: List[str] = []

    for finding in all_findings:
        for paper in finding.get("top_papers", [])[:5]:  # Limit papers per finding
            arxiv_id = paper.get("arxiv_id", "")
            if arxiv_id and arxiv_id not in seen_ids:
                seen_ids.add(arxiv_id)
                title = paper.get("title", "Unknown")
                url = paper.get("abs_url", f"https://arxiv.org/abs/{arxiv_id}")
                authors = paper.get("authors", [])
                author_str = authors[0] if authors else "Unknown"
                if len(authors) > 1:
                    author_str += " et al."
                papers.append(f"- {title} ({author_str}) - {url}")

            if len(papers) >= 15:  # Cap total papers
                break
        if len(papers) >= 15:
            break

    return "\n".join(papers) if papers else "No papers found"
