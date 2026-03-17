from __future__ import annotations

import asyncio
import sys

from mcp_consumer import DEFAULT_PROMPT, ask_repository_question


def main() -> None:
    prompt = ' '.join(sys.argv[1:]).strip() or DEFAULT_PROMPT
    if len(sys.argv) < 2:
        print('No prompt supplied; using default repository question:')
        print(f'  {DEFAULT_PROMPT}')
    print(asyncio.run(ask_repository_question(prompt)))


if __name__ == '__main__':
    main()
