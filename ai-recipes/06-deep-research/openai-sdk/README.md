# Competitive Analysis Agent (`openai-sdk`)

This recipe turns a comparison prompt into a decision-oriented workflow instead of generic open-ended research. The durable coordinator identifies comparison dimensions, fans out focused analysts in parallel, then produces a structured comparison matrix and recommendation.

This README focuses on the `openai-sdk/` implementation. See the parent [`../README.md`](../README.md) for the cross-variant overview and the `copilot-sdk/` alternative.

## How this differs from generic research

- The workflow starts with explicit comparison dimensions instead of broad sub-questions.
- Each analyst owns one dimension only, such as performance, ecosystem, or operational complexity.
- The output is a competitive analysis report with a comparison matrix and pros/cons table, not just a narrative summary.

## Architecture

```text
+---------------------------+
| analysis_coordinator      |
| - identify products       |
| - identify dimensions     |
| - fan out analysts        |
+-------------+-------------+
              |
   +----------+----------+----------+
   |          |          |          |
+--v---+  +---v--+  +----v-+  +-----v----+
|perf  |  |ecosys|  |ops   |  |fit/learn |
|analyst| |analyst| |analyst| |analyst   |
+--+---+  +---+--+  +----+-+  +-----+----+
   |          |          |           |
   +----------+----------+-----------+
              |
              v
   +---------------------------+
   | create_comparison_report  |
   +---------------------------+
```

## What the workflow does

1. Accept a comparison query such as `Compare React vs Vue vs Svelte for enterprise apps`.
2. Use an LLM activity to identify the products being compared and the most useful decision dimensions.
3. Launch one `dimension_analyst` sub-orchestration per dimension.
4. Let each analyst run a focused two-iteration search loop for its dimension.
5. Fan the dimension results back into `create_comparison_report` to generate a structured matrix, pros/cons table, and recommendation.

## Running the openai-sdk variant

```bash
cd ai-recipes/06-deep-research/openai-sdk
python3 -m pip install -r requirements.txt
# Configure Azure OpenAI credentials (one-time setup)
cp ../../.env.example ../../.env
# Edit ../../.env with your Azure OpenAI API key and endpoint

# Terminal 1
python3 worker.py

# Terminal 2
python3 client.py
python3 client.py "Compare React vs Vue vs Svelte for enterprise apps"
```

The client defaults to:

```text
Compare PostgreSQL vs MySQL vs SQLite for a new web application
```

Start the Durable Task Scheduler emulator first if you are running locally:

```bash
docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 \
  mcr.microsoft.com/dts/dts-emulator:latest
```

View execution history at `http://localhost:8082`.

## Files

```text
ai-recipes/06-deep-research/
├── openai-sdk/
│   ├── activities/
│   │   ├── llm_activity.py
│   │   ├── search.py
│   │   └── synthesize.py
│   ├── orchestrations/
│   │   ├── analysis_coordinator.py
│   │   └── dimension_analyst.py
│   ├── client.py
│   ├── requirements.txt
│   └── worker.py
```
