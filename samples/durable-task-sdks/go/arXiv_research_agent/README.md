# arXiv research agent (Go)

A research agent that saves its progress with Go workflows.
It searches for papers, reads their details, asks a model to analyze them, and
decides whether to search again. It then writes a report.
A REST API lets you start research, check progress, and read the report.

The default **fixture mode uses made-up sample data**.
Tests on the emulator and live DTS need **no arXiv access or model credentials**.
`fixture-001`, `fixture-002`, and `fixture-003` are not real arXiv IDs.
Reports made from this data are not academic evidence.

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

All network calls happen in activities. Orchestrators use typed checkpoint data
and durable tasks. They do not read environment variables, make HTTP calls,
read the system clock, or start ordinary goroutines.
Child IDs include the iteration and query position so later executions do not
reuse them. Results are combined in input order. Papers are sorted by ID, and
duplicate IDs are removed.

Both groups of parallel tasks use `WhenAll` before reading results.
It waits for every child, even if one fails, before returning an error.
This prevents a failed root from leaving its child research calls running.
Explicit termination can still interrupt a workflow, and it cannot undo
external calls that have already started.

The agent retains up to two follow-up queries and runs their sub-orchestrations
concurrently. Fetching retrieves paper **metadata and abstracts via `id_list`**,
not PDF contents.

## Prerequisites

- Go 1.25+.
- A running DTS emulator or an existing Azure task hub.
- [Shared Go README](../README.md) for dependencies, emulator setup, and Azure
  credentials and roles. This sample does not create Azure resources.
- Only for optional real mode: arXiv outbound access and an Azure OpenAI
  deployment supporting the v1 Responses API and JSON-object output.

## Run one research request

From this sample directory:

```sh
go run .
# Explicitly select the same default:
RESEARCH_MODE=fixture go run . -timeout 2m
```

From the Go module root, use `go run ./arXiv_research_agent`.
The demo starts the worker and a local HTTP server on an available port.
It submits **one** research job using sample data and two iterations.
It waits for the report through the HTTP API, prints it, and shuts down.
Detailed checks run in the tests, not in the demo.
The sample data is in the Go code and does not depend on the current directory.
The default runtime is two minutes. HTTP and worker shutdown have separate
time limits.

Example output includes:

```text
Research mode: fixture (fixture papers and reports are synthetic, not academic evidence)
Research go-arxiv-... started; waiting for its report
... "iterations": 2, "findings_count": 3 ...
... "paper_ids": ["fixture-001", "fixture-002", "fixture-003"] ...
... "# Fixture research report\n\n> Synthetic fixture only: ..." ...
```

## Testing

```sh
go test .
# Requires a configured, running emulator or live DTS task hub:
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

Offline tests check sample-data stages, Atom and model responses, HTTP behavior,
retries, and cancellation. They also check that instructions stay separate from
data, checkpoints can be saved and read, and failed parallel tasks finish.
These checks do not replace tests against real DTS.

When enabled, `TestIntegration` has a two-minute timeout.
It runs the same handlers and workflows as the demo against **real DTS**.
It checks the complete sample report, three paper IDs, two iterations, and three
analyses. It also checks HTTP status, headers, termination, and workflow output.
The test reads history for a specific execution ID. Its `ExecutionStarted`
checkpoint must contain earlier findings, paper details, and follow-up queries.

DTS metadata can retain the **original start input** across continue-as-new.
For this reason, `verification_test.go` reads history for a specific execution.
The HTTP API uses custom status and completed output to show current progress.
Go test output reports these checks separately from demo output.

## Code map / read order

| File | Responsibility |
|---|---|
| `main.go`, `app.go`, `client.go` | Command-line setup, worker/providers, and one-job client |
| `models.go` | Requests, checkpoints, results, and input checks |
| `workflows.go` | Research iterations, history resets, parallel work, and combined results |
| `activities.go` | Activity registration, sample data, and provider calls |
| `arxiv.go`, `model.go` | arXiv and Azure OpenAI clients with input checks |
| `http.go`, `server.go` | Status/report/termination API and loopback server |
| `integration_test.go`, `verification_test.go`, other tests | DTS integration tests and offline tests |

## Interactive API

```sh
go run . -serve -listen 127.0.0.1:8000 -timeout 10m
curl -i -X POST http://127.0.0.1:8000/agents \
  -H 'Content-Type: application/json' \
  -d '{"topic":"durable workflow reliability","max_iterations":2}'
