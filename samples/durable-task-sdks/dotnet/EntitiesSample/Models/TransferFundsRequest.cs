namespace AccountTransferBackend.Models;

/// <summary>
/// Request to transfer funds between two accounts.
/// </summary>
/// <param name="SourceId">The source account to transfer funds FROM.</param>
/// <param name="DestinationId">The destination account to transfer funds TO.</param>
/// <param name="Amount">The amount of funds to transfer.</param>
public record TransferFundsRequest(string SourceId, string DestinationId, double Amount);
