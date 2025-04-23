using Microsoft.DurableTask.Entities;

namespace AccountTransferBackend.Entities;

class Account : TaskEntity<int>
{
    public void Deposit(int amount) => this.State += amount;

    public void Withdraw(int amount) => this.State -= amount;

    public int GetBalance() => this.State;
}
