# Multi-Agent Orchestration (`openai-sdk`)

This recipe demonstrates a coordinator plus agent swarm pattern where multiple durable
sub-orchestrations collaborate through a durable entity that holds shared state.

This README focuses on the `openai-sdk/` implementation. See the parent [`../README.md`](../README.md) for the cross-variant overview and the `copilot-sdk/` alternative.

## Architecture

```text
+-------------------+
| coordinator       |
| - assign roles    |
| - launch agents   |
| - synthesize      |
+---------+---------+
          |
  +-------+--------+--------+
  |                |        |
+-v------+     +---v----+ +-v------+
|Agent A |     |Agent B | |Agent C |
|Planner |     |Research| |Critic  |
+---+----+     +---+----+ +---+----+
    \             |          /
     \            |         /
      +-----------v--------+
      | sharedstate entity |
      | findings + status  |
      +--------------------+
```

## Why durable entities are useful here

Durable entities provide:

- Shared context without introducing an external coordination database.
- Durable, replay-safe state updates as agents progress.
- Ordered operations for predictable inter-agent communication.
- Native orchestration integration through `signal_entity` and `call_entity`.

## What this workflow does

1. Take a complex task description.
2. Use an LLM activity to decompose the task into agent assignments.
3. Spawn multiple agent sub-orchestrations in parallel.
4. Let each agent run a small tool-using loop and signal the shared entity with findings.
5. Read the shared entity state and synthesize a final result.

## Running the openai-sdk variant

```bash
cd ai-recipes/07-multi-agent/openai-sdk
pip install -r requirements.txt
# Configure Azure OpenAI credentials (one-time setup)
cp ../../.env.example ../../.env
# Edit ../../.env with your Azure OpenAI API key and endpoint

# Terminal 1
python worker.py

# Terminal 2
python client.py "Plan a durable AI rollout for a regulated enterprise team"
```

Start the Durable Task Scheduler emulator first if you are running locally:

```bash
docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 \
  mcr.microsoft.com/dts/dts-emulator:latest
```

View execution history at `http://localhost:8082`.
