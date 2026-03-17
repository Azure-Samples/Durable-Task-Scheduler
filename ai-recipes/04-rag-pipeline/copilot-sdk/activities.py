from __future__ import annotations

import asyncio
import os

from copilot import CopilotClient, PermissionHandler
from copilot.tools import define_tool
from durabletask import task
from pydantic import BaseModel, Field


class SearchQueryParams(BaseModel):
    query: str = Field(description='The user question or retrieval query to search for')


RAG_ANALYST_AGENT = {
    'name': 'rag-analyst',
    'display_name': 'RAG Analyst',
    'description': 'Grounds answers by consulting retrieval tools before responding.',
    'prompt': (
        'You are a RAG analyst for a durable retrieval-and-generation workflow. '
        'Before answering, gather evidence by calling the available retrieval tools that are relevant to the question. '
        'Use the precomputed durable retrieval context in the prompt as your starting evidence, then consult tools to '
        'corroborate, enrich, or clarify gaps. Cite which sources informed your answer and explicitly note when the '
        'evidence is simulated or incomplete. Format the final response with Answer, Evidence, and Gaps.'
    ),
}


def _vector_results(query: str) -> str:
    matches = [
        f"semantic chunk: Durable execution persists progress for '{query}'.",
        f"semantic chunk: Fan-out/fan-in helps parallelize retrieval for '{query}'.",
    ]
    return 'Vector DB results:\n- ' + '\n- '.join(matches)


@define_tool(name='search_vector_db', description='Search the simulated vector database for semantically relevant snippets.')
async def search_vector_db_tool(params: SearchQueryParams) -> str:
    return _vector_results(params.query)



def _document_results(query: str) -> str:
    matches = [
        f"document excerpt: The knowledge base contains operational notes about '{query}'.",
        'document excerpt: Source documents can be chunked, embedded, and refreshed on a schedule.',
    ]
    return 'Document store results:\n- ' + '\n- '.join(matches)


@define_tool(name='search_document_store', description='Search the simulated document store for supporting excerpts.')
async def search_document_store_tool(params: SearchQueryParams) -> str:
    return _document_results(params.query)



def _graph_results(query: str) -> str:
    matches = [
        f"entity relation: '{query}' -> durable workflows -> retries -> resilience.",
        'entity relation: retrieval pipelines connect documents, entities, and generated answers.',
    ]
    return 'Knowledge graph results:\n- ' + '\n- '.join(matches)


@define_tool(name='search_knowledge_graph', description='Search the simulated knowledge graph for entities and relationships.')
async def search_knowledge_graph_tool(params: SearchQueryParams) -> str:
    return _graph_results(params.query)


async def _run_copilot_session(query: str, context: str) -> str:
    prompt = (
        'Generate the final answer for this durable RAG pipeline demo. The workflow already ran Durable Task fan-out '
        'retrieval, and that context is included below. Use that durable retrieval context as baseline evidence, then '
        'decide which retrieval tools to call for additional support before answering.\n\n'
        f'User query: {query}\n\n'
        f'Durable retrieval context:\n{context}'
    )

    client = CopilotClient()
    await client.start()

    try:
        session = await client.create_session(
            {
                'model': os.getenv('COPILOT_MODEL', 'gpt-5.4'),
                'on_permission_request': PermissionHandler.approve_all,
                'tools': [search_vector_db_tool, search_knowledge_graph_tool, search_document_store_tool],
                'custom_agents': [RAG_ANALYST_AGENT],
                'agent': RAG_ANALYST_AGENT['name'],
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



def retrieve_context(ctx: task.ActivityContext, payload: dict[str, str]) -> str:
    source = payload['source']
    query = payload['query']
    handlers = {
        'vector_db': _vector_results,
        'document_store': _document_results,
        'knowledge_graph': _graph_results,
    }
    if source not in handlers:
        raise ValueError(f'Unknown retrieval source: {source}')
    return handlers[source](query)



def search_vector_db(ctx: task.ActivityContext, query: str) -> str:
    return retrieve_context(ctx, {'source': 'vector_db', 'query': query})



def search_document_store(ctx: task.ActivityContext, query: str) -> str:
    return retrieve_context(ctx, {'source': 'document_store', 'query': query})



def search_knowledge_graph(ctx: task.ActivityContext, query: str) -> str:
    return retrieve_context(ctx, {'source': 'knowledge_graph', 'query': query})



def generate_answer_copilot(ctx: task.ActivityContext, context_and_query: dict[str, str]) -> str:
    del ctx
    query = context_and_query['query']
    context = context_and_query['context']

    try:
        response = asyncio.run(_run_copilot_session(query, context)).strip()
        if response:
            return response
    except Exception as exc:
        print(f'Copilot answer generation failed ({exc}); returning a fallback grounded summary.')

    return (
        'Answer:\n'
        f'- Based on the retrieved context, {query} is supported by the available vector, document, and graph summaries.\n\n'
        'Evidence:\n'
        f'{context}\n\n'
        'Gaps:\n'
        '- Copilot SDK generation was unavailable, so this is a fallback summary over the retrieved context.'
    )
