# arXiv research agent (Go)

A Go counterpart to the Python research agent: durable research iterations,
paper search and metadata fetching, model analysis, continuation decisions,
follow-up queries, synthesis, and a REST status/report API.

The default is an **explicit synthetic fixture**, so both emulator and live DTS
verification require **no arXiv access or model credentials**. `fixture-001`,
`fixture-002`, and `fixture-003` are intentionally not real arXiv IDs. Fixture
reports are not academic evidence.

## Architecture

```text
POST /agents -> GoArxivResearch
  iteration 1: query -> GoArxivPaperResearch
    search activity -> fan-out metadata fetch activities -> analysis activity
  continuation decision + follow-up query activities
  ContinueAsNew(checkpoint: topic, mode, iteration, queries, findings, papers)
  iteration 2+: fan-out up to two GoArxivPaperResearch children
  bounded continuation / early stop -> synthesis activity -> report
GET /agents/{id}, /wait -> persisted DTS metadata and output
DELETE /agents/{id} -> recursive termination
```

All network access is in activities. Orchestrators only manipulate typed,
deterministic checkpoint data and durable tasks; they never read environment
variables, call HTTP, use wall-clock time, or launch ordinary goroutines.
Child IDs include the iteration and query slot, avoiding reuse across
continue-as-new generations. Results merge in input order and papers deduplicate
in sorted ID order.

Both query-child and paper-fetch fan-outs use the SDK's `WhenAll` barrier before
decoding results. It drains every sibling, including when one fails, before
propagating failure; a failed root does not leave its sibling research calls
running. Explicit termination can still interrupt orchestration progress and
cannot undo already-started external calls.

Compared with Python's single selected follow-up query, Go retains up to two
queries and runs their sub-orchestrations concurrently. Fetching retrieves paper
**metadata and abstracts via `id_list`**, not PDF contents, matching the Python
sample's abstract-level analysis scope.

## Prerequisites

- Go 1.25+.
- A running DTS emulator or an existing Azure task hub.
- [Shared Go README](../README.md) for dependency setup, emulator connection, and
  live DTS credentials/roles. This sample does not provision Azure resources.
- Only for optional real mode: arXiv outbound access and an Azure OpenAI
  deployment supporting the v1 Responses API and JSON-object output.

## Bounded fixture demonstration

From `samples/durable-task-sdks/go`:

```sh
go run ./arXiv_research_agent
go test -mod=readonly ./arXiv_research_agent
```

The demo starts a worker and an actual loopback HTTP test server. It starts
research via HTTP and validates:

- `202`, Location, Retry-After, fixture mode headers, and health/status/wait APIs.
- Exactly **2 iterations**, **3 query analyses**, and deduplicated paper IDs
  `fixture-001`, `fixture-002`, `fixture-003`.
- The **entire exact fixture report**, fetched metadata, query order, and analyses.
- Equality between HTTP results and completed DTS output.
- The current execution's `ExecutionStarted` history input contains iteration
  1's exact findings, fetched papers, and two follow-up queries. The history
  read is pinned to `metadata.ExecutionID` and requires exactly one matching
  execution start; this is not a process-memory iteration loop.
- A scheduled job's termination and terminal HTTP status, plus missing-job `404`.

DTS metadata can retain the **original start input** across continue-as-new.
`metadata.ReadInput` is therefore not a current-checkpoint API. The demo uses
execution-pinned history for checkpoint evidence, while the HTTP status API
uses custom status/completed output for current progress and results.

The demo has a 65-second verification deadline plus bounded worker shutdown.

Expected output includes:

```text
Research mode: fixture (fixture papers and reports are synthetic, not academic evidence)
... "iterations": 2, "findings_count": 3 ...
... "paper_ids": ["fixture-001", "fixture-002", "fixture-003"] ...
... "# Fixture research report\n\n> Synthetic fixture only: ..." ...
SAMPLE_OK arXiv_research_agent
```

Only successful assertions and shutdown produce `SAMPLE_OK`. Unit tests are
offline and cover fixture stages, original metadata versus current-execution
checkpoint regression cases, checkpoint/result serialization, Atom parsing,
search/fetch query encoding, rate-limit retries and cancellation, model response
parsing, error propagation, prompt/data separation, and HTTP contracts.

## Interactive API

```sh
go run ./arXiv_research_agent -serve -listen 127.0.0.1:8000 -timeout 10m
curl -i -X POST http://127.0.0.1:8000/agents \
  -H 'Content-Type: application/json' \
  -d '{"topic":"durable workflow reliability","max_iterations":2}'
curl http://127.0.0.1:8000/agents/go-arxiv-REPLACE
curl 'http://127.0.0.1:8000/agents/go-arxiv-REPLACE/wait?timeout=30'
curl -i -X DELETE http://127.0.0.1:8000/agents/go-arxiv-REPLACE
```

Only loopback addresses are accepted. Ctrl+C or `-timeout` shuts down the API and
worker; the shared default timeout is two minutes. This unauthenticated sample
API is not suitable for public exposure.

