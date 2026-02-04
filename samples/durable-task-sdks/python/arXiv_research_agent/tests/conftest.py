"""Pytest configuration and fixtures for tests."""

import pytest
from unittest.mock import Mock, patch, MagicMock
import os


@pytest.fixture(autouse=True)
def mock_env_vars(monkeypatch):
    """Set default environment variables for tests."""
    monkeypatch.setenv("AZURE_OPENAI_ENDPOINT", "https://test.openai.azure.com")
    monkeypatch.setenv("AZURE_OPENAI_API_KEY", "test-api-key")
    monkeypatch.setenv("AZURE_OPENAI_DEPLOYMENT", "gpt-4o-mini")
    monkeypatch.setenv("ENDPOINT", "http://localhost:8080")
    monkeypatch.setenv("TASKHUB", "default")


@pytest.fixture
def mock_openai_client():
    """Mock OpenAI client for LLM tests (Responses API)."""
    with patch("arxiv_research_agent.llm._client") as mock_client:
        mock_response = Mock()
        mock_response.output_text = '{"test": "response"}'
        mock_client.responses.create.return_value = mock_response
        yield mock_client


@pytest.fixture
def sample_papers():
    """Sample arXiv papers for testing."""
    return [
        {
            "arxiv_id": "2301.12345v1",
            "title": "Deep Learning for Natural Language Processing",
            "summary": "This paper presents a novel approach to NLP using transformer architectures.",
            "authors": ["John Smith", "Jane Doe", "Bob Wilson"],
            "published": "2023-01-15T00:00:00Z",
            "updated": "2023-01-20T00:00:00Z",
            "categories": ["cs.CL", "cs.AI", "cs.LG"],
            "primary_category": "cs.CL",
            "pdf_url": "https://arxiv.org/pdf/2301.12345v1",
            "abs_url": "https://arxiv.org/abs/2301.12345v1",
            "comment": "12 pages, 5 figures",
            "journal_ref": "",
            "doi": "",
        },
        {
            "arxiv_id": "2301.67890v2",
            "title": "Reinforcement Learning in Robotics",
            "summary": "We explore the application of RL techniques in robotic manipulation tasks.",
            "authors": ["Alice Brown", "Charlie Davis"],
            "published": "2023-01-10T00:00:00Z",
            "updated": "2023-02-15T00:00:00Z",
            "categories": ["cs.RO", "cs.AI"],
            "primary_category": "cs.RO",
            "pdf_url": "https://arxiv.org/pdf/2301.67890v2",
            "abs_url": "https://arxiv.org/abs/2301.67890v2",
            "comment": "Accepted at ICRA 2023",
            "journal_ref": "ICRA 2023",
            "doi": "10.1109/ICRA.2023.12345",
        },
    ]


@pytest.fixture
def sample_evaluation_result():
    """Sample evaluation result for testing."""
    return {
        "query": "deep learning NLP",
        "insights": ["Transformers outperform RNNs", "Attention mechanisms are key"],
        "relevance_score": 8,
        "summary": "Research in NLP has been transformed by deep learning approaches",
        "key_points": ["Transformer architecture", "Pre-training benefits"],
        "research_gaps": ["Efficiency improvements needed", "Multilingual models underexplored"],
        "top_papers": [],
    }


@pytest.fixture
def mock_httpx_client():
    """Mock httpx client for API tests."""
    with patch("arxiv_research_agent.arxiv_api._get_client") as mock_get_client:
        mock_client = MagicMock()
        mock_get_client.return_value = mock_client
        yield mock_client


@pytest.fixture
def mock_activity_context():
    """Mock DurableTask activity context."""
    return Mock()


@pytest.fixture
def mock_orchestration_context():
    """Mock DurableTask orchestration context."""
    ctx = Mock()
    ctx.call_activity = Mock()
    ctx.call_sub_orchestrator = Mock()
    return ctx
