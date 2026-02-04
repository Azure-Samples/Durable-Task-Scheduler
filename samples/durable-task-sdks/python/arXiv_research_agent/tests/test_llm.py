"""Tests for LLM utilities."""

import pytest
import json
from unittest.mock import patch, Mock

from arxiv_research_agent.llm import (
    call_llm,
    parse_json_response,
    DEFAULT_MODEL,
    DEFAULT_TEMPERATURE,
    DEFAULT_MAX_TOKENS,
)


class TestParseJsonResponse:
    """Tests for parse_json_response function."""

    def test_parse_valid_json(self):
        """Test parsing valid JSON."""
        response = '{"key": "value", "number": 42}'
        result = parse_json_response(response)
        assert result == {"key": "value", "number": 42}

    def test_parse_json_array(self):
        """Test parsing JSON array."""
        response = '["item1", "item2", "item3"]'
        result = parse_json_response(response)
        assert result == ["item1", "item2", "item3"]

    def test_parse_invalid_json_raises(self):
        """Test that invalid JSON raises exception."""
        response = 'not valid json'
        with pytest.raises(json.JSONDecodeError):
            parse_json_response(response)


class TestCallLlm:
    """Tests for call_llm function."""

    def test_call_llm_success(self, mock_openai_client):
        """Test successful LLM call."""
        messages = [{"role": "user", "content": "Hello"}]
        
        result = call_llm(messages)
        
        assert result == '{"test": "response"}'
        mock_openai_client.responses.create.assert_called_once()

    def test_call_llm_with_custom_params(self, mock_openai_client):
        """Test LLM call with custom parameters."""
        messages = [{"role": "user", "content": "Hello"}]
        
        call_llm(
            messages,
            model="gpt-5.2",
            temperature=0.5,
            max_tokens=1000,
        )
        
        call_args = mock_openai_client.responses.create.call_args
        assert call_args.kwargs["model"] == "gpt-5.2"

    def test_call_llm_uses_defaults(self, mock_openai_client):
        """Test that LLM call uses default parameters."""
        messages = [{"role": "user", "content": "Hello"}]
        
        call_llm(messages)
        
        call_args = mock_openai_client.responses.create.call_args
        assert call_args.kwargs["model"] == DEFAULT_MODEL

    def test_call_llm_api_error(self, mock_openai_client):
        """Test LLM call handles API errors."""
        mock_openai_client.responses.create.side_effect = Exception("API Error")

        messages = [{"role": "user", "content": "Hello"}]

        with pytest.raises(Exception) as exc_info:
            call_llm(messages)

        assert "API Error" in str(exc_info.value)

    def test_call_llm_no_client(self, monkeypatch):
        """Test LLM call raises error when endpoint is not set."""
        # Reset the cached client and remove the endpoint env var
        import arxiv_research_agent.llm as llm_module
        llm_module._client = None
        monkeypatch.delenv("AZURE_OPENAI_ENDPOINT", raising=False)

        messages = [{"role": "user", "content": "Hello"}]

        with pytest.raises(RuntimeError) as exc_info:
            call_llm(messages)

        assert "AZURE_OPENAI_ENDPOINT" in str(exc_info.value)


class TestDefaultConstants:
    """Tests for default constants."""

    def test_default_model(self):
        """Test default model constant is set from deployment."""
        # DEFAULT_MODEL comes from AZURE_OPENAI_DEPLOYMENT env var (defaults to gpt-4o-mini)
        assert DEFAULT_MODEL is not None
        assert isinstance(DEFAULT_MODEL, str)

    def test_default_temperature(self):
        """Test default temperature constant."""
        assert DEFAULT_TEMPERATURE == 0.1

    def test_default_max_tokens(self):
        """Test default max tokens constant."""
        assert DEFAULT_MAX_TOKENS == 2000
