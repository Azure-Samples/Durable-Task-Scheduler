"""
Currency converter tool that uses the free ExchangeRate-API to convert between currencies.
API Documentation: https://www.exchangerate-api.com/docs/free
No API key required for basic usage.
"""
import httpx
import logging
from datetime import datetime, timedelta
from typing import Annotated, Dict, Tuple
from pydantic import BaseModel

logger = logging.getLogger(__name__)

# In-memory cache for exchange rates: {from_currency: (rates_dict, timestamp)}
_exchange_rate_cache: Dict[str, Tuple[dict, datetime]] = {}
_CACHE_TTL = timedelta(minutes=5)  # Cache rates for 5 minutes


class CurrencyConversion(BaseModel):
    """Result of a currency conversion operation."""
    from_currency: str
    to_currency: str
    original_amount: float
    converted_amount: float
    exchange_rate: float
    timestamp: str  # ISO format string for JSON serialization


async def _get_rates_for_currency(from_currency: str) -> dict:
    """
    Gets all exchange rates for a base currency, with caching.
    Returns the full rates dictionary from the API.
    """
    from_currency = from_currency.upper()
    
    # Check cache first
    if from_currency in _exchange_rate_cache:
        rates, cached_at = _exchange_rate_cache[from_currency]
        if datetime.utcnow() - cached_at < _CACHE_TTL:
            logger.info(f"Using cached exchange rates for {from_currency}")
            return rates
    
    # Fetch fresh rates
    logger.info(f"Fetching fresh exchange rates for {from_currency}")
    async with httpx.AsyncClient(timeout=30.0) as client:
        response = await client.get(
            f"https://open.er-api.com/v6/latest/{from_currency}"
        )
        response.raise_for_status()
        
        data = response.json()
        
        # Check if the response has an error
        if "error-type" in data:
            raise ValueError(f"Invalid currency code: {from_currency}")
        
        rates = data.get("rates", {})
        if not rates:
            raise ValueError(f"No rates found for {from_currency}")
        
        # Cache the rates
        _exchange_rate_cache[from_currency] = (data, datetime.utcnow())
        
        return data


async def convert_currency(
    amount: Annotated[float, "The amount to convert"],
    from_currency: Annotated[str, "Source currency code (e.g., USD, EUR, GBP, JPY)"],
    to_currency: Annotated[str, "Target currency code (e.g., USD, EUR, GBP, JPY)"]
) -> CurrencyConversion:
    """
    Converts an amount from one currency to another using current exchange rates.
    Useful for helping travelers understand costs in their home currency.
    """
    try:
        data = await _get_rates_for_currency(from_currency)
        
        rates = data.get("rates", {})
        if to_currency.upper() not in rates:
            raise ValueError(f"Unable to find exchange rate for {to_currency}")
        
        exchange_rate = rates[to_currency.upper()]
        converted_amount = round(amount * exchange_rate, 2)
        
        # Get timestamp as ISO string for JSON serialization
        timestamp = datetime.utcnow().isoformat()
        if "time_last_update_unix" in data:
            timestamp = datetime.utcfromtimestamp(data["time_last_update_unix"]).isoformat()
        
        return CurrencyConversion(
            from_currency=from_currency.upper(),
            to_currency=to_currency.upper(),
            original_amount=amount,
            converted_amount=converted_amount,
            exchange_rate=exchange_rate,
            timestamp=timestamp
        )
        
    except httpx.HTTPError as ex:
        raise RuntimeError(f"Failed to fetch exchange rates: {ex}")
    except Exception as ex:
        raise RuntimeError(f"Currency conversion failed: {ex}")


async def get_exchange_rate(
    from_currency: Annotated[str, "Source currency code (e.g., USD, EUR, GBP, JPY)"],
    to_currency: Annotated[str, "Target currency code (e.g., USD, EUR, GBP, JPY)"]
) -> float:
    """
    Gets the current exchange rate between two currencies.
    Use this to check conversion rates before calculating costs.
    """
    conversion = await convert_currency(1.0, from_currency, to_currency)
    return conversion.exchange_rate
