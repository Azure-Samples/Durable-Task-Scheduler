namespace AccountTransferBackend.Models;

/// <summary>
/// Request to deposit funds into or withdraw funds from an account.
/// </summary>
/// <param name="Amount">The amount of funds to deposit.</param>
public record TransactionRequest(double Amount);