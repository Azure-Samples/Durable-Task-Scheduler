"""Tests for activities."""

import pytest
from unittest.mock import patch, Mock

from arxiv_research_agent.activities import (
    search_arxiv_activity,
    analyze_papers_activity,
    identify_research_gaps_activity,
    decide_continuation_activity,
    synthesize_research_activity,
)


class TestSearchArxivActivity:
    """Tests for search_arxiv_activity."""

    @patch("arxiv_research_agent.activities.search_arxiv")
    def test_search_returns_papers(self, mock_search, mock_activity_context, sample_papers):
        """Test that activity returns papers from API."""
        mock_search.return_value = sample_papers
        
        result = search_arxiv_activity(mock_activity_context, "deep learning")
        
        assert result == sample_papers
        mock_search.assert_called_once_with("deep learning", max_results=30)

    @patch("arxiv_research_agent.activities.search_arxiv")
    def test_search_empty_results(self, mock_search, mock_activity_context):
        """Test that activity handles empty results."""
        mock_search.return_value = []
        
        result = search_arxiv_activity(mock_activity_context, "nonexistent topic xyz")
        
        assert result == []


class TestAnalyzePapersActivity:
    """Tests for analyze_papers_activity."""

    @patch("arxiv_research_agent.activities.call_llm")
    @patch("arxiv_research_agent.activities.parse_json_response")
    def test_analyze_returns_dict(
        self,
        mock_parse,
        mock_llm,
        mock_activity_context,
        sample_papers
    ):
        """Test that activity returns analysis dictionary."""
        mock_llm.return_value = '{"insights": [], "relevance_score": 7, "summary": "test", "key_points": [], "research_gaps": []}'
        mock_parse.return_value = {
            "insights": [],
            "relevance_score": 7,
            "summary": "test",
            "key_points": [],
            "research_gaps": []
        }
        
        result = analyze_papers_activity(
            mock_activity_context,
            {
                "topic": "deep learning",
                "query": "transformer attention",
                "papers": sample_papers,
            }
        )
        
        assert "query" in result
        assert "top_papers" in result
        assert result["query"] == "transformer attention"

    @patch("arxiv_research_agent.activities.call_llm")
    @patch("arxiv_research_agent.activities.parse_json_response")
    def test_analyze_handles_empty_papers(
        self,
        mock_parse,
        mock_llm,
        mock_activity_context
    ):
        """Test that activity handles empty papers list."""
        mock_llm.return_value = '{"insights": [], "relevance_score": 5, "summary": "test", "key_points": [], "research_gaps": []}'
        mock_parse.return_value = {
            "insights": [],
            "relevance_score": 5,
            "summary": "test",
            "key_points": [],
            "research_gaps": []
        }
        
        result = analyze_papers_activity(
            mock_activity_context,
            {
                "topic": "deep learning",
                "query": "deep learning",
                "papers": []
            }
        )
        
        assert result is not None


class TestIdentifyResearchGapsActivity:
    """Tests for identify_research_gaps_activity."""

    @patch("arxiv_research_agent.activities.call_llm")
    @patch("arxiv_research_agent.activities.parse_json_response")
    def test_identify_returns_query(
        self,
        mock_parse,
        mock_llm,
        mock_activity_context,
        sample_evaluation_result
    ):
        """Test that activity returns a follow-up query."""
        mock_llm.return_value = '["transformer attention", "neural network optimization"]'
        mock_parse.return_value = ["transformer attention", "neural network optimization"]
        
        result = identify_research_gaps_activity(
            mock_activity_context,
            {
                "topic": "deep learning",
                "current_findings": [sample_evaluation_result],
                "iteration": 1
            }
        )
        
        assert result == "transformer attention"

    @patch("arxiv_research_agent.activities.call_llm")
    @patch("arxiv_research_agent.activities.parse_json_response")
    def test_identify_handles_empty_array(
        self,
        mock_parse,
        mock_llm,
        mock_activity_context,
        sample_evaluation_result
    ):
        """Test that activity handles empty query array."""
        mock_llm.return_value = '[]'
        mock_parse.return_value = []
        
        result = identify_research_gaps_activity(
            mock_activity_context,
            {
                "topic": "deep learning",
                "current_findings": [sample_evaluation_result],
                "iteration": 1
            }
        )
        
        assert result is None

    @patch("arxiv_research_agent.activities.call_llm")
    @patch("arxiv_research_agent.activities.parse_json_response")
    def test_identify_handles_non_list(
        self,
        mock_parse,
        mock_llm,
        mock_activity_context,
        sample_evaluation_result
    ):
        """Test that activity handles non-list response."""
        mock_llm.return_value = '"single query"'
        mock_parse.return_value = "single query"
        
        result = identify_research_gaps_activity(
            mock_activity_context,
            {
                "topic": "deep learning",
                "current_findings": [sample_evaluation_result],
                "iteration": 1
            }
        )
        
        assert result is None


