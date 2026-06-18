# Copyright (c) Microsoft Corporation.
# Licensed under the MIT License.

"""Activity identities shared between the declarer app and the sandbox worker."""

from durabletask.azuremanaged.preview.sandboxes import SandboxActivity

# ExecuteCode runs in a DTS-managed on-demand sandbox. Python orchestrations call
# activities by name, so the sandbox activity identity is unversioned.
EXECUTE_CODE = SandboxActivity(name="execute_code", version=None)

# In-process activities that run inside the main app worker.
GENERATE_CODE = "generate_code"
FORMAT_ANSWER = "format_answer"
