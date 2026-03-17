from __future__ import annotations

import argparse
import os
import sys
from pathlib import Path

from copilot import CopilotClient, PermissionHandler

DEFAULT_PROMPT = 'Tell me about the microsoft/durabletask-python repository'


def _mcp_server_config() -> dict:
    server_path = Path(__file__).resolve().parent.parent / 'mcp_server.py'
    env = {
        key: value
        for key in ('DTS_ENDPOINT', 'DTS_TASKHUB', 'GITHUB_TOKEN')
        if (value := os.getenv(key))
    }
    return {
        'command': sys.executable,
        'args': [str(server_path)],
        'env': env,
    }


async def ask_repository_question(prompt: str) -> str:
    client = CopilotClient()
    await client.start()

    try:
        session = await client.create_session(
            {
                'model': os.getenv('COPILOT_MODEL', 'gpt-5.4'),
                'on_permission_request': PermissionHandler.approve_all,
                'mcp_servers': {
                    'github-inspector': _mcp_server_config(),
                },
            }
        )
        try:
            response = await session.send_and_wait({'prompt': prompt})
            if response and hasattr(response, 'data') and hasattr(response.data, 'content'):
                return response.data.content or ''
            return ''
        finally:
            await session.disconnect()
    finally:
        await client.stop()


async def main() -> None:
    parser = argparse.ArgumentParser(description='Ask a repository question through a Copilot SDK session with MCP tools')
    parser.add_argument('prompt', nargs='*', help='Question to ask the Copilot SDK session')
    args = parser.parse_args()
    prompt = ' '.join(args.prompt).strip() or DEFAULT_PROMPT
    print(await ask_repository_question(prompt))


if __name__ == '__main__':
    import asyncio

    asyncio.run(main())