class TestDecideContinuationActivity:
    """Tests for decide_continuation_activity."""

    @patch("arxiv_research_agent.activities.call_llm")
    @patch("arxiv_research_agent.activities.parse_json_response")
    def test_decide_continue_true(
        self,
        mock_parse,
        mock_llm,
        mock_activity_context,
        sample_evaluation_result
    ):
        """Test that activity returns True when should continue."""
        mock_llm.return_value = '{"should_continue": true}'
        mock_parse.return_value = {"should_continue": True}
        
        result = decide_continuation_activity(
            mock_activity_context,
            {
                "topic": "deep learning",
                "all_findings": [sample_evaluation_result],
                "current_iteration": 1,
                "max_iterations": 3
            }
        )
        
        assert result is True

    @patch("arxiv_research_agent.activities.call_llm")
    @patch("arxiv_research_agent.activities.parse_json_response")
    def test_decide_continue_false(
        self,
        mock_parse,
        mock_llm,
        mock_activity_context,
        sample_evaluation_result
    ):
        """Test that activity returns False when should stop."""
        mock_llm.return_value = '{"should_continue": false}'
        mock_parse.return_value = {"should_continue": False}
        
        result = decide_continuation_activity(
            mock_activity_context,
            {
                "topic": "deep learning",
                "all_findings": [sample_evaluation_result],
                "current_iteration": 2,
                "max_iterations": 3
            }
        )
        
        assert result is False

    def test_decide_max_iterations_reached(
        self,
        mock_activity_context,
        sample_evaluation_result
    ):
        """Test that activity returns False when max iterations reached."""
        result = decide_continuation_activity(
            mock_activity_context,
            {
                "topic": "deep learning",
                "all_findings": [sample_evaluation_result],
                "current_iteration": 3,
                "max_iterations": 3
            }
        )
        
        assert result is False


class TestSynthesizeResearchActivity:
    """Tests for synthesize_research_activity."""

    @patch("arxiv_research_agent.activities.call_llm")
    def test_synthesize_returns_report(
        self,
        mock_llm,
        mock_activity_context,
        sample_evaluation_result
    ):
        """Test that activity returns the LLM response as markdown."""
        expected_report = "## Summary\nThis is the research summary."
        mock_llm.return_value = expected_report

        result = synthesize_research_activity(
            mock_activity_context,
            {
                "topic": "deep learning",
                "all_findings": [sample_evaluation_result]
            }
        )

        assert result == expected_report
        # Verify json_output=False is passed for plain text output
        mock_llm.assert_called_once()
        call_kwargs = mock_llm.call_args
        assert call_kwargs.kwargs.get("json_output") is False

    @patch("arxiv_research_agent.activities.call_llm")
    def test_synthesize_includes_paper_links(
        self,
        mock_llm,
        mock_activity_context,
        sample_evaluation_result
    ):
        """Test that activity passes paper information to LLM."""
        mock_llm.return_value = "Report with papers"

        synthesize_research_activity(
            mock_activity_context,
            {
                "topic": "deep learning",
                "all_findings": [sample_evaluation_result]
            }
        )

        # Check that call_llm was called with paper info in the prompt
        call_args = mock_llm.call_args
        messages = call_args.args[0]
        user_message = messages[1]["content"]
        assert "deep learning" in user_message