| Method | Route | Contract |
|---|---|---|
| GET | `/health` | `200`, process liveness and configured mode |
| POST | `/agents` | `202`, `{ok,instance_id,status_url,mode}`, polling headers |
| GET | `/agents/{id}` | `200`, durable runtime status, progress, IDs, completed report |
| GET | `/agents/{id}/wait?timeout=30` | `200` completed result; `408` wait timeout; `500` failed job; `409` terminated/canceled |
| DELETE | `/agents/{id}` | `202` recursive termination requested; `409` already terminal |
| GET | `/agents?continuation_token=...` | Paged `{agents,continuation_token}` from DTS, filtered to Go research roots |

Listing uses the Go SDK's real query API instead of Python's always-empty list.
If the scheduler does not support that capability, the endpoint reports `501`
and directs users to instance lookup/the dashboard; it does not fabricate an
empty result. A page may be empty after filtering child orchestrations; follow
its continuation token.

Request bodies are limited to 4096 bytes, topics to 200 bytes, iterations to 1–10
(default 3), and each iteration to two queries / three papers per query.
`start_delay_seconds` optionally schedules a start 0–30 seconds ahead (used for
deterministic cancellation verification). Invalid ranges return `400` rather
than Python's silent clamping. Unknown JSON fields, invalid content type,
oversized inputs, absent/foreign instances, and backend errors return
`400`/`415`/`413`/`404`/`502` or `504`, respectively.

Client disconnection and `/wait` timeout do **not** cancel a durable job.
DELETE recursively stops orchestration progress; already-running activities may
finish and external model calls cannot be undone. Stopping the worker leaves
unfinished durable jobs resumable by a worker with the same configured mode.

## Optional real arXiv + Azure OpenAI

```sh
export AZURE_OPENAI_ENDPOINT='https://YOUR-RESOURCE.openai.azure.com'
export AZURE_OPENAI_DEPLOYMENT='YOUR-RESPONSES-DEPLOYMENT'
# Optional: set AZURE_OPENAI_API_KEY securely. Otherwise DefaultAzureCredential is used.
go run ./arXiv_research_agent -serve -mode real -timeout 15m
```

Real mode uses:

- The official arXiv Atom API, with per-worker serialized requests spaced at
  least three seconds apart and at most three attempts for `429`/`503`.
  Retry-After is honored within a bounded budget. Query keywords and arXiv
  field/category syntax are URL-encoded. Paper IDs and canonical link hosts are
  validated; arbitrary URLs returned by arXiv are never fetched.
- Azure OpenAI **`/openai/v1/responses`** for analysis, continuation, query
  generation, and synthesis. Fixed instructions are separate from user/paper
  JSON data. Analysis shapes, scores, query counts and output sizes are checked.
  Recognized arXiv citations outside retrieved evidence fail synthesis.
- Context-bounded activities and durable retries. A model/auth/API/parse/budget
  failure fails the activity/job, never silently changes to fixtures or a
  placeholder report. Completed activity outputs are reused on replay; calls
  interrupted before their result is committed can repeat and incur charges.

No PDF downloading, browser UI, or real-paper accuracy verification is claimed.
The real model chooses whether to stop early, so its iterations/results are not
deterministic like fixture output. Human review is required before treating an
LLM summary as academic evidence. Real arXiv/OpenAI calls are **not** claimed as
tested. Real mode requires `-serve`; bounded verification always uses fixtures,
even with a live Azure DTS backend.

Budgets include 60 distinct papers, 20 findings, a 512 KiB checkpoint/model input,
24 KiB model text, 30-second model calls, 45-second arXiv activity calls, and
bounded retries. Metadata/abstract fields are clipped before model use. The
arXiv rate limit is per worker; coordinate an application-wide limiter before
scaling real workers out.

## Environment

| Variable | Default | Purpose |
|---|---|---|
| `DTS_CONNECTION_STRING` | unset | Shared full connection string, takes precedence |
| `ENDPOINT` | `http://localhost:8080` | DTS endpoint |
| `TASKHUB` | `default` | DTS task hub |
| `DTS_AUTHENTICATION` | inferred | `None` for HTTP loopback, otherwise `DefaultAzure` |
| `RESEARCH_MODE` | `fixture` | Default for `-mode`; credentials do not switch modes |
| `ARXIV_API_ENDPOINT` | `https://export.arxiv.org/api/query` | Real mode only; official HTTPS arXiv query endpoint |
| `AZURE_OPENAI_ENDPOINT` | required in real mode | HTTPS Azure resource root; no path/query/userinfo |
| `AZURE_OPENAI_DEPLOYMENT` | required in real mode | Responses-capable deployment |
| `AZURE_OPENAI_API_KEY` | unset | Optional API key; otherwise `DefaultAzureCredential` |
| `AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_CLIENT_SECRET` | unset | Optional standard Azure environment credentials; CLI/managed identity also supported |

Workers use shared automatic task filters and stable Go-specific names. Do not
put confidential material in topics: inputs, retrieved evidence, and reports
are persisted in DTS and, in real mode, sent to the configured model resource.
arXiv is an independent open-access archive, not a Microsoft service.
