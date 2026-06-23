"""Declarer app for the On-demand Sandboxes code-interpreter demo.

A three-step Durable Task workflow:

    generate_code (in-process, Azure OpenAI -> Python)
        -> execute_code (on-demand sandbox, python3 + pandas, fanned out per region)
        -> format_answer (in-process)

Only ``execute_code`` runs in a DTS-managed sandbox; the LLM-generated Python is
untrusted, so it never executes inside this process.
"""

import os
import sys
import threading

from azure.identity import DefaultAzureCredential, get_bearer_token_provider
from openai import AzureOpenAI

from durabletask import client, task
from durabletask.azuremanaged.client import DurableTaskSchedulerClient
from durabletask.azuremanaged.preview.sandboxes import (
    SandboxActivitiesClient,
    SandboxWorkerProfile,
    sandbox_worker_profile,
)
from durabletask.azuremanaged.worker import DurableTaskSchedulerWorker

from activities import EXECUTE_CODE, FORMAT_ANSWER, GENERATE_CODE


SYSTEM_PROMPT = """\
You are a Python code generator. Given a question about a sales dataset,
produce a single self-contained Python script that:

1. Reads /tmp/data.csv with pandas. The columns are: date, region, product, units, revenue.
2. Assumes the CSV contains rows for exactly one region.
3. Computes the total revenue for March 2025 in this subset.
4. Prints ONLY the numeric revenue total to stdout. No code fences, no explanation, no commentary.

Constraints:
- Use only the Python standard library and pandas.
- Do not access the network or filesystem outside /tmp.
- If there is no March 2025 data in this subset, print 0.
- Output must be plain text containing only the number.

Respond with the Python script only. No markdown, no backticks.
"""


def _require(name: str) -> str:
    value = os.getenv(name)
    if value and value.strip():
        return value.strip()
    raise RuntimeError(f"Set {name} before running the sandbox demo.")


def _strip_code_fences(code: str) -> str:
    trimmed = code.strip()
    if trimmed.startswith("```"):
        newline = trimmed.find("\n")
        if newline > 0:
            trimmed = trimmed[newline + 1:]
        if trimmed.endswith("```"):
            trimmed = trimmed[:-3]
    return trimmed.strip()


def split_csv_by_region(csv_data: str) -> list[tuple[str, str]]:
    """Partition the dataset deterministically so one generated script runs once per region."""
    lines = [line for line in csv_data.replace("\r\n", "\n").split("\n") if line.strip()]
    if len(lines) < 2:
        return []

    header = lines[0]
    rows_by_region: dict[str, list[str]] = {}
    for row in lines[1:]:
        cells = row.split(",")
        if len(cells) < 2:
            continue
        region = cells[1].strip()
        rows_by_region.setdefault(region, []).append(row)

    return [
        (region, "\n".join([header, *rows_by_region[region]]))
        for region in sorted(rows_by_region, key=str.casefold)
    ]


# --- In-process activities (run in this app's worker) -----------------------

def generate_code(ctx: task.ActivityContext, question: str) -> str:
    """Translate a natural-language question into a self-contained pandas script."""
    endpoint = _require("AOAI_ENDPOINT")
    deployment = _require("AOAI_DEPLOYMENT")

    token_provider = get_bearer_token_provider(
        DefaultAzureCredential(), "https://cognitiveservices.azure.com/.default")
    aoai = AzureOpenAI(
        azure_endpoint=endpoint,
        azure_ad_token_provider=token_provider,
        api_version="2024-10-21")

    completion = aoai.chat.completions.create(
        model=deployment,
        messages=[
            {"role": "system", "content": SYSTEM_PROMPT},
            {"role": "user", "content": question},
        ])

    code = _strip_code_fences(completion.choices[0].message.content or "")
    print(f"[generate] AOAI returned {len(code.splitlines())} lines of Python:")
    print("---")
    print(code)
    print("---")
    return code


def format_answer(ctx: task.ActivityContext, payload: dict) -> str:
    """Aggregate the per-region sandbox results into a single answer."""
    question = payload["question"]
    totals: list[tuple[str, float]] = []
    for result in payload["results"]:
        region = result["region"]
        if result["exit_code"] != 0:
            return (f"Sandbox execution failed for region '{region}' "
                    f"(exit code {result['exit_code']}): {result['stderr']}")
        stdout = (result["stdout"] or "").strip()
        try:
            revenue = float(stdout)
        except ValueError:
            return f"Sandbox execution returned a non-numeric result for region '{region}': {stdout}"
        totals.append((region, revenue))

    for region, revenue in sorted(totals, key=lambda item: item[1], reverse=True):
        print(f"[fan-out] Region {region}: {revenue}")

    top_region = sorted(totals, key=lambda item: (-item[1], item[0]))[0][0]
    return f"Q: {question}\nA: {top_region}"


# --- Orchestrator -----------------------------------------------------------

