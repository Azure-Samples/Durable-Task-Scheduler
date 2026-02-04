"""LLM utilities for the arXiv Research Agent.

This module provides a simple interface to Azure OpenAI's Responses API
for making LLM calls with JSON output support.
"""

import json
import logging
import os
from typing import Any, Dict, List, Optional

from dotenv import load_dotenv
from openai import OpenAI

load_dotenv()

logger = logging.getLogger(__name__)

# Configuration constants
DEFAULT_MODEL: str = os.environ.get("AZURE_OPENAI_DEPLOYMENT", "gpt-5.2")
DEFAULT_TEMPERATURE: float = 0.1
DEFAULT_MAX_TOKENS: int = 6000

# Lazy-initialized client
_client: Optional[OpenAI] = None


def _get_client() -> OpenAI:
    """Get or create the OpenAI client (lazy initialization).

    Returns:
        Initialized OpenAI client

    Raises:
        RuntimeError: If required environment variables are not set
    """
    global _client

    if _client is not None:
        return _client

    endpoint = os.environ.get("AZURE_OPENAI_ENDPOINT")
    if not endpoint:
        raise RuntimeError(
            "AZURE_OPENAI_ENDPOINT environment variable is required. "
            "Set it to your Azure OpenAI endpoint URL."
        )

    base_url = f"{endpoint.rstrip('/')}/openai/v1/"
    api_key = os.environ.get("AZURE_OPENAI_API_KEY")

    if api_key:
        # Use API key authentication
        logger.info("Initializing OpenAI client with API key authentication")
        _client = OpenAI(base_url=base_url, api_key=api_key)
    else:
        # Use Entra ID (Azure AD) authentication
        logger.info("Initializing OpenAI client with Entra ID authentication")
        from azure.identity import DefaultAzureCredential, get_bearer_token_provider

        token_provider = get_bearer_token_provider(
            DefaultAzureCredential(),
            "https://cognitiveservices.azure.com/.default",
        )
        _client = OpenAI(base_url=base_url, api_key=token_provider)

    return _client


def call_llm(
    messages: List[Dict[str, str]],
    model: str = DEFAULT_MODEL,
    temperature: float = DEFAULT_TEMPERATURE,
    max_tokens: int = DEFAULT_MAX_TOKENS,
    json_output: bool = True,
) -> str:
    """Make an LLM API call using Azure OpenAI's Responses API.

    Args:
        messages: List of message dictionaries with 'role' and 'content' keys
        model: The deployment/model name to use
        temperature: Sampling temperature (0.0-1.0)
        max_tokens: Maximum tokens in response
        json_output: If True, request structured JSON output

    Returns:
        The LLM response content as a string

    Raises:
        RuntimeError: If client initialization fails
        ValueError: If the LLM returns an empty response
        Exception: If the API call fails
    """
    client = _get_client()

    # Convert messages to input format for Responses API
    input_text = "\n".join(f"{msg['role'].upper()}: {msg['content']}" for msg in messages)

    try:
        if json_output:
            response = client.responses.create(  # type: ignore[attr-defined]
                model=model,
                input=input_text,
                temperature=temperature,
                max_output_tokens=max_tokens,
                text={"format": {"type": "json_object"}},
            )
        else:
            response = client.responses.create(  # type: ignore[attr-defined]
                model=model,
                input=input_text,
                temperature=temperature,
                max_output_tokens=max_tokens,
            )

        content: Optional[str] = response.output_text  # type: ignore[attr-defined]
        if not content:
            raise ValueError("LLM returned empty response")
        return content

    except Exception as e:
        logger.error(f"LLM API call failed: {e}")
        raise


def parse_json_response(response: str) -> Any:
    """Parse JSON from LLM response.

    Args:
        response: Raw LLM response string

    Returns:
        Parsed JSON (could be dict, list, or primitive)
    """
    return json.loads(response.strip())
