"""Authentication utilities for the arXiv Research Agent."""

import logging
import os
from typing import Optional, Union

from azure.identity import DefaultAzureCredential, ManagedIdentityCredential

logger = logging.getLogger(__name__)


def get_credential() -> Optional[Union[DefaultAzureCredential, ManagedIdentityCredential]]:
    """Get Azure credential for authentication.

    Returns:
        Credential object for Azure authentication, or None for local emulator.
    """
    endpoint = os.getenv("ENDPOINT", "http://localhost:8080")

    if endpoint == "http://localhost:8080":
        return None

    try:
        client_id = os.getenv("AZURE_MANAGED_IDENTITY_CLIENT_ID")
        if client_id:
            logger.info(f"Using Managed Identity with client ID: {client_id}")
            credential = ManagedIdentityCredential(client_id=client_id)
            credential.get_token("https://management.azure.com/.default")
            logger.info("Successfully authenticated with Managed Identity")
            return credential
        else:
            logger.info("Using DefaultAzureCredential")
            return DefaultAzureCredential()
    except Exception as e:
        logger.error(f"Authentication error: {e}")
        logger.warning("Continuing without authentication - this may only work with local emulator")
        return None
