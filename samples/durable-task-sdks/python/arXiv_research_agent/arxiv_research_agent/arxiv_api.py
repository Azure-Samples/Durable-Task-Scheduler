"""arXiv API utilities for searching papers and retrieving metadata."""

import logging
import re
import time
import xml.etree.ElementTree as ET
from typing import Any, Dict, List, Optional

import httpx

logger = logging.getLogger(__name__)

# arXiv API configuration
ARXIV_API_URL = "https://export.arxiv.org/api/query"
API_TIMEOUT = 60.0  # arXiv API can be slow, use longer timeout

# Rate limiting: arXiv recommends no more than 1 request per 3 seconds
RATE_LIMIT_DELAY = 3.0  # seconds between requests
MAX_RETRIES = 3
RETRY_BACKOFF_BASE = 5.0  # base seconds for exponential backoff on 429

# Track last request time for rate limiting
_last_request_time: float = 0.0

# Shared httpx client with connection pooling for efficiency
_http_client: Optional[httpx.Client] = None


def _get_client() -> httpx.Client:
    """Get or create shared httpx client with connection pooling."""
    global _http_client
    if _http_client is None:
        _http_client = httpx.Client(
            timeout=API_TIMEOUT,
            limits=httpx.Limits(max_keepalive_connections=5, max_connections=10),
        )
    return _http_client


def _rate_limited_request(client: httpx.Client, url: str, params: Dict[str, Any]) -> httpx.Response:
    """Make a rate-limited request with retry logic for 429/503 errors."""
    global _last_request_time

    last_response: Optional[httpx.Response] = None

    for attempt in range(MAX_RETRIES):
        # Enforce rate limit
        elapsed = time.time() - _last_request_time
        if elapsed < RATE_LIMIT_DELAY:
            sleep_time = RATE_LIMIT_DELAY - elapsed
            logger.debug(f"Rate limiting: sleeping {sleep_time:.1f}s")
            time.sleep(sleep_time)

        _last_request_time = time.time()
        response: httpx.Response = client.get(url, params=params)
        last_response = response

        if response.status_code == 429:
            # Too Many Requests - exponential backoff
            backoff = RETRY_BACKOFF_BASE * (2 ** attempt)
            logger.warning(f"arXiv rate limit hit (429), retrying in {backoff}s (attempt {attempt + 1}/{MAX_RETRIES})")
            time.sleep(backoff)
            continue

        if response.status_code == 503:
            # Service Unavailable - exponential backoff
            backoff = RETRY_BACKOFF_BASE * (2 ** attempt)
            logger.warning(f"arXiv service unavailable (503), retrying in {backoff}s (attempt {attempt + 1}/{MAX_RETRIES})")
            time.sleep(backoff)
            continue

        return response

    # Return last response after all retries exhausted
    if last_response is not None:
        return last_response
    raise ArxivAPIError("Max retries exceeded")

# XML namespaces used by arXiv API
NAMESPACES = {
    "atom": "http://www.w3.org/2005/Atom",
    "arxiv": "http://arxiv.org/schemas/atom",
}


class ArxivAPIError(Exception):
    """Exception raised when arXiv API calls fail."""
    pass


def _clean_text(text: Optional[str]) -> str:
    """Clean text by removing extra whitespace and newlines."""
    if not text:
        return ""
    return re.sub(r'\s+', ' ', text.strip())


