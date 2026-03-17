from __future__ import annotations

import asyncio
import json
import os
import re
import uuid
from typing import Any

from copilot import CopilotClient, PermissionHandler
from copilot.tools import define_tool
from durabletask import task
from pydantic import BaseModel, Field

HIGH_VALUE_THRESHOLD = 10_000.0
DOMESTIC_COUNTRIES = {'us', 'usa', 'united states', 'united states of america'}
CURRENCY_SYMBOLS = {
    '$': 'USD',
    '€': 'EUR',
    '£': 'GBP',
}
SIMULATED_SANCTIONED_COUNTRIES = {
    'belarus',
    'iran',
    'north korea',
    'russia',
    'syria',
}
IBAN_COUNTRY_NAMES = {
    'BE': 'Belgium',
    'CH': 'Switzerland',
    'DE': 'Germany',
    'ES': 'Spain',
    'FR': 'France',
    'GB': 'United Kingdom',
    'IE': 'Ireland',
    'IT': 'Italy',
    'NL': 'Netherlands',
    'PL': 'Poland',
}
RISK_ANALYST_AGENT = {
    'name': 'risk-analyst',
    'display_name': 'Risk Analyst',
    'description': 'Reviews transfer requests with simulated compliance and payment validation tools.',
    'prompt': (
        'You are a bank risk analyst reviewing outbound transfer requests. Decide which tools to call based on the '
        'request details. Use check_sanctions_list when destination-country screening would add evidence, and use '
        'validate_iban when the request includes or implies an IBAN. Base the final answer on the request and any '
        'tool results. Return only JSON with keys sender, recipient, amount, currency, country, description, '
        'risk_level, analysis, proposed_action. Use risk_level values HIGH_RISK, MEDIUM_RISK, or LOW_RISK. Use '
        'proposed_action values WAIT_FOR_APPROVAL or EXECUTE_TRANSFER.'
    ),
}


class SanctionsCheckParams(BaseModel):
    country: str = Field(description='Destination country to screen against a simulated sanctions list')


@define_tool(description='Screen a destination country against a simulated sanctions list and return clear or flagged')
def check_sanctions_list(params: SanctionsCheckParams) -> str:
    normalized_country = re.sub(r'\s+', ' ', params.country).strip()
    if not normalized_country:
        return 'FLAGGED: no destination country was provided for sanctions screening.'

    if normalized_country.lower() in SIMULATED_SANCTIONED_COUNTRIES:
        return (
            f'FLAGGED: {normalized_country} matched the simulated sanctions list. '
            'Escalate for manual compliance review.'
        )

    return f'CLEAR: {normalized_country} did not match the simulated sanctions list.'


class IbanValidationParams(BaseModel):
    iban: str = Field(description='IBAN to validate for basic structure and country extraction')


@define_tool(description='Validate an IBAN format and extract its country from the IBAN prefix')
def validate_iban(params: IbanValidationParams) -> str:
    normalized_iban = re.sub(r'\s+', '', params.iban).upper()
    if not normalized_iban:
        return 'INVALID: no IBAN was provided.'

    if not re.fullmatch(r'[A-Z]{2}\d{2}[A-Z0-9]{11,30}', normalized_iban):
        return (
            'INVALID: IBAN must start with a two-letter country code, two check digits, '
            'and contain 15-34 alphanumeric characters.'
        )

    country_code = normalized_iban[:2]
    country_name = IBAN_COUNTRY_NAMES.get(country_code, f'country code {country_code}')
    return f'VALID: IBAN format looks correct. Country={country_name} ({country_code}). Normalized={normalized_iban}'


async def _run_copilot_session(
    prompt: str,
    system_prompt: str,
    *,
    tools: list[Any] | None = None,
    custom_agents: list[dict[str, Any]] | None = None,
    agent: str | None = None,
) -> str:
    client = CopilotClient()
    await client.start()

    try:
        session_config: dict[str, Any] = {
            'model': os.getenv('COPILOT_MODEL', 'gpt-5.4'),
            'on_permission_request': PermissionHandler.approve_all,
            'system_message': {'content': system_prompt},
        }
        if tools:
            session_config['tools'] = tools
        if custom_agents:
            session_config['custom_agents'] = custom_agents
        if agent:
            session_config['agent'] = agent

        session = await client.create_session(session_config)
        try:
            response = await session.send_and_wait({'prompt': prompt})
            if response and hasattr(response, 'data') and hasattr(response.data, 'content'):
                return response.data.content or ''
            return ''
        finally:
            await session.disconnect()
    finally:
        await client.stop()


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


def _normalize_risk_level(value: str | None) -> str:
    normalized = (value or '').strip().lower()
    if normalized in {'high', 'high_risk', 'high-risk'}:
        return 'HIGH_RISK'
    if normalized in {'medium', 'medium_risk', 'medium-risk'}:
        return 'MEDIUM_RISK'
    return 'LOW_RISK'


def _extract_json_payload(raw_response: str) -> dict[str, Any]:
    cleaned = raw_response.strip()
    if cleaned.startswith('```'):
        cleaned = re.sub(r'^```(?:json)?\s*', '', cleaned)
        cleaned = re.sub(r'\s*```$', '', cleaned)
    return json.loads(cleaned)