def analyze_sales(ctx: task.OrchestrationContext, payload: dict):
    """3-step workflow answering a question over a CSV using LLM-generated Python."""
    question = payload["question"]
    csv_data = payload["csv"]

    # Generate one chunk-friendly Python script up front and reuse it for every region.
    code = yield ctx.call_activity(GENERATE_CODE, input=question)

    # Fan out: one sandbox execution per region-specific CSV partition.
    chunks = split_csv_by_region(csv_data)
    executions = [
        ctx.call_activity(EXECUTE_CODE.name, input={"code": code, "csv": chunk_csv})
        for _region, chunk_csv in chunks
    ]

    # Fan in: wait for every sandbox result, then hand the set to the formatter.
    results = yield task.when_all(executions)
    region_results = [
        {"region": region, **result}
        for (region, _csv), result in zip(chunks, results)
    ]

    return (yield ctx.call_activity(
        FORMAT_ANSWER,
        input={"question": question, "results": region_results}))


# --- Sandbox worker profile -------------------------------------------------

worker_profile_id = os.getenv("DTS_WORKER_PROFILE_ID", "code-executor")
container_image = os.getenv("DTS_SANDBOX_CONTAINER_IMAGE") or "dts-codegen-sandbox-python:local"
image_pull_managed_identity_client_id = _require("DTS_SANDBOX_IMAGE_PULL_UMI_CLIENT_ID")
scheduler_managed_identity_client_id = _require("DTS_SANDBOX_SCHEDULER_UMI_CLIENT_ID")


@sandbox_worker_profile(worker_profile_id)
class CodeSandboxWorkerProfile(SandboxWorkerProfile):
    """Declares the on-demand sandbox that hosts ``execute_code`` in an isolated container."""

    def configure(self, options) -> None:
        options.image.image_ref = container_image
        options.image.managed_identity_client_id = image_pull_managed_identity_client_id
        options.scheduler_managed_identity_client_id = scheduler_managed_identity_client_id
        options.cpu = "1000m"
        options.memory = "2048Mi"
        options.max_concurrent_activities = 1
        options.add_activity(EXECUTE_CODE.name, version=EXECUTE_CODE.version)


# --- Entry point ------------------------------------------------------------

def main() -> int:
    endpoint = _require("DTS_ENDPOINT")
    taskhub = os.getenv("DTS_TASK_HUB", "default")
    csv_path = os.getenv("DEMO_CSV_PATH") or os.path.join(
        os.path.dirname(os.path.abspath(__file__)), "data", "sales_q1.csv")
    question = " ".join(sys.argv[1:]) if len(sys.argv) > 1 else (
        "Which region had the highest total revenue in March 2025?")

    if not os.path.exists(csv_path):
        print(f"CSV file not found at: {csv_path}", file=sys.stderr)
        return 1

    with open(csv_path, encoding="utf-8") as handle:
        csv_data = handle.read()

    # Print demo context so the audience understands the dataset before orchestration starts.
    all_lines = csv_data.splitlines()
    headers = all_lines[0].split(",")
    print(f"[demo] Dataset: {os.path.abspath(csv_path)}")
    print(f"[demo] {len(all_lines) - 1} rows x {len(headers)} columns: [{', '.join(headers)}]")
    print(f"[demo] Question: {question}\n")

    secure_channel = endpoint.startswith("https://") or endpoint.startswith("grpcs://")
    credential = DefaultAzureCredential() if secure_channel else None

    # Declare the sandbox worker profile with DTS so it can route execute_code to a sandbox.
    sandbox_client = SandboxActivitiesClient(
        host_address=endpoint,
        secure_channel=secure_channel,
        taskhub=taskhub,
        token_credential=credential)
    sandbox_client.enable_sandbox_activities()

    with DurableTaskSchedulerWorker(
            host_address=endpoint,
            secure_channel=secure_channel,
            taskhub=taskhub,
            token_credential=credential) as worker:
        worker.add_orchestrator(analyze_sales)
        worker.add_activity(generate_code)
        worker.add_activity(format_answer)
        worker.use_work_item_filters()
        worker.start()

        durable_client = DurableTaskSchedulerClient(
            host_address=endpoint,
            secure_channel=secure_channel,
            taskhub=taskhub,
            token_credential=credential)
        instance_id = durable_client.schedule_new_orchestration(
            analyze_sales,
            input={"question": question, "csv": csv_data})
        print(f"Started orchestration: {instance_id}")

        state = durable_client.wait_for_orchestration_completion(instance_id, timeout=300)
        print(f"Status: {state.runtime_status if state else 'unknown'}\n")
        if state and state.runtime_status == client.OrchestrationStatus.COMPLETED:
            print(state.serialized_output)
        elif state and state.failure_details:
            print(f"[failure] {state.failure_details}")

        # Schedule exactly once. Don't exit the process: this runs as an always-on Container
        # App, so returning would let Container Apps restart the replica and re-run main()
        # (scheduling a new orchestration every time). Instead, keep the worker running so it
        # stays connected to DTS and ready to serve sandbox work items, and block until the
        # container receives a shutdown signal.
        print("\nOrchestration complete. Worker host staying alive; press Ctrl+C to exit.")
        try:
            threading.Event().wait()
        except KeyboardInterrupt:
            pass
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
