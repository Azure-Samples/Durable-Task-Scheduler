"""Orchestrations for the arXiv Research Agent using DurableTask SDK.

Orchestrations define the workflow logic that coordinates multiple activities.
They are durable and can survive process restarts, automatically resuming
from their last checkpoint.
"""

import logging
from datetime import timedelta
from typing import Any, Dict, Generator, List, Optional

from durabletask import task

logger = logging.getLogger(__name__)

# Retry policy for arXiv API calls (network-related)
ARXIV_RETRY_POLICY = task.RetryPolicy(
    first_retry_interval=timedelta(seconds=5),
    max_number_of_attempts=3,
    backoff_coefficient=2.0,
    max_retry_interval=timedelta(seconds=30),
    retry_timeout=timedelta(minutes=2)
)

# Retry policy for LLM calls (may have rate limits or transient failures)
LLM_RETRY_POLICY = task.RetryPolicy(
    first_retry_interval=timedelta(seconds=2),
    max_number_of_attempts=3,
    backoff_coefficient=2.0,
    max_retry_interval=timedelta(seconds=30),
    retry_timeout=timedelta(minutes=5)
)


def paper_research_orchestrator(
    ctx: task.OrchestrationContext, workflow_input: Dict[str, Any]
) -> Generator[Any, Any, Dict[str, Any]]:
    """Sub-orchestration: Research papers for a specific query.

    This orchestrator:
    1. Searches arXiv for papers about the query
    2. Analyzes papers and extracts academic insights

    Args:
        ctx: Orchestration context
        workflow_input: Dictionary with main_topic and query

    Yields:
        Activity calls for searching and analyzing

    Returns:
        Analysis result dictionary
    """
    main_topic = workflow_input["main_topic"]
    query = workflow_input["query"]
    
    logger.info(f"Starting paper research for query: {query}")
    
    # Step 1: Search arXiv for papers
    papers = yield ctx.call_activity(
        "search_arxiv_activity",
        input=query,
        retry_policy=ARXIV_RETRY_POLICY
    )
    
    if not papers:
        logger.info(f"No papers found for query: {query}")
        return {
            "query": query,
            "insights": [],
            "relevance_score": 0,
            "summary": "No papers found for this query",
            "key_points": [],
            "research_gaps": [],
            "top_papers": []
        }
    
    logger.info(f"Found {len(papers)} papers, analyzing...")
    
    # Step 2: Analyze papers and extract insights
    analysis = yield ctx.call_activity(
        "analyze_papers_activity",
        input={
            "topic": main_topic,
            "query": query,
            "papers": papers,
        },
        retry_policy=LLM_RETRY_POLICY
    )
    
    return analysis


def _synthesize_and_return(
    ctx: task.OrchestrationContext,
    topic: str,
    all_findings: List[Dict[str, Any]],
    current_iteration: int,
    reason: str,
) -> Generator[Any, Any, Dict[str, Any]]:
    """Helper generator to synthesize findings and return final result.

    Args:
        ctx: Orchestration context
        topic: Research topic
        all_findings: All collected findings
        current_iteration: Current iteration number
        reason: Log message explaining why synthesis is happening

    Yields:
        Activity call for synthesis

    Returns:
        Final research report dictionary
    """
    logger.info(reason)
    final_report = yield ctx.call_activity(
        "synthesize_research_activity",
        input={"topic": topic, "all_findings": all_findings},
        retry_policy=LLM_RETRY_POLICY
    )
    return {
        "topic": topic,
        "iterations": current_iteration,
        "report": final_report,
        "findings_count": len(all_findings)
    }


def arxiv_research_orchestrator(
    ctx: task.OrchestrationContext, workflow_input: Dict[str, Any]
) -> Generator[Any, Any, Optional[Dict[str, Any]]]:
    """Main orchestration: Autonomous arXiv research workflow.

    This orchestrator performs automated academic research using the continue_as_new
    pattern to prevent unbounded history growth:
    1. Executes one research iteration per orchestration instance
    2. Calls continue_as_new with updated state to proceed to next iteration
    3. Returns final result when max iterations reached or early termination

    The continue_as_new pattern resets orchestration history after each iteration,
    making it suitable for long-running research workflows.

    Args:
        ctx: Orchestration context
        workflow_input: Dictionary containing:
            - topic: The research topic (required)
            - max_iterations: Maximum research iterations (default: 3)
            - current_iteration: Current iteration number (default: 0)
            - all_findings: Accumulated findings from previous iterations
            - current_query: Query to use for this iteration

    Yields:
        Sub-orchestration and activity calls

    Returns:
        Final research report dictionary
    """
    # Extract state (supports both initial call and continue_as_new)
    topic = workflow_input["topic"]
    max_iterations = workflow_input.get("max_iterations", 3)
    current_iteration = workflow_input.get("current_iteration", 0)
    all_findings: List[Dict[str, Any]] = workflow_input.get("all_findings", [])
    current_query = workflow_input.get("current_query", topic)

    # Log start of research on first iteration
    if current_iteration == 0:
        logger.info(f"Starting arXiv research for topic: '{topic}'")

    # Check if we've reached max iterations - synthesize and return
    if current_iteration >= max_iterations:
        return (yield from _synthesize_and_return(
            ctx, topic, all_findings, current_iteration,
            "Max iterations reached, synthesizing research report..."
        ))

    # Increment iteration counter
    current_iteration += 1
    logger.info(f"Starting iteration {current_iteration}/{max_iterations}")

    # Research papers for the current query using a sub-orchestration
    analysis = yield ctx.call_sub_orchestrator(
        "paper_research_orchestrator",
        input={"main_topic": topic, "query": current_query}
    )
    all_findings.append(analysis)

    # Decide whether to continue the literature review
    should_continue = yield ctx.call_activity(
        "decide_continuation_activity",
        input={
            "topic": topic,
            "all_findings": all_findings,
            "current_iteration": current_iteration,
            "max_iterations": max_iterations
        },
        retry_policy=LLM_RETRY_POLICY
    )

    if not should_continue:
        return (yield from _synthesize_and_return(
            ctx, topic, all_findings, current_iteration,
            "Concluding research early based on LLM decision"
        ))

    # Generate next research query based on gaps identified
    follow_up_query = yield ctx.call_activity(
        "identify_research_gaps_activity",
        input={
            "topic": topic,
            "current_findings": all_findings,
            "iteration": current_iteration
        },
        retry_policy=LLM_RETRY_POLICY
    )

    if not follow_up_query:
        return (yield from _synthesize_and_return(
            ctx, topic, all_findings, current_iteration,
            "No additional research gaps identified, concluding..."
        ))

    # Continue as new with updated state (resets history)
    logger.info(f"Continuing to next iteration with query: '{follow_up_query}'")
    ctx.continue_as_new({
        "topic": topic,
        "max_iterations": max_iterations,
        "current_iteration": current_iteration,
        "all_findings": all_findings,
        "current_query": follow_up_query
    })
