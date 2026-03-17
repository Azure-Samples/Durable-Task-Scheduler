"""Shared configuration for Durable Task AI Hub recipes."""

import os
from pathlib import Path

from dotenv import load_dotenv

# Load .env from the ai-recipes root directory
_env_path = Path(__file__).resolve().parent.parent / ".env"
load_dotenv(_env_path)

# Durable Task Scheduler emulator defaults
DEFAULT_ENDPOINT = "http://localhost:8080"
DEFAULT_TASKHUB = "default"


def get_endpoint() -> str:
    return os.environ.get("DTS_ENDPOINT", DEFAULT_ENDPOINT)


def get_taskhub() -> str:
    return os.environ.get("DTS_TASKHUB", DEFAULT_TASKHUB)


def get_openai_api_key() -> str:
    key = os.environ.get("OPENAI_API_KEY")
    if not key:
        raise EnvironmentError("OPENAI_API_KEY environment variable is required")
    return key


def get_openai_base_url() -> str | None:
    """Return the Azure OpenAI base URL, or None to use the default OpenAI endpoint."""
    return os.environ.get("OPENAI_BASE_URL")


def get_openai_model() -> str:
    return os.environ.get("OPENAI_MODEL", "gpt-5.4")


def get_embedding_model() -> str:
    return os.environ.get("OPENAI_EMBEDDING_MODEL", "text-embedding-3-small")