def _parse_entry(entry: ET.Element) -> Dict[str, Any]:
    """Parse an arXiv entry element into a dictionary.
    
    Args:
        entry: XML element representing an arXiv entry
        
    Returns:
        Dictionary with paper metadata
    """
    # Extract arxiv ID from the id URL
    id_elem = entry.find("atom:id", NAMESPACES)
    arxiv_id = ""
    if id_elem is not None and id_elem.text:
        # ID format: http://arxiv.org/abs/2301.12345v1
        arxiv_id = id_elem.text.split("/abs/")[-1] if "/abs/" in id_elem.text else id_elem.text
    
    # Title
    title_elem = entry.find("atom:title", NAMESPACES)
    title = _clean_text(title_elem.text if title_elem is not None else "")
    
    # Summary/Abstract
    summary_elem = entry.find("atom:summary", NAMESPACES)
    summary = _clean_text(summary_elem.text if summary_elem is not None else "")
    
    # Authors
    authors: List[str] = []
    for author in entry.findall("atom:author", NAMESPACES):
        name_elem = author.find("atom:name", NAMESPACES)
        if name_elem is not None and name_elem.text:
            authors.append(name_elem.text.strip())
    
    # Published date
    published_elem = entry.find("atom:published", NAMESPACES)
    published = published_elem.text if published_elem is not None and published_elem.text else ""

    # Updated date
    updated_elem = entry.find("atom:updated", NAMESPACES)
    updated = updated_elem.text if updated_elem is not None and updated_elem.text else ""
    
    # Categories
    categories: List[str] = []
    for category in entry.findall("atom:category", NAMESPACES):
        term = category.get("term")
        if term:
            categories.append(term)
    
    # Primary category
    primary_category_elem = entry.find("arxiv:primary_category", NAMESPACES)
    primary_category = ""
    if primary_category_elem is not None:
        primary_category = primary_category_elem.get("term", "")
    
    # Links
    pdf_url = ""
    abs_url = ""
    for link in entry.findall("atom:link", NAMESPACES):
        link_type = link.get("type", "")
        link_title = link.get("title", "")
        href = link.get("href", "")
        
        if link_title == "pdf" or link_type == "application/pdf":
            pdf_url = href
        elif link.get("rel") == "alternate":
            abs_url = href
    
    # Comment (often contains page count, conference info, etc.)
    comment_elem = entry.find("arxiv:comment", NAMESPACES)
    comment = _clean_text(comment_elem.text if comment_elem is not None else "")
    
    # Journal reference
    journal_elem = entry.find("arxiv:journal_ref", NAMESPACES)
    journal_ref = _clean_text(journal_elem.text if journal_elem is not None else "")
    
    # DOI
    doi_elem = entry.find("arxiv:doi", NAMESPACES)
    doi = doi_elem.text.strip() if doi_elem is not None and doi_elem.text else ""
    
    return {
        "arxiv_id": arxiv_id,
        "title": title,
        "summary": summary,
        "authors": authors,
        "published": published,
        "updated": updated,
        "categories": categories,
        "primary_category": primary_category,
        "pdf_url": pdf_url,
        "abs_url": abs_url or f"https://arxiv.org/abs/{arxiv_id}",
        "comment": comment,
        "journal_ref": journal_ref,
        "doi": doi,
    }


def search_arxiv(
    query: str,
    max_results: int = 30,
    sort_by: str = "relevance",
    sort_order: str = "descending"
) -> List[Dict[str, Any]]:
    """Search arXiv for papers matching the query.
    
    Args:
        query: Search query string (supports arXiv query syntax)
        max_results: Maximum number of results to return (1-100)
        sort_by: Sort field - "relevance", "lastUpdatedDate", or "submittedDate"
        sort_order: Sort order - "ascending" or "descending"
        
    Returns:
        List of paper dictionaries with metadata
        
    Raises:
        ArxivAPIError: If the API request fails
        ValueError: If parameters are invalid
    """
    if not 1 <= max_results <= 100:
        raise ValueError("max_results must be between 1 and 100")

    if not query or not query.strip():
        raise ValueError("query cannot be empty")
    
    if sort_by not in ["relevance", "lastUpdatedDate", "submittedDate"]:
        raise ValueError("sort_by must be 'relevance', 'lastUpdatedDate', or 'submittedDate'")

    if sort_order not in ["ascending", "descending"]:
        raise ValueError("sort_order must be 'ascending' or 'descending'")
    
    params = {
        "search_query": f"all:{query.strip()}",
        "start": 0,
        "max_results": max_results,
        "sortBy": sort_by,
        "sortOrder": sort_order,
    }
    
    try:
        client = _get_client()
        response = _rate_limited_request(client, ARXIV_API_URL, params)
        response.raise_for_status()
        
        # Parse XML response
        root = ET.fromstring(response.text)

        papers: List[Dict[str, Any]] = []
        for entry in root.findall("atom:entry", NAMESPACES):
            paper = _parse_entry(entry)
            papers.append(paper)
        
        return papers
            
    except httpx.TimeoutException as e:
        logger.error(f"Timeout searching arXiv for '{query}': {e}")
        raise ArxivAPIError(f"Request timed out: {e}") from e
    except httpx.HTTPStatusError as e:
        logger.error(f"HTTP error searching arXiv: {e}")
        raise ArxivAPIError(f"HTTP error: {e.response.status_code}") from e
    except httpx.RequestError as e:
        logger.error(f"Network error searching arXiv: {e}")
        raise ArxivAPIError(f"Network error: {e}") from e
    except ET.ParseError as e:
        logger.error(f"Failed to parse arXiv API response: {e}")
        raise ArxivAPIError(f"Invalid API response: {e}") from e