def _heuristic_analysis(user_request: str) -> dict[str, Any]:
    request_text = user_request.strip()
    amount, currency = _extract_amount_and_currency(request_text)
    country = _extract_country(request_text)
    recipient = _extract_recipient(request_text)
    description = _extract_description(request_text)
    sender = _extract_sender(request_text)

    risk_factors: list[str] = []
    if amount > HIGH_VALUE_THRESHOLD:
        risk_factors.append('The amount exceeds the $10,000 approval threshold.')
    if country.strip().lower() not in DOMESTIC_COUNTRIES:
        risk_factors.append(f'The destination is international ({country}).')
    if any(token in request_text.lower() for token in {'iban', 'swift', 'urgent', 'offshore', 'after hours'}):
        risk_factors.append('The request contains elevated-risk language or routing details.')

    risk_level = 'HIGH_RISK' if risk_factors else 'LOW_RISK'
    proposed_action = 'WAIT_FOR_APPROVAL' if risk_level == 'HIGH_RISK' else 'EXECUTE_TRANSFER'

    return {
        'request_id': str(uuid.uuid4()),
        'sender': sender,
        'recipient': recipient,
        'amount': amount,
        'currency': currency,
        'country': country,
        'description': description,
        'risk_level': risk_level,
        'analysis': ' '.join(risk_factors) or 'Low-risk domestic transfer with no unusual indicators.',
        'proposed_action': proposed_action,
    }


def analyze_request(ctx: task.ActivityContext, request: str) -> dict[str, Any]:
    del ctx
    heuristic = _heuristic_analysis(request)
    prompt = (
        'Review this transfer request and decide which available checks to run before finalizing the assessment. '
        'Return strict JSON with keys sender, recipient, amount, currency, country, description, risk_level, '
        'analysis, proposed_action. '
        f'Transfer request: {request.strip()}'
    )

    try:
        raw_response = asyncio.run(
            _run_copilot_session(
                prompt,
                RISK_ANALYST_AGENT['prompt'],
                tools=[check_sanctions_list, validate_iban],
                custom_agents=[RISK_ANALYST_AGENT],
                agent=RISK_ANALYST_AGENT['name'],
            )
        )
        payload = _extract_json_payload(raw_response)
        heuristic.update(
            {
                'sender': str(payload.get('sender') or heuristic['sender']),
                'recipient': str(payload.get('recipient') or heuristic['recipient']),
                'amount': float(payload.get('amount') or heuristic['amount']),
                'currency': str(payload.get('currency') or heuristic['currency']).upper(),
                'country': str(payload.get('country') or heuristic['country']),
                'description': str(payload.get('description') or heuristic['description']),
                'risk_level': _normalize_risk_level(payload.get('risk_level')),
                'analysis': str(payload.get('analysis') or heuristic['analysis']),
                'proposed_action': str(payload.get('proposed_action') or heuristic['proposed_action']).upper(),
            }
        )
    except Exception as exc:
        print(f'Copilot analysis failed ({exc}); falling back to heuristic review.')

    if heuristic['risk_level'] == 'HIGH_RISK':
        heuristic['proposed_action'] = 'WAIT_FOR_APPROVAL'
    elif heuristic['proposed_action'] not in {'EXECUTE_TRANSFER', 'WAIT_FOR_APPROVAL'}:
        heuristic['proposed_action'] = 'EXECUTE_TRANSFER'

    return heuristic


def notify_approvers(ctx: task.ActivityContext, analysis: dict[str, Any]) -> str:
    message = (
        'Approval required for transfer '
        f"{analysis['request_id']}: {analysis['amount']} {analysis['currency']} to {analysis['recipient']} "
        f"in {analysis['country']}. Risk={analysis['risk_level']}. Reason: {analysis['analysis']}"
    )
    print(message)
    print('Approve with:')
    print(
        '  python3 send_approval.py '
        f"<instance_id> approve \"Approved transfer {analysis['request_id']}\""
    )
    return message


def execute_transfer(ctx: task.ActivityContext, analysis: dict[str, Any]) -> str:
    del ctx
    approval = analysis.get('approval') or {'approved': True, 'approver': 'system', 'reason': 'Auto-approved'}
    prompt = (
        'Execute this approved transfer. Use any available tool capabilities if configured, and if no live banking tool '
        'is available, explicitly state that the execution is simulated for the demo. '
        'Return a concise operational summary including a transaction identifier, the transfer details, and the '
        'approval source. '
        f'Approved transfer payload: {json.dumps({**analysis, "approval": approval}, sort_keys=True)}'
    )
    system_prompt = (
        'You are a payment operations copilot. Execute only already-approved transfers and summarize the result '
        'for an audit log.'
    )

    try:
        response = asyncio.run(_run_copilot_session(prompt, system_prompt)).strip()
        if response:
            return response
    except Exception as exc:
        print(f'Copilot transfer execution failed ({exc}); returning simulated execution result.')

    transaction_id = f"tx-{uuid.uuid4().hex[:12]}"
    return (
        f"Simulated transfer executed. transaction_id={transaction_id}; recipient={analysis['recipient']}; "
        f"amount={analysis['amount']} {analysis['currency']}; approved_by={approval.get('approver', 'system')}"
    )
