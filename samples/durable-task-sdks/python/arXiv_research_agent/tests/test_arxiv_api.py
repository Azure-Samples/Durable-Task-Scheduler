"""Tests for arXiv API utilities."""

import pytest
from unittest.mock import Mock, MagicMock, patch
import httpx

from arxiv_research_agent.arxiv_api import (
    search_arxiv,
    ArxivAPIError,
    ARXIV_API_URL,
    API_TIMEOUT,
)


# Sample XML response for testing
SAMPLE_ARXIV_XML = """<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:arxiv="http://arxiv.org/schemas/atom">
  <entry>
    <id>http://arxiv.org/abs/2301.12345v1</id>
    <title>Test Paper About Machine Learning</title>
    <summary>This paper presents a novel approach to machine learning.</summary>
    <author><name>John Smith</name></author>
    <author><name>Jane Doe</name></author>
    <published>2023-01-15T00:00:00Z</published>
    <updated>2023-01-20T00:00:00Z</updated>
    <category term="cs.LG"/>
    <category term="cs.AI"/>
    <arxiv:primary_category term="cs.LG"/>
    <link href="http://arxiv.org/abs/2301.12345v1" rel="alternate" type="text/html"/>
    <link href="http://arxiv.org/pdf/2301.12345v1" title="pdf" type="application/pdf"/>
    <arxiv:comment>12 pages, 5 figures</arxiv:comment>
  </entry>
</feed>"""

EMPTY_ARXIV_XML = """<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
</feed>"""


class TestSearchArxiv:
    """Tests for search_arxiv function."""

    def test_search_success(self, mock_httpx_client):
        """Test successful search."""
        mock_response = Mock()
        mock_response.text = SAMPLE_ARXIV_XML
        mock_response.raise_for_status = Mock()
        mock_httpx_client.get.return_value = mock_response
        
        result = search_arxiv("machine learning")
        
        assert len(result) == 1
        assert result[0]["title"] == "Test Paper About Machine Learning"
        assert result[0]["arxiv_id"] == "2301.12345v1"
        assert "John Smith" in result[0]["authors"]
        mock_httpx_client.get.assert_called_once()

    def test_search_with_max_results(self, mock_httpx_client):
        """Test search with custom max_results."""
        mock_response = Mock()
        mock_response.text = EMPTY_ARXIV_XML
        mock_response.raise_for_status = Mock()
        mock_httpx_client.get.return_value = mock_response
        
        search_arxiv("test", max_results=50)
        
        call_args = mock_httpx_client.get.call_args
        assert call_args.kwargs["params"]["max_results"] == 50

    def test_search_empty_query_raises(self):
        """Test that empty query raises ValueError."""
        with pytest.raises(ValueError) as exc_info:
            search_arxiv("")
        
        assert "query cannot be empty" in str(exc_info.value)

    def test_search_whitespace_query_raises(self):
        """Test that whitespace-only query raises ValueError."""
        with pytest.raises(ValueError) as exc_info:
            search_arxiv("   ")
        
        assert "query cannot be empty" in str(exc_info.value)

    def test_search_max_results_too_low_raises(self):
        """Test that max_results < 1 raises ValueError."""
        with pytest.raises(ValueError) as exc_info:
            search_arxiv("test", max_results=0)
        
        assert "max_results must be between 1 and 100" in str(exc_info.value)

    def test_search_max_results_too_high_raises(self):
        """Test that max_results > 100 raises ValueError."""
        with pytest.raises(ValueError) as exc_info:
            search_arxiv("test", max_results=101)
        
        assert "max_results must be between 1 and 100" in str(exc_info.value)

    def test_search_invalid_sort_by_raises(self):
        """Test that invalid sort_by raises ValueError."""
        with pytest.raises(ValueError) as exc_info:
            search_arxiv("test", sort_by="invalid")
        
        assert "sort_by must be" in str(exc_info.value)

    def test_search_invalid_sort_order_raises(self):
        """Test that invalid sort_order raises ValueError."""
        with pytest.raises(ValueError) as exc_info:
            search_arxiv("test", sort_order="invalid")
        
        assert "sort_order must be" in str(exc_info.value)

    def test_search_timeout_error(self, mock_httpx_client):
        """Test that timeout raises ArxivAPIError."""
        mock_httpx_client.get.side_effect = httpx.TimeoutException("Timeout")
        
        with pytest.raises(ArxivAPIError) as exc_info:
            search_arxiv("test")
        
        assert "Request timed out" in str(exc_info.value)

    def test_search_http_error(self, mock_httpx_client):
        """Test that HTTP error raises ArxivAPIError."""
        mock_response = Mock()
        mock_response.status_code = 500
        mock_httpx_client.get.side_effect = httpx.HTTPStatusError(
            "Server Error",
            request=Mock(),
            response=mock_response
        )
        
        with pytest.raises(ArxivAPIError) as exc_info:
            search_arxiv("test")
        
        assert "HTTP error" in str(exc_info.value)

    def test_search_network_error(self, mock_httpx_client):
        """Test that network error raises ArxivAPIError."""
        mock_httpx_client.get.side_effect = httpx.RequestError("Network error")
        
        with pytest.raises(ArxivAPIError) as exc_info:
            search_arxiv("test")
        
        assert "Network error" in str(exc_info.value)

    def test_search_strips_whitespace(self, mock_httpx_client):
        """Test that query whitespace is stripped."""
        mock_response = Mock()
        mock_response.text = EMPTY_ARXIV_XML
        mock_response.raise_for_status = Mock()
        mock_httpx_client.get.return_value = mock_response
        
        search_arxiv("  machine learning  ")
        
        call_args = mock_httpx_client.get.call_args
        assert call_args.kwargs["params"]["search_query"] == "all:machine learning"


class TestConstants:
    """Tests for module constants."""

    def test_arxiv_api_url(self):
        """Test ARXIV_API_URL constant."""
        assert ARXIV_API_URL == "https://export.arxiv.org/api/query"

    def test_api_timeout(self):
        """Test API_TIMEOUT constant."""
        assert API_TIMEOUT == 60.0


class TestArxivAPIError:
    """Tests for ArxivAPIError exception."""

    def test_exception_message(self):
        """Test exception can be raised with message."""
        with pytest.raises(ArxivAPIError) as exc_info:
            raise ArxivAPIError("Test error message")
        
        assert str(exc_info.value) == "Test error message"

    def test_exception_inheritance(self):
        """Test exception inherits from Exception."""
        assert issubclass(ArxivAPIError, Exception)
