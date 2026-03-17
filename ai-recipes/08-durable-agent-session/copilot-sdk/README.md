# Durable Agent Session (Copilot SDK)

This sample demonstrates a **durable Copilot SDK session** whose lifecycle is managed by a Durable Task orchestration.

This recipe currently ships only a `copilot-sdk/` implementation, so this README is the runnable guide for the full sample.

## Durable session pattern

The orchestration receives a list of user messages and processes them one at a time. Each turn is a separate durable activity call:

1. The orchestration derives a deterministic `session_id` from the orchestration instance ID.
2. Each activity tries to `resume_session()` using that ID.
3. If the session does not exist yet, the activity creates it.
4. Durable Task checkpoints the completed turn before moving to the next one.

If the worker crashes between turns, Durable Task replays the orchestration history, rehydrates the completed activity results, and continues from the last unfinished turn. Because the Copilot SDK session uses the same `session_id`, the conversation resumes from its checkpoint instead of starting over.

## Files

- `activities.py` - sends a single message to a Copilot SDK session
- `orchestrations.py` - loops through the conversation durably, one turn at a time
- `worker.py` - hosts the Durable Task worker
- `client.py` - starts a conversation with a list of prompts

## Run locally

Start the Durable Task Scheduler emulator first:

```bash
docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 \
  mcr.microsoft.com/dts/dts-emulator:latest
```

Then install dependencies:

```bash
cd ai-recipes/08-durable-agent-session/copilot-sdk
pip install -r requirements.txt
```

> [!NOTE]
> You also need GitHub Copilot SDK authentication configured in your environment before running this sample.

Start the worker:

```bash
python worker.py
```

In another terminal, start a multi-turn conversation:

```bash
python client.py \
  "What is Durable Task?" \
  "How does durable execution help after a worker restart?" \
  "Give me a code example in Python"
```

## Recovery demo

1. Start the worker and client.
2. Stop the worker after one or more turns complete.
3. Start the worker again.
4. Watch the orchestration continue from the next unfinished turn.

You can inspect execution history and custom status in the dashboard at `http://localhost:8082`.
