import json
import os
import re
import uuid
from pathlib import Path
from typing import Any

from dotenv import load_dotenv
from durabletask import task
from openai import OpenAI

from models import TransferRequest

load_dotenv(Path(__file__).resolve().parents[3] / ".env")

HIGH_VALUE_THRESHOLD = 10_000.0
DOMESTIC_COUNTRIES = {'us', 'usa', 'united states', 'united states of america'}
NEW_RECIPIENT_HINTS = {'new recipient', 'new vendor', 'first-time', 'iban', 'swift'}
UNUSUAL_PATTERN_HINTS = {'urgent', 'immediately', 'after hours', 'expedite', 'offshore', 'crypto'}
CURRENCY_SYMBOLS = {
    '$': 'USD',
    '€': 'EUR',
    '£': 'GBP',
}


def _extract_amount_and_currency(user_request: str) -> tuple[float, str]:
    symbol_match = re.search(r'([$€£])\s*([\d,]+(?:\.\d+)?)', user_request)
    if symbol_match:
        currency = CURRENCY_SYMBOLS[symbol_match.group(1)]
        amount = float(symbol_match.group(2).replace(',', ''))
        return amount, currency

    code_match = re.search(r'\b(USD|EUR|GBP|JPY|CAD|AUD)\s*([\d,]+(?:\.\d+)?)', user_request, re.IGNORECASE)
    if code_match:
        currency = code_match.group(1).upper()
        amount = float(code_match.group(2).replace(',', ''))
        return amount, currency

    generic_match = re.search(r'([\d,]+(?:\.\d+)?)', user_request)
    if generic_match:
        amount = float(generic_match.group(1).replace(',', ''))
        return amount, 'USD'

    return 0.0, 'USD'


def _extract_country(user_request: str) -> str:
    match = re.search(r'\bin\s+([A-Z][A-Za-z .-]+?)(?:\s+for\b|$)', user_request)
    if match:
        return match.group(1).strip(' .')
    return 'United States'


def _extract_recipient(user_request: str) -> str:
    match = re.search(r'\bto\b\s+(.+?)(?:\s+in\s+[A-Z][A-Za-z .-]+?(?:\s+for\b|$)|\s+for\b|$)', user_request)
    if match:
        return match.group(1).strip(' .')
    return 'Unspecified recipient'


def _extract_description(user_request: str) -> str:
    match = re.search(r'\bfor\b\s+(.+)$', user_request)
    if match:
        return match.group(1).strip(' .')
    return 'General business transfer'


def _extract_sender(user_request: str) -> str:
    match = re.search(r'\bfrom\b\s+(.+?)\s+to\b', user_request)
    if match:
        return match.group(1).strip(' .')
    return 'Primary operating account'


def _is_international(country: str) -> bool:
    normalized_country = country.strip().lower()
    return normalized_country not in DOMESTIC_COUNTRIES


def _heuristic_analysis(user_request: str) -> TransferRequest:
    request_text = user_request.strip()
    amount, currency = _extract_amount_and_currency(request_text)
    country = _extract_country(request_text)
    recipient = _extract_recipient(request_text)
    description = _extract_description(request_text)
    sender = _extract_sender(request_text)
    normalized = request_text.lower()

    risk_factors: list[str] = []
    if amount > HIGH_VALUE_THRESHOLD:
        risk_factors.append('The transfer amount exceeds the $10,000 approval threshold.')
    if _is_international(country):
        risk_factors.append(f'The transfer destination is international ({country}).')
    if any(hint in normalized for hint in NEW_RECIPIENT_HINTS):
        risk_factors.append('The request may involve a new or difficult-to-verify recipient.')
    if any(hint in normalized for hint in UNUSUAL_PATTERN_HINTS):
        risk_factors.append('The request contains urgency or unusual-pattern language.')

    if amount > HIGH_VALUE_THRESHOLD or _is_international(country):
        risk_level = 'high'
    elif risk_factors:
        risk_level = 'medium'
    else:
        risk_level = 'low'

    analysis = ' '.join(risk_factors) if risk_factors else 'Domestic transfer below the approval threshold with no unusual indicators.'

    return TransferRequest(
        request_id=str(uuid.uuid4()),
        sender=sender,
        recipient=recipient,
        amount=amount,
        currency=currency,
        country=country,
        description=description,
        risk_level=risk_level,
        analysis=analysis,
    )


def _coerce_amount(value: Any, fallback: float) -> float:
    if value is None:
        return fallback
    if isinstance(value, (int, float)):
        return float(value)
    try:
        return float(str(value).replace(',', '').strip())
    except ValueError:
        return fallback


def analyze_transfer_request_activity(ctx: task.ActivityContext, user_request: str) -> dict[str, Any]:
    request_text = user_request.strip()
    api_key = os.getenv('OPENAI_API_KEY')
    heuristic_transfer = _heuristic_analysis(request_text)

    if not api_key:
        print('OPENAI_API_KEY not found; using heuristic transfer risk analysis for demo purposes.')
        return heuristic_transfer.model_dump()

    client = OpenAI(
        base_url=os.getenv("OPENAI_BASE_URL"),
        api_key=api_key,
        max_retries=0,
    )
    model = os.getenv('OPENAI_MODEL', 'gpt-5.4')

    prompt = (
        'Analyze this financial transfer request for risk. '\
        'Extract the transfer details and return strict JSON with keys: '\
        'sender, recipient, amount, currency, country, description, risk_level, analysis. '\
        'Consider these risk factors: high amount (over $10,000), international transfer, new recipient, '\
        f'unusual payment pattern. Use an ISO currency code when possible and make amount numeric. '\
        f'User request: {request_text}'
    )

    try:
        response = client.chat.completions.create(
            model=model,
            response_format={'type': 'json_object'},
            messages=[
                {
                    'role': 'system',
                    'content': (
                        'You analyze financial transfer requests for a durable approval workflow. '
                        'Extract concrete transfer fields and keep the analysis concise.'
                    ),
                },
                {'role': 'user', 'content': prompt},
            ],
        )
        payload = json.loads(response.choices[0].message.content or '{}')
        transfer_request = TransferRequest(
            request_id=str(uuid.uuid4()),
            sender=str(payload.get('sender') or heuristic_transfer.sender),
            recipient=str(payload.get('recipient') or heuristic_transfer.recipient),
            amount=_coerce_amount(payload.get('amount'), heuristic_transfer.amount),
            currency=str(payload.get('currency') or heuristic_transfer.currency).upper(),
            country=str(payload.get('country') or heuristic_transfer.country),
            description=str(payload.get('description') or heuristic_transfer.description),
            risk_level=str(payload.get('risk_level') or heuristic_transfer.risk_level).lower(),
            analysis=str(payload.get('analysis') or heuristic_transfer.analysis),
        )
    except Exception as exc:
        print(f'OpenAI analysis failed ({exc}); falling back to heuristic analysis.')
        transfer_request = heuristic_transfer

    return transfer_request.model_dump()
