using AccountTransferBackend.Entities;
using AccountTransferBackend.Models;
using Microsoft.DurableTask;
using Microsoft.DurableTask.Entities;

namespace AccountTransferBackend.Orchestrations;

/// <summary>
/// Orchestration to transfer funds between two accounts.
/// </summary>
[DurableTask]
class TransferFundsOrchestration : TaskOrchestrator<TransferFundsRequest, bool>
{
    public override async Task<bool> RunAsync(TaskOrchestrationContext context, TransferFundsRequest input)
    {
        TransferFundsRequest? request = context.GetInput<TransferFundsRequest>();
        if (request is null)
        {
            return false;
        }

        // The source and destination accounts are both entities identified by their account ID.
        EntityInstanceId sourceAccount = new(nameof(Account), request.SourceId);
        EntityInstanceId destinationAccount = new(nameof(Account), request.DestinationId);

        // We need to modify both the source and destination accounts in a single transaction.
        // To do this, we create a critical section to that locks these entities and prevents
        // any other caller from changing their state until our locks are released. Only the
        // orchestration holding the lock can modify the state of these entities.
        await using (await context.Entities.LockEntitiesAsync(sourceAccount, destinationAccount))
        {
            ILogger logger = context.CreateReplaySafeLogger<TransferFundsOrchestration>();
            logger.LogInformation(
                "Transfer initiated from {SourceId} to {DestinationId} for {TransferAmount}.",
                request.SourceId,
                request.DestinationId,
                request.Amount);

            // Check the balance of the source account to ensure it has enough funds.
            double sourceBalance = await context.Entities.CallEntityAsync<double>(
                id: sourceAccount,
                operationName: nameof(Account.GetBalance));

            if (sourceBalance >= request.Amount)
            {
                // Withdraw from the source account.
                await context.Entities.CallEntityAsync(
                    id: sourceAccount,
                    operationName: nameof(Account.Withdraw),
                    input: request.Amount);

                // Deposit into the destination account.
                await context.Entities.CallEntityAsync(
                    id: destinationAccount,
                    operationName: nameof(Account.Deposit),
                    input: request.Amount);

                // The transfer succeeded!
                logger.LogInformation(
                    "Transfer completed from {SourceId} to {DestinationId} for {TransferAmount}.",
                    request.SourceId,
                    request.DestinationId,
                    request.Amount);
                return true;
            }
            else
            {
                // The transfer failed due to insufficient funds.
                logger.LogWarning(
                    "Insufficient funds in account {SourceId} to transfer {TransferAmount}.",
                    request.SourceId,
                    request.Amount);
                return false;
            }
        }
    }
}