curl http://127.0.0.1:8000/agents/go-arxiv-REPLACE
curl 'http://127.0.0.1:8000/agents/go-arxiv-REPLACE/wait?timeout=30'
curl -i -X DELETE http://127.0.0.1:8000/agents/go-arxiv-REPLACE
```

The server accepts only local addresses. Ctrl+C or `-timeout` stops the API and
worker. The default timeout is two minutes.
The API has no authentication. Do not expose it to the public.

| Method | Route | Contract |
|---|---|---|
| GET | `/health` | `200`, confirms the process is running and shows its mode |
| POST | `/agents` | `202`, `{ok,instance_id,status_url,mode}`, polling headers |
| GET | `/agents/{id}` | `200`, durable runtime status, progress, IDs, completed report |
| GET | `/agents/{id}/wait?timeout=30` | `200` completed result; `408` wait timeout; `500` failed job; `409` terminated/canceled |
| DELETE | `/agents/{id}` | `202` requests a stop for the root and children; `409` if already finished |
| GET | `/agents?continuation_token=...` | Paged `{agents,continuation_token}` from DTS, filtered to Go research roots |

Listing uses the Go SDK's query API.
If the scheduler does not support listing, the endpoint returns `501` and
suggests instance lookup or the dashboard. It does not return a false empty list.
A page may be empty after child workflows are filtered out.
Use its continuation token to request the next page.

Request bodies are limited to 4096 bytes, topics to 200 bytes, iterations to 1–10
(default 3), and each iteration to two queries / three papers per query.
`start_delay_seconds` can schedule a start 0–30 seconds later. Tests use this
delay to check cancellation. Invalid ranges or unknown JSON fields return `400`.
An invalid content type returns `415`, and oversized input returns `413`.
Missing instances or instances from other samples return `404`.
Backend errors return `502` or `504`.

Client disconnection and `/wait` timeout do **not** cancel a durable job.
DELETE stops the root workflow and its children. Activities that are already
running may finish, and model calls cannot be undone.
If the worker stops, another worker with the same mode can resume unfinished jobs.

## Optional real arXiv + Azure OpenAI

```sh
export AZURE_OPENAI_ENDPOINT='https://YOUR-RESOURCE.openai.azure.com'
export AZURE_OPENAI_DEPLOYMENT='YOUR-RESPONSES-DEPLOYMENT'
# Optional: set AZURE_OPENAI_API_KEY securely. Otherwise DefaultAzureCredential is used.
go run . -serve -mode real -timeout 15m
```

Real mode uses:

- The official arXiv Atom API. Each worker sends requests one at a time, at least
  three seconds apart. It makes at most three attempts for `429` or `503`.
  It follows Retry-After within the retry time limit.
  Queries are URL-encoded. Paper IDs and link hosts are checked; the sample does
  not fetch arbitrary URLs returned in a response.
- Azure OpenAI **`/openai/v1/responses`** to analyze papers, decide whether to
  continue, generate queries, and write the report. Fixed instructions are
  separate from user and paper data. The code checks response format, scores,
  query counts, and output sizes. The report fails if it includes a recognized
  arXiv citation outside the retrieved evidence.
- Activities with time limits and durable retries. Model, authentication, API,
  parsing, or budget errors fail the activity or job. They never silently switch
  to sample data or a placeholder report. Saved activity results are reused
  during replay. A call interrupted before its result is saved may repeat and
  cause another charge.

The sample does not download PDFs, provide a browser UI, or check the accuracy
of real papers. A real model decides whether to stop early, so its results and
iteration count can vary. A person must review an LLM summary before using it
as academic evidence. Real arXiv/OpenAI calls are **not** claimed as tested.
Real mode requires `-serve`. The default demo and `TestIntegration` use sample
data even when connected to live Azure DTS.

Limits include 60 different papers, 20 findings, a 512 KiB checkpoint/model input,
24 KiB model text, 30-second model calls, 45-second arXiv activity calls, and
retries with a time limit. Long metadata and abstract fields are shortened
before they are sent to the model. The arXiv rate limit applies to each worker.
Add a shared rate limit before running several real workers.

## Environment

| Variable | Default | Purpose |
|---|---|---|
| `DTS_CONNECTION_STRING` | unset | Full connection string; used instead of the connection settings below |
| `ENDPOINT` | `http://localhost:8080` | DTS endpoint |
| `TASKHUB` | `default` | DTS task hub |
| `DTS_AUTHENTICATION` | chosen automatically | `None` for local HTTP, otherwise `DefaultAzure` |
| `RESEARCH_MODE` | `fixture` | Default for `-mode`; credentials do not switch modes |
| `ARXIV_API_ENDPOINT` | `https://export.arxiv.org/api/query` | Real mode only; official HTTPS arXiv query endpoint |
| `AZURE_OPENAI_ENDPOINT` | required in real mode | HTTPS Azure resource URL without a path, query, or user information |
| `AZURE_OPENAI_DEPLOYMENT` | required in real mode | Deployment that supports the Responses API |
| `AZURE_OPENAI_API_KEY` | unset | Optional API key; otherwise `DefaultAzureCredential` |
| `AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_CLIENT_SECRET` | unset | Optional standard Azure environment credentials; CLI/managed identity also supported |

Workers use shared automatic task filters and stable Go-specific names. Do not
put confidential material in topics. Inputs, retrieved evidence, and reports
are saved in DTS. In real mode, they are also sent to the configured model resource.
arXiv is an independent open-access archive, not a Microsoft service.
