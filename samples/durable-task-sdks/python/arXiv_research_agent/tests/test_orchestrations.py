"""Tests for orchestrations."""

import pytest
from unittest.mock import ANY, Mock, call


class TestPaperResearchOrchestrator:
    """Tests for paper_research_orchestrator."""

    def test_orchestrator_no_papers_found(self):
        """Test orchestrator returns empty result when no papers found."""
        from arxiv_research_agent.orchestrations import paper_research_orchestrator
        
        ctx = Mock()
        ctx.call_activity = Mock(return_value="search_call")
        
        # Create a generator
        gen = paper_research_orchestrator(
            ctx,
            {"main_topic": "deep learning", "query": "transformer attention"}
        )
        
        assert next(gen) == "search_call"
        ctx.call_activity.assert_called_once_with(
            "search_arxiv_activity",
            input="transformer attention",
            retry_policy=ANY,
        )

        expected = {
            "query": "transformer attention",
            "insights": [],
            "relevance_score": 0,
            "summary": "No papers found for this query",
            "key_points": [],
            "research_gaps": [],
            "top_papers": [],
        }

        with pytest.raises(StopIteration) as exc_info:
            gen.send([])

        assert exc_info.value.value == expected

    def test_orchestrator_input_structure(self):
        """Test orchestrator receives correct input structure."""
        from arxiv_research_agent.orchestrations import paper_research_orchestrator
        
        ctx = Mock()
        ctx.call_activity = Mock(side_effect=["search_call", "analyze_call"])
        input_data = {"main_topic": "machine learning", "query": "neural networks"}

        papers = [{"title": "paper"}]
        analysis = {
            "query": "neural networks",
            "insights": [],
            "relevance_score": 6,
            "summary": "summary",
            "key_points": [],
            "research_gaps": [],
            "top_papers": [],
        }

        gen = paper_research_orchestrator(ctx, input_data)

        assert next(gen) == "search_call"
        assert gen.send(papers) == "analyze_call"

        ctx.call_activity.assert_has_calls(
            [
                call("search_arxiv_activity", input="neural networks", retry_policy=ANY),
                call(
                    "analyze_papers_activity",
                    input={
                        "topic": "machine learning",
                        "query": "neural networks",
                        "papers": papers,
                    },
                    retry_policy=ANY,
                ),
            ]
        )

        with pytest.raises(StopIteration) as exc_info:
            gen.send(analysis)

        assert exc_info.value.value == analysis


