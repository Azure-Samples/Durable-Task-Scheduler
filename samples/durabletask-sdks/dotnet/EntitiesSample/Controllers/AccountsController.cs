using AccountTransferBackend.Entities;
using AccountTransferBackend.Models;
using Microsoft.AspNetCore.Mvc;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Client;
using Microsoft.DurableTask.Client.Entities;
using Microsoft.DurableTask.Entities;

namespace AccountTransferBackend.Controllers;

/// <summary>
/// HTTP APIs for managing accounts. Includes APIs for depositing, withdrawing, and transferring funds.
/// Each API uses the injected DurableTaskClient to interact with entities or orchestrations.
/// </summary>
/// <param name="durableTaskClient">The injected Durable Task client.</param>
/// <param name="logger">A logger to use for logging.</param>
[Route("accounts")]
[ApiController]
public partial class AccountsController(
    DurableTaskClient durableTaskClient,
    ILogger<AccountsController> logger) : ControllerBase
{
    readonly ILogger<AccountsController> logger = logger;
    readonly DurableTaskClient durableTaskClient = durableTaskClient;

    [HttpPost("{accountId}/deposit")]
    public async Task<IActionResult> Deposit(string accountId, [FromBody] TransactionRequest request)
    {
        if (request.Amount <= 0)
        {
            return this.BadRequest("Amount must be greater than zero.");
        }

        this.logger.LogInformation("Depositing {Amount} into account {AccountId}.", request.Amount, accountId);

        EntityInstanceId entityId = new(nameof(Account), accountId);
        await this.durableTaskClient.Entities.SignalEntityAsync(
            id: entityId,
            operationName: nameof(Account.Deposit),
            input: request.Amount);
        return this.Accepted();
    }

    [HttpPost("{accountId}/withdraw")]
    public async Task<IActionResult> Withdraw(string accountId, [FromBody] TransactionRequest request)
    {
        if (request.Amount <= 0)
        {
            return this.BadRequest("Amount must be greater than zero.");
        }

        this.logger.LogInformation("Withdrawing {Amount} from account {AccountId}.", request.Amount, accountId);

        EntityInstanceId entityId = new(nameof(Account), accountId);
        await this.durableTaskClient.Entities.SignalEntityAsync(
            id: entityId,
            operationName: nameof(Account.Withdraw),
            input: request);
        return this.Accepted();
    }

    [HttpGet("{accountId}/balance")]
    public async Task<IActionResult> GetBalance(string accountId)
    {
        EntityInstanceId entityId = new(nameof(Account), accountId);
        EntityMetadata<double>? accountBalance = await this.durableTaskClient.Entities.GetEntityAsync<double>(entityId);
        if (accountBalance is null)
        {
            return this.NotFound();
        }

        return this.Ok(new { accountId, balance = accountBalance.State });
    }

    [HttpPost("transfers")]
    public async Task<IActionResult> Transfer([FromBody] TransferFundsRequest request)
    {
        string instanceId = await this.durableTaskClient.ScheduleNewTransferFundsOrchestrationInstanceAsync(request);
        return this.Accepted(new { transactionId = instanceId });
    }

    [HttpGet("transfers/{transactionId}")]
    public async Task<IActionResult> GetTransferStatus(string transactionId, CancellationToken cancellationToken)
    {
        // We infer the status of the transfer from the status of the orchestration.
        OrchestrationMetadata? metadata = await this.durableTaskClient.GetInstanceAsync(
            transactionId,
            getInputsAndOutputs: true,
            cancellationToken);
        if (metadata is null)
        {
            return this.NotFound();
        }

        string transferResult = "InProgress";
        if (metadata.IsCompleted)
        {
            transferResult = metadata.ReadOutputAs<bool>() ? "Transferred" : "Rejected";
        }

        return this.Ok(new
        {
            transactionId,
            initiatedAt = metadata.CreatedAt.ToString("s"),
            status = metadata.RuntimeStatus.ToString(),
            transferResult,
        });
    }
}
