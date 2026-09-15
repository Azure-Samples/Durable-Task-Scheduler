# Contributing to Durable Task Scheduler

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.opensource.microsoft.com.

When you submit a pull request, a CLA bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., status check, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

 - [Code of Conduct](#coc)
 - [Issues and Bugs](#issue)
 - [Feature Requests](#feature)
 - [Submission Guidelines](#submit)

## <a name="coc"></a> Code of Conduct
Help us keep this project open and inclusive. Please read and follow our [Code of Conduct](https://opensource.microsoft.com/codeofconduct/).

## <a name="issue"></a> Found an Issue?
If you find a bug in the source code or a mistake in the documentation, you can help us by
[submitting an issue](#submit-issue) to the GitHub Repository. Even better, you can
[submit a Pull Request](#submit-pr) with a fix.

## <a name="feature"></a> Want a Feature?
You can *request* a new feature by [submitting an issue](#submit-issue) to the GitHub
Repository. If you would like to *implement* a new feature, please submit an issue with
a proposal for your work first, to be sure that we can use it.

* **Small Features** can be crafted and directly [submitted as a Pull Request](#submit-pr).

## <a name="submit"></a> Submission Guidelines

### <a name="submit-issue"></a> Submitting an Issue
Before you submit an issue, search the archive, maybe your question was already answered.

If your issue appears to be a bug, and hasn't been reported, open a new issue.
Help us to maximize the effort we can spend fixing issues and adding new
features, by not reporting duplicate issues.  Providing the following information will increase the
chances of your issue being dealt with quickly:

* **Overview of the Issue** - if an error is being thrown a non-minified stack trace helps
* **Version** - what version is affected (e.g. 0.1.2)
* **Motivation for or Use Case** - explain what are you trying to do and why the current behavior is a bug for you
* **Browsers and Operating System** - is this a problem with all browsers?
* **Reproduce the Error** - provide a live example or a unambiguous set of steps
* **Related Issues** - has a similar issue been reported before?
* **Suggest a Fix** - if you can't fix the bug yourself, perhaps you can point to what might be
  causing the problem (line of code or commit)

You can file new issues by providing the above information at the corresponding repository's issues link: https://github.com/Azure/Durable-Task-Scheduler/issues/new.

### <a name="submit-pr"></a> Submitting a Pull Request (PR)
Before you submit your Pull Request (PR) consider the following guidelines:

* Search the repository (https://github.com/Azure/Durable-Task-Scheduler/pulls) for an open or closed PR
  that relates to your submission. You don't want to duplicate effort.

* Make your changes in a new git fork:

* Commit your changes using a descriptive commit message
* Push your fork to GitHub:
* In GitHub, create a pull request
* If we suggest changes then:
  * Make the required updates.
  * Rebase your fork and force push to your GitHub repository (this will update your Pull Request):

    ```shell
    git rebase master -i
    git push -f
    ```

That's it! Thank you for your contribution!

## Contributing Samples

- Each sample should have its own directory under the appropriate framework/language folder.
- Include a `README.md` following a consistent structure: description, prerequisites, how to run, and expected output.
- Use the Durable Task Scheduler emulator as the default development experience.
- Include a `requirements.txt` (Python) or project file (`.csproj`/`.sln` for .NET, `build.gradle` for Java). Go samples share the single `go.mod` and `go.sum` in `samples/durable-task-sdks/go`; do not create nested modules.
- Test your sample with the emulator before submitting.
- Add the sample to the [catalog](./samples/README.md) and relevant [pattern documentation](./docs/patterns.md).

### Go samples

Use Go **1.25.0 or later** and the SDK version pinned in the shared module (currently `github.com/microsoft/durabletask-go` **v1.0.0-beta.1**). Put each runnable sample in its own package under `samples/durable-task-sdks/go`. The default command should start the worker and client, demonstrate the pattern, print a result, shut down, and exit.

Keep `main.go` limited to the entrypoint and CLI wiring. Put orchestrations, activities, worker setup, and client code in focused files within the same package. Keep domain types near the code that uses them; avoid catch-all utility files and unnecessary package layers. A reader should be able to understand the workflow without reading a verification harness.

Put assertions, exhaustive scenarios, and verification-only helpers in `*_test.go`. Each sample must have an opt-in `TestIntegration` in `integration_test.go`, using `testutil.IntegrationContext(t)` to select real-backend tests. Keep operational error handling, input validation, and resource cleanup in production code. README descriptions should stand alone and include a short code map.

Format changed Go files with `gofmt`, then run the same offline checks as CI:

```bash
cd samples/durable-task-sdks/go
go mod download
go build ./...
go test ./...
go vet ./...
```

Ordinary tests must not require an emulator, Azure credentials, or cloud resources. The Go beta SDK has no public in-memory testing backend: test shared business logic offline through a local step adapter, as the testing sample does. Do not claim that these unit tests validate SDK execution or replay.

Keep replay/integration tests against real DTS opt-in with `DTS_SAMPLES_E2E=1`. To run all demonstrations and their integration tests, first prepare an isolated task hub and Blob endpoint as described in the [Go validation guide](./samples/durable-task-sdks/go/README.md#verify-every-sample-on-either-backend). From the Go module, use the sequential runner rather than enabling resource-backed tests across all packages concurrently:

```bash
HISTORY_EXPORT_ISOLATED_TASKHUB=1 DTS_SAMPLES_E2E=1 \
  go test -v -count=1 -timeout 30m ./e2e
```

History-export validation requires a dedicated emulator or Azure task hub with no other export workers and no unrelated workloads completing during the sample's export window. Do not run it in parallel with shared-hub sample validation. For both emulator and Azure runs, set `HISTORY_EXPORT_ISOLATED_TASKHUB=1` only after confirming isolation; the flag is an acknowledgment, not an isolation mechanism.

Use the emulator connection string by default: `Endpoint=http://localhost:8080;TaskHub=default;Authentication=None`. Document any additional prerequisites and read `DTS_CONNECTION_STRING` for Azure connections. Use placeholders, never real resource identifiers or credentials, in committed examples. See the [Go quickstart](./docs/quickstart.md#go) for connection setup.