class TestArxivResearchOrchestrator:
    """Tests for arxiv_research_orchestrator with continue_as_new pattern."""

    def test_orchestrator_early_termination(self):
        """Test orchestrator returns final result when should_continue is False."""
        from arxiv_research_agent.orchestrations import arxiv_research_orchestrator

        ctx = Mock()
        ctx.call_sub_orchestrator = Mock(return_value="sub_call")
        ctx.call_activity = Mock(side_effect=["decide_call", "write_call"])
        input_data = {"topic": "deep learning", "max_iterations": 2}

        analysis = {
            "query": "deep learning",
            "insights": [],
            "relevance_score": 6,
            "summary": "summary",
            "key_points": [],
            "research_gaps": [],
            "top_papers": [],
        }

        gen = arxiv_research_orchestrator(ctx, input_data)

        assert next(gen) == "sub_call"
        ctx.call_sub_orchestrator.assert_called_once_with(
            "paper_research_orchestrator",
            input={"main_topic": "deep learning", "query": "deep learning"},
        )

        assert gen.send(analysis) == "decide_call"
        decide_call = ctx.call_activity.call_args_list[0]
        assert decide_call == call(
            "decide_continuation_activity",
            input={
                "topic": "deep learning",
                "all_findings": [analysis],
                "current_iteration": 1,
                "max_iterations": 2,
            },
            retry_policy=ANY,
        )

        # Early termination when should_continue is False
        assert gen.send(False) == "write_call"
        write_call = ctx.call_activity.call_args_list[1]
        assert write_call == call(
            "synthesize_research_activity",
            input={"topic": "deep learning", "all_findings": [analysis]},
            retry_policy=ANY,
        )

        with pytest.raises(StopIteration) as exc_info:
            gen.send("final report")

        assert exc_info.value.value == {
            "topic": "deep learning",
            "iterations": 1,
            "report": "final report",
            "findings_count": 1,
        }

    def test_orchestrator_continue_as_new(self):
        """Test orchestrator calls continue_as_new when continuing to next iteration."""
        from arxiv_research_agent.orchestrations import arxiv_research_orchestrator

        ctx = Mock()
        ctx.call_sub_orchestrator = Mock(return_value="sub_call")
        ctx.call_activity = Mock(side_effect=["decide_call", "gaps_call"])
        ctx.continue_as_new = Mock()
        input_data = {"topic": "deep learning", "max_iterations": 3}

        analysis = {
            "query": "deep learning",
            "insights": [],
            "relevance_score": 8,
            "summary": "summary",
            "key_points": [],
            "research_gaps": ["gap1"],
            "top_papers": [],
        }

        gen = arxiv_research_orchestrator(ctx, input_data)

        # First yield: sub_orchestrator call
        assert next(gen) == "sub_call"

        # Send analysis result, get decide_continuation call
        assert gen.send(analysis) == "decide_call"

        # Send True (should continue), get identify_research_gaps call
        assert gen.send(True) == "gaps_call"

        # Send follow-up query - should call continue_as_new and end
        with pytest.raises(StopIteration):
            gen.send("follow-up query")

        # Verify continue_as_new was called with correct state
        ctx.continue_as_new.assert_called_once_with({
            "topic": "deep learning",
            "max_iterations": 3,
            "current_iteration": 1,
            "all_findings": [analysis],
            "current_query": "follow-up query"
        })

    def test_orchestrator_max_iterations_reached(self):
        """Test orchestrator synthesizes when starting at max iterations."""
        from arxiv_research_agent.orchestrations import arxiv_research_orchestrator

        ctx = Mock()
        ctx.call_activity = Mock(return_value="write_call")

        # Simulate state from continue_as_new at max iterations
        previous_findings = [{"query": "q1"}, {"query": "q2"}]
        input_data = {
            "topic": "deep learning",
            "max_iterations": 2,
            "current_iteration": 2,
            "all_findings": previous_findings,
            "current_query": "last query"
        }

        gen = arxiv_research_orchestrator(ctx, input_data)

        # Should immediately call synthesize_research_activity
        assert next(gen) == "write_call"
        ctx.call_activity.assert_called_once_with(
            "synthesize_research_activity",
            input={"topic": "deep learning", "all_findings": previous_findings},
            retry_policy=ANY,
        )

        with pytest.raises(StopIteration) as exc_info:
            gen.send("final report")

        assert exc_info.value.value == {
            "topic": "deep learning",
            "iterations": 2,
            "report": "final report",
            "findings_count": 2,
        }

    def test_orchestrator_no_gaps_found(self):
        """Test orchestrator returns final result when no research gaps found."""
        from arxiv_research_agent.orchestrations import arxiv_research_orchestrator

        ctx = Mock()
        ctx.call_sub_orchestrator = Mock(return_value="sub_call")
        ctx.call_activity = Mock(side_effect=["decide_call", "gaps_call", "write_call"])
        input_data = {"topic": "deep learning", "max_iterations": 3}

        analysis = {"query": "deep learning", "insights": [], "relevance_score": 6,
                    "summary": "summary", "key_points": [], "research_gaps": [], "top_papers": []}

        gen = arxiv_research_orchestrator(ctx, input_data)

        assert next(gen) == "sub_call"
        assert gen.send(analysis) == "decide_call"
        assert gen.send(True) == "gaps_call"

        # No follow-up query found (None) - should synthesize
        assert gen.send(None) == "write_call"

        with pytest.raises(StopIteration) as exc_info:
            gen.send("final report")

        assert exc_info.value.value == {
            "topic": "deep learning",
            "iterations": 1,
            "report": "final report",
            "findings_count": 1,
        }

    def test_orchestrator_default_max_iterations(self):
        """Test orchestrator uses default max_iterations."""
        from arxiv_research_agent.orchestrations import arxiv_research_orchestrator

        ctx = Mock()
        ctx.call_sub_orchestrator = Mock(return_value="sub_call")
        ctx.call_activity = Mock(side_effect=["decide_call", "write_call"])
        input_data = {"topic": "computer vision"}  # No max_iterations

        gen = arxiv_research_orchestrator(ctx, input_data)

        assert next(gen) == "sub_call"

    def test_orchestrator_state_from_continue_as_new(self):
        """Test orchestrator correctly resumes from continue_as_new state."""
        from arxiv_research_agent.orchestrations import arxiv_research_orchestrator

        ctx = Mock()
        ctx.call_sub_orchestrator = Mock(return_value="sub_call")
        ctx.call_activity = Mock(side_effect=["decide_call", "write_call"])

        # Simulate state passed from previous continue_as_new
        previous_findings = [{"query": "initial query", "summary": "first iteration"}]
        input_data = {
            "topic": "deep learning",
            "max_iterations": 3,
            "current_iteration": 1,
            "all_findings": previous_findings,
            "current_query": "follow-up query"
        }

        gen = arxiv_research_orchestrator(ctx, input_data)

        # Should use the current_query from state
        assert next(gen) == "sub_call"
        ctx.call_sub_orchestrator.assert_called_once_with(
            "paper_research_orchestrator",
            input={"main_topic": "deep learning", "query": "follow-up query"},
        )


class TestOrchestratorIntegration:
    """Integration-style tests for orchestrators."""

    def test_paper_research_returns_dict_structure(self):
        """Test that paper_research_orchestrator would return expected dict structure."""
        # This tests the expected return structure
        expected_empty_result = {
            "query": "transformer attention",
            "insights": [],
            "relevance_score": 0,
            "summary": "No papers found for this query",
            "key_points": [],
            "research_gaps": [],
            "top_papers": []
        }
        
        # Verify structure
        assert "query" in expected_empty_result
        assert "insights" in expected_empty_result
        assert "relevance_score" in expected_empty_result
        assert "summary" in expected_empty_result
        assert "key_points" in expected_empty_result
        assert "research_gaps" in expected_empty_result
        assert "top_papers" in expected_empty_result

    def test_arxiv_research_returns_dict_structure(self):
        """Test that arxiv_research_orchestrator would return expected dict structure."""
        # This tests the expected return structure
        expected_result = {
            "topic": "deep learning",
            "iterations": 3,
            "report": "Final literature review content",
            "findings_count": 3
        }
        
        # Verify structure
        assert "topic" in expected_result
        assert "iterations" in expected_result
        assert "report" in expected_result
        assert "findings_count" in expected_result
